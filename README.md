# sga_mid_beneficios_egresados

API MID (lógica de negocio) del submódulo **Beneficios para Egresados** del Sistema
de Gestión Académica (SGA) de la Universidad Distrital Francisco José de Caldas.
Orquesta el CRUD del módulo y los servicios institucionales (parámetros, Ágora,
autenticación) y aplica las reglas de negocio.

Capas hermanas:
[`sga_crud_beneficios_egresados`](https://github.com/DanielVelandia2407/sga_crud_beneficios_egresados)
(persistencia) y
[`sga_cliente_beneficios_egresados_mf`](https://github.com/DanielVelandia2407/sga_cliente_beneficios_egresados_mf)
(micro-frontend).

## Especificaciones técnicas

- **Go** 1.22 · **Beego** v2.2
- Sin acceso directo a base de datos: todo pasa por el CRUD del módulo o por
  servicios institucionales vía HTTP

### Responsabilidades clave

- **Catálogos (C-1):** `services/parametros_service.go` centraliza el acceso al
  servicio institucional de parámetros (`GetParametrosPorTipo`, `ResolverParametroId`,
  `ResolverParametroCodigo`). Tipos usados: `ESTADO_EMPRESA`, `ESTADO_BENEFICIO`,
  `ESTADO_SOLICITUD`, `CATEGORIA_BENEFICIO`, `SECTOR_ECONOMICO`, `PARAMETRO_SISTEMA`
  (creados en el servicio institucional el 2026-07-07, área EGR).
- **Estado de solicitudes (C-4b):** el estado vigente se deriva del historial
  (`GET /historial_solicitud/solicitud/:id/vigente` del CRUD); la máquina de estados
  (RN-005) valida transiciones y cada cambio es un INSERT en el historial.
- **Radicados:** los genera la base de datos al insertar la solicitud
  (`fn_siguiente_radicado()`, C-5); el MID no envía radicado y lo devuelve leído
  de la solicitud creada. Formato `BNF-YYYY-NNNNNN`.
- **Catálogo de beneficios (RN-008):** solo PUBLICADO, `fecha_fin >= hoy`,
  `cupos_disponibles > 0`; filtros por categoría/empresa y búsqueda por título,
  todo delegado al CRUD vía `query=`.

## Variables de entorno

| Variable | Default | Descripción |
|---|---|---|
| `BENEFICIOS_EGRESADOS_MID_CRUD_URL` | `http://localhost:8080/v1` | URL del CRUD del módulo |
| `BENEFICIOS_EGRESADOS_MID_PARAMETROS_URL` | `https://autenticacion.portaloas.udistrital.edu.co/apioas/parametros/v1` | Servicio institucional de parámetros |
| `BENEFICIOS_EGRESADOS_MID_AUTENTICACION_URL` | `https://autenticacion.portaloas.udistrital.edu.co/apioas/autenticacion_mid/v1` | autenticacion_mid (userRol) |
| `BENEFICIOS_EGRESADOS_MID_AMAZON_URL` | `https://autenticacion.portaloas.udistrital.edu.co/apioas/administrativa_amazon_api/v1` | Datos de proveedor/empresa (C-2b) |
| `BENEFICIOS_EGRESADOS_MID_GESTOR_DOCUMENTAL_URL` | `https://autenticacion.portaloas.udistrital.edu.co/apioas/gestor_documental_mid/v1` | Gestor documental institucional (Nuxeo): subir/consultar/eliminar los PDFs de solicitudes. El cliente Angular nunca lo llama directo, solo el MID (`IdTipoDocumento=167` fijo) |
| `BENEFICIOS_EGRESADOS_MID_PARAMETROS_LOCAL` | `false` | **Solo dev/offline**: si `true`, resuelve los parámetros desde un catálogo EN MEMORIA (`parametros_service.go`), sin token ni servicio institucional. El seed local usa los MISMOS ids institucionales (7199+), así que modo local y real son intercambiables. |
| `BENEFICIOS_EGRESADOS_MID_PORT` | `8080` | Puerto HTTP (en desarrollo local se usa `8081` para no chocar con el CRUD) |
| `BENEFICIOS_EGRESADOS_MID_RUNMODE` | `dev` | Modo de ejecución de Beego |

## Ejecución

```bash
# requiere el CRUD corriendo (ver su README)
export BENEFICIOS_EGRESADOS_MID_PORT=8081
go run .
```

## Endpoints (`/v1`)

```
# Egresado
GET  /beneficios                              catálogo (page, limit, categoria_id, empresa_id, q)
GET  /beneficios/:id                          detalle (+ total_solicitudes)
POST /egresados/provision                     JIT provisioning al login (identidad del token, sin body)
POST /solicitudes                             crear solicitud (radicado + estado PENDIENTE)
GET  /solicitudes/egresado/:egresado_id       mis solicitudes (con estado vigente)
GET  /solicitudes/egresado/:egresado_id/resumen
GET  /solicitudes/:id/historial               bitácora de estados (C-4b)
PUT  /solicitudes/:id/cancelar                RN-005: solo estados en curso; devuelve cupo

# Empresa
POST /empresas/provision                      JIT provisioning al login (identidad del token, sin body)
GET  /empresas/:id                            perfil público (whitelist RNF-002b + métricas)
GET  /usuarios/:usuario_id/empresas           selector multiempresa (caso 1:N)
GET  /empresas/:empresa_id/solicitudes        bandeja (datos mínimos del egresado, RNF-002b)
POST /empresas/:empresa_id/beneficios         publicar beneficio (RN-008b, empresa ACTIVA)
GET  /empresas/:empresa_id/beneficios         gestión del dueño (todos los estados + métricas)
PUT  /beneficios/:id                          editar beneficio (RN-008b de edición)
PUT  /beneficios/:id/retirar                  retirar beneficio (→ RETIRADO)
PUT  /solicitudes/:id/responder               aprobar / rechazar / requerir info
POST /solicitudes/:id/mensajes                mensajes (REQUIERE_INFO / EN_REVISION)
GET  /solicitudes/:id/mensajes

# Admin / catálogos
PUT  /empresas/:id/suspender
GET  /categorias-beneficio                    proxy del servicio de parámetros
GET  /sectores-economicos                     proxy del servicio de parámetros

# Documentos requeridos / subidos (gestor_documental_mid, IdTipoDocumento=167 fijo)
GET    /beneficios/:id/documentos-requeridos       documentos que la empresa exige (definidos al publicar)
GET    /solicitudes/:id/documentos                 requeridos vs. subidos (egresado y empresa)
POST   /solicitudes/:id/documentos                 egresado sube/reemplaza un PDF (solo solicitud en curso)
DELETE /solicitudes/:id/documentos/:doc_id         egresado quita un documento (solo solicitud en curso)
PUT    /documentos/:doc_id/comentario              empresa comenta un documento (campo único, se sobreescribe)
GET    /documentos/:doc_id/archivo                 ver/descargar (proxy de solo lectura al gestor documental)
GET    /solicitudes/:id/comprobante                comprobante OPCIONAL adjuntado por la empresa al aprobar
```

### Documentos requeridos por beneficio

Al publicar un beneficio (`POST /empresas/:empresa_id/beneficios`), la empresa puede incluir
`documentos_requeridos: [{titulo, descripcion}]` para pedirle al egresado soportes (hoja de vida,
cédula, etc.) al postularse. El detalle del beneficio (`GET /beneficios/:id`) los devuelve en
`documentos_requeridos`. Solo se definen al publicar (no hay edición posterior en este alcance).

El PDF en sí nunca pasa por este MID hacia el cliente en texto plano más de lo necesario ni se
guarda aquí: viaje `POST/GET/DELETE .../documentos` sube/consulta/elimina contra el servicio
institucional `gestor_documental_mid` (Nuxeo) con `IdTipoDocumento=167` fijo; solo el uid/`Enlace`
que ese servicio devuelve se guarda en el CRUD (`documento_solicitud.enlace_gestor_documental`).
Subir/reemplazar/eliminar documentos solo está permitido mientras la solicitud sigue en curso
(PENDIENTE, REQUIERE_INFO o EN_REVISION — mismo criterio que cancelar, RN-005).

### Comprobante de aprobación (opcional)

Al aprobar una solicitud (`PUT /solicitudes/:id/responder` con `estado_nuevo: "APROBADA"`), la
empresa puede adjuntar opcionalmente un PDF de comprobante (`body.comprobante: { nombre_archivo,
file (base64) }`). Se sube al gestor documental (mismo `IdTipoDocumento=167`) **antes** de registrar
el cambio de estado; si el comprobante falla al subirse, la aprobación se aborta (no se aprueba sin
él si la empresa quiso adjuntarlo). El uid/`Enlace` queda en el propio registro de
`historial_solicitud` de esa transición (`enlace_comprobante`/`nombre_archivo_comprobante`), no en
`documento_solicitud`. El egresado lo consulta con `GET /solicitudes/:id/comprobante`. Adjuntar un
comprobante en cualquier transición que no sea APROBADA es un error (400).

## Estado y seguridad

- Reglas de negocio del núcleo implementadas: RN-002b/c (cupos atómicos con
  compensación), RN-003, RN-004/RN-005 (máquina de estados sobre historial),
  RN-007, RN-008/RN-008b, RN-010.
- JIT provisioning de **ambos perfiles** (`POST /egresados/provision` y
  `POST /empresas/provision`); la identidad SIEMPRE se deriva del token
  (OIDC userinfo), nunca del body.
- Validación del token entrante en `/v1/*` (`middleware/jwt.go`): JWT por firma
  RS256 contra el JWKS de WSO2; tokens opacos contra userinfo (401 si inválido).
- Autorización por recurso (`services/autorizacion_service.go`): cada operación
  verifica el vínculo del usuario del token con el recurso (403 si es ajeno).
- Pendientes: ver `specs/logica-negocio/tasks.md`.

## Documentación (SDD)

- `specs/system/` — especificaciones **transversales a los tres repos**:
  visión general, autenticación/identidad y catálogos institucionales. Este
  repo es su dueño; los otros dos enlazan hacia aquí.
- `specs/logica-negocio/` — spec, plan y tareas de esta API.
- `docs/` — referencias de apoyo (Ágora, parámetros, servicios del ecosistema).

## Contexto

Desarrollado en el marco de la pasantía de Ingeniería de Sistemas (2026) para la
Oficina Asesora de Sistemas (OAS) / OATI. Lineamientos: APIs separadas CRUD/MID,
plantillas `udistrital/plantilla_api_mid`, autenticación OAuth2/OIDC sobre WSO2.
