# Spec — Autenticación, identidad y autorización

> **Última actualización:** 2026-07-08 · **Estado:** implementado y probado e2e
> (login ambos perfiles 2026-07-02; JWT entrante + anti-IDOR 2026-07-07).
> Referencias de fondo: `docs/referencia-autenticacion-agora.md` (este repo) y
> `docs/referencia-autenticacion-wso2.md` (repo `egresados_cliente`).

## Objetivo

Definir cómo se autentican egresados y empresas, cómo se deriva su identidad
local (JIT provisioning) y cómo se autoriza cada operación, sin gestionar
credenciales propias: todo delega en WSO2 Identity Server institucional.

## Alcance

**In scope:** flujo OIDC del frontend, ramificación egresado/empresa post-login,
JIT provisioning de ambos perfiles, validación del token entrante en el MID,
propagación del Bearer saliente, autorización por recurso (anti-IDOR).
**Out of scope:** creación de usuarios en WSO2 (auto-registro institucional),
gestión de roles/menús del SGA (D-8, trámite OATI).

## Repos involucrados

- `sga_cliente_beneficios_egresados_mf` — login OIDC implicit, sesión, guards.
- `egresados_service` — JIT, validación JWT, autorización, propagación.
- (El CRUD no valida auth; recibe el token de forma uniforme para un futuro filtro.)

## Requisitos funcionales

1. **Login único** para ambos perfiles: OAuth2 Implicit Flow + OIDC contra WSO2 (`autenticacion.portaloas.udistrital.edu.co`), scopes `openid email role documento`, mismo `CLIENTE_ID`. No hay user store ni flujo distinto para empresas (D-5 confirmado empíricamente).
2. **Ramificación post-login por `Estado` de `userRol`:** `"E"` = egresado → `/catalogo`; distinto de `"E"` (incluye vacío) = empresa → `/empresa/dashboard`. Regla defensiva: egresado SOLO si `Estado == "E"`.
3. **JIT de egresado** (`POST /v1/egresados/provision`, sin body): userinfo(token) → userRol → documento (userinfo, fallback userRol) → nombre real vía `terceros_crud` (best-effort) → código institucional (`userRol.Codigo`, fallback `consultar_persona`) → alta idempotente de `usuario` (EGR) + `egresado`.
4. **JIT de empresa** (`POST /v1/empresas/provision`, sin body): userinfo(token) → userRol (rechaza egresados) → `informacion_proveedor?query=correo:{email}` → por cada proveedor, amarre de identidad y alta idempotente de `usuario` (EMP) + `empresa` + `usuario_empresa`. Multiempresa: itera todos los proveedores del correo; devuelve la lista completa (incluye `nit` por empresa).
5. **La identidad SIEMPRE se deriva del token** (endpoint `oauth2/userinfo` → `{sub, email, documento?}`), nunca del body: un usuario autenticado no puede provisionar la identidad de otro correo.
6. **Validación del token entrante en el MID** (`middleware/jwt.go`, filtro BeforeRouter en `/v1/*`): JWT → firma RS256 contra el JWKS de WSO2 + exp/nbf (JWKS cacheado, recarga por rotación de kid máx. 1/min, SOLO RS256); token opaco → userinfo con caché positiva de 5 min. Sin token o inválido → 401 con envelope OATI. Toggle solo-dev `EGRESADOS_SERVICE_VALIDAR_JWT=false`.
7. **Propagación saliente:** el MID propaga el Bearer del request entrante a TODAS las llamadas salientes (CRUD, parámetros, terceros, Ágora, gestor documental). El token se threadea controller→service→helper (request-scoped, sin variable global). Motivo: todo el gateway exige Bearer y el token de usuario expira ~1h (no sirve como token estático de servicio).
8. **Autorización por recurso (anti-IDOR)** (`services/autorizacion_service.go`): identidad del token (userinfo → `sub` → `usuario` local por `id_externo`) y verificación de vínculo antes de operar. Familias: empresa (bandeja, mis-beneficios, publicar/editar/retirar, responder, comentar), egresado (mis-solicitudes, resumen, crear, cancelar, subir/eliminar documento) y bidireccionales (mensajes, documentos, comprobante = egresado dueño O usuario de la empresa del beneficio). Acceso denegado → 403.

## Amarre de identidad (reglas de seguridad del JIT de empresa)

- Proveedor **NATURAL**: exige `token.documento == NumDocumento` **cuando el documento existe** en el token.
- Proveedor **JURIDICA** (y usuarios sin documento): la barrera es la coincidencia de correo (derivado del token) + la operación humana del ciclo de vida.
- Los usuarios de empresa self-signup **no tienen documento en ninguna fuente** (userinfo ni userRol): la identidad local se keya por `(sistema_origen, id_externo = sub WSO2)` (UNIQUE `uq_usuario_id_externo`); `usuario.documento` es NULLABLE.

## Contrato de integración

### Servicios institucionales consumidos

| Servicio | Endpoint | Uso |
|---|---|---|
| WSO2 OIDC | `GET /oauth2/userinfo` (Bearer) | Identidad del dueño del token: `{sub, email, documento?}` |
| WSO2 OIDC | `GET /oauth2/jwks` | Llaves públicas para validar JWT entrante |
| autenticacion_mid | `POST /v1/token/userRol` (Bearer, body `{user: email}`) | `{role[], documento, Estado, Codigo, email}` — `Estado` decide el perfil |
| administrativa_amazon_api | `GET /v1/informacion_proveedor?query=correo:{email}` (Bearer) | Datos de proveedor Ágora (respuesta = **array**, correo 1:N) |

### Endpoints propios (MID → frontend)

| Endpoint | Respuesta |
|---|---|
| `POST /v1/egresados/provision` | `{usuario_id, egresado_id, codigo_institucional, nombre}` |
| `POST /v1/empresas/provision` | `{usuario_id, empresas: [{empresa_id, usuario_empresa_id, nit, proveedor: ProveedorPublico}]}` |
| `GET /v1/usuarios/:usuario_id/empresas` | Lista local para el selector (sin re-pegar a Ágora) |

`ProveedorPublico` es una **whitelist**: los campos bancarios y anexos de Ágora
no se declaran en el struct de deserialización, por lo que nunca entran al MID
ni salen por él (minimización RNF-002b).

## Criterios de aceptación

1. Login de egresado real → sesión con nombre, código y programa; catálogo accesible.
2. Login de empresa real → vinculación automática a su(s) empresa(s) de Ágora; bandeja con datos reales (verificado con un usuario de empresa real el 2026-07-02).
3. Relogin no duplica registros (JIT idempotente).
4. Request al MID sin token, con JWT corrupto o con opaco inválido → 401 (no 500).
5. Operar un recurso ajeno (solicitud/empresa/beneficio de otro) → 403.
6. Ningún response del MID contiene `NumCuentaBancaria`, `Anexorut` ni equivalentes.

## Casos borde

- **Token expira (~1h) a mitad de sesión:** la sesión termina; no hay refresh token (limitación del implicit flow institucional).
- **`prompt=login` NO es soportado** por el WSO2 de la UD (callback sin id_token). Cambio de cuenta = logout del SSO.
- **userRol de empresa responde 200 con todo vacío** (`Estado:""`, `role:[]`): se trata como empresa (regla defensiva del punto 2).
- **`usuario` local sin `id_externo`** (creado por flujos previos al JIT): recibirá 403 del autorizador; requiere backfill o re-provisión.
- **Correo institucionalmente 1:N con proveedores**: todas las empresas quedan vinculadas; `es_principal=true` en la primera.

## Pendientes (dependen de OATI)

- **D-6:** `CLIENTE_ID` de producción (el actual `developer_4200` es solo dev, callback `localhost:4200`).
- **D-8:** roles EGRESADO/EMPRESA en WSO2 + menú del rol en el servicio de configuración; hasta entonces los guards del front validan solo "sesión autenticada + perfil".
- **Pregunta de seguridad:** confirmar que WSO2 verifica la propiedad del correo en el auto-registro (única llave de identidad de empresa).
- **Gestión con OATI (canal interno):** validación del alcance de autorización y minimización de datos en los servicios institucionales que este flujo consume.
- **Verificación menor:** falta observar con token vivo qué rama del middleware usan los access_token reales de WSO2 (¿JWT o opacos?).
