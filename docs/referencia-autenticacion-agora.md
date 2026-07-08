# Referencia — Autenticación de Ágora (empresas/proveedores)

> **Estado del documento (2026-07-08):** análisis histórico (2026-06-04/24), antes
> archivo `AUTENTICACION_AGORA.md`. Lo vigente está en
> `specs/system/autenticacion/spec.md`. **Errata verificada 2026-07-07:**
> `utils_oas/security.SetSecurityHeaders()` NO valida JWT (solo pone headers de
> respuesta) — la evidencia de la sección 1 sobre ese middleware quedó corregida
> en la spec.

> **Alcance:** Análisis de `agora_api_crud` (este repo) + proyectos relacionados encontrados localmente
> (`agora_desarrollo`, `sga_cliente`). Solo lectura — ningún archivo fue modificado.
> Fecha del análisis: 2026-06-04.

---

## 1. ¿Existe autenticación de Ágora en este repo?

**RESPUESTA CORTA: No directamente.** `agora_api_crud` es un microservicio CRUD puro; la
autenticación está delegada a una capa upstream.

Sin embargo, la plataforma de autenticación de Ágora **sí existe** y está completamente
identificada a través de los proyectos relacionados en el mismo entorno. Hay **dos sistemas
coexistiendo**:

| Sistema                | Repo fuente                        | Protocolo                  | Usado para                                       |
|------------------------|------------------------------------|----------------------------|--------------------------------------------------|
| **Ágora Legacy (PHP)** | `agora_desarrollo/`                | Custom PHP sessions + AES  | Portal monolítico pre-microservicios (Mayo 2019) |
| **OAuth2/OIDC (WSO2)** | `sga_cliente/` + `agora_api_crud/` | OAuth2 Implicit Flow + JWT | Microservicios modernos (Single-SPA + Angular)   |

**Evidencia de que este repo ya espera tokens OAuth2:**

- `main.go:44` — CORS permite explícitamente el header `authorization`
- `main.go:15` — importa `github.com/udistrital/utils_oas/security` (middleware de validación)
- `main.go:57` — llama a `security.SetSecurityHeaders()` (protección estándar de APIs protegidas)
- `sga_cliente/src/environments/environment.ts:92` — el entorno SGA ya lista `AGORA_SERVICE` apuntando al mismo API
  gateway donde vive este microservicio

---

## 2. Protocolo y estándar

### 2.1 — Sistema moderno (el que aplica a microservicios)

**OAuth2 Implicit Flow + OIDC** sobre **WSO2 Identity Server**.

- Proveedor de identidad: `https://autenticacion.portaloas.udistrital.edu.co`
- Tokens emitidos: `id_token` (JWT con claims de identidad) + `access_token` (JWT Bearer)
- Scopes: `openid email role documento`
- Response type: `id_token token`

> **Nota de seguridad:** El Implicit Flow está marcado como deprecado en OAuth 2.1.
> La recomendación actual es usar **Authorization Code Flow + PKCE**. Si el módulo nuevo
> es una app Angular/SPA, considerar migrar a PKCE desde el inicio.

Evidencia en código:

- `sga_cliente/src/app/@core/utils/implicit_autentication.service.ts` (líneas 48–96): decodifica
  `id_token` del hash de la URL y almacena `access_token` en `localStorage`
- `sga_cliente/src/environments/environment.ts` (líneas 21–28): todos los parámetros OAuth2

### 2.2 — Sistema legacy (solo referencia)

**Custom PHP sessions + cifrado AES.** Documentado en:
`agora_desarrollo/AUTENTICACION_AGORA.md`

Este sistema no expone ninguna API REST. Solo aplica al portal PHP monolítico.

---

## 3. Flujo completo — OAuth2 Implicit Flow

```
┌─────────────────────────────────────────────────────────────────────┐
│ PASO 1 — SPA pide autorización                                      │
│                                                                     │
│  Angular llama a getAuthorizationUrl() y redirige al navegador a:  │
│                                                                     │
│  GET https://autenticacion.portaloas.udistrital.edu.co              │
│          /oauth2/authorize                                          │
│          ?client_id=<CLIENTE_ID>                                    │
│          &redirect_uri=<REDIRECT_URL>                               │
│          &response_type=id_token%20token                            │
│          &scope=openid%20email%20role%20documento                   │
│          &state=<md5_random>                                        │
│          &nonce=<md5_random>                                        │
│          &state_url=<hash_actual_de_la_url>                         │
└────────────────────────────┬────────────────────────────────────────┘
                             │ WSO2 muestra formulario de login
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│ PASO 2 — Usuario ingresa credenciales en WSO2                       │
│                                                                     │
│  El proveedor de identidad (WSO2) valida el usuario.               │
│  Para empresas/proveedores: WSO2 está configurado con un           │
│  user store que conecta a la BD de Ágora o a un directorio LDAP.  │
└────────────────────────────┬────────────────────────────────────────┘
                             │ Redirección al callback
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│ PASO 3 — WSO2 redirige de vuelta al SPA con tokens en el hash      │
│                                                                     │
│  GET <REDIRECT_URL>                                                 │
│       #access_token=<jwt_opaco>                                     │
│       &id_token=<jwt.claims.sig>                                    │
│       &expires_in=<segundos>                                        │
│       &state=<valor_devuelto>                                       │
│       &token_type=Bearer                                            │
└────────────────────────────┬────────────────────────────────────────┘
                             │ ImplicitAutenticationService.init()
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│ PASO 4 — Angular procesa el callback                                │
│  Archivo: implicit_autentication.service.ts (líneas 48–73)         │
│                                                                     │
│  1. Parsea los parámetros del hash de la URL                       │
│  2. Decodifica el payload del id_token: atob(id_token.split('.')[1])│
│  3. Guarda en localStorage:                                         │
│       access_token, expires_in, state, id_token                    │
│  4. Llama updateAuth(payload) → POST al MID de autenticación:      │
│                                                                     │
│     POST https://autenticacion.portaloas.udistrital.edu.co          │
│               /apioas/autenticacion_mid/v1/token/userRol           │
│          Headers: Authorization: Bearer <access_token>              │
│          Body:    { "user": "<email_del_payload>" }                 │
│                                                                     │
│  5. Respuesta: objeto con { role, email, documento, ... }          │
│  6. Guarda en localStorage['user'] = btoa(JSON.stringify(merged))  │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│ PASO 5 — Peticiones autenticadas al backend (agora_api_crud)        │
│  Archivo: auth.Interceptor.ts (líneas 21–50)                       │
│                                                                     │
│  Cada request HTTP clona sus headers y añade:                       │
│    Authorization: Bearer <access_token>                             │
│    Accept: application/json                                         │
│    Content-Type: application/json                                   │
│                                                                     │
│  El backend (Go/Beego) recibe el token y lo valida vía:            │
│    utils_oas/security.SetSecurityHeaders() (main.go:57)            │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│ PASO 6 — Validación del token en el backend                         │
│                                                                     │
│  utils_oas/security verifica el JWT contra la clave pública de     │
│  WSO2 (o via introspección). Si es válido → la petición pasa.      │
│  Si no es válido → 401 Unauthorized.                               │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│ PASO 7 — Logout                                                     │
│  Archivo: implicit_autentication.service.ts (líneas 143–155)       │
│                                                                     │
│  GET https://autenticacion.portaloas.udistrital.edu.co              │
│          /oidc/logout                                               │
│          ?id_token_hint=<id_token>                                  │
│          &post_logout_redirect_uri=<SIGN_OUT_REDIRECT_URL>         │
│          &state=<state>                                             │
│                                                                     │
│  + localStorage.clear() + sessionStorage.clear()                   │
└─────────────────────────────────────────────────────────────────────┘
```

### Autologout automático

`implicit_autentication.service.ts` (líneas 272–300): calcula `expires_at` a partir de
`expires_in`, programa un timer con RxJS `delay()`. 5 minutos antes de expirar muestra
un `SweetAlert2`. Al expirar llama a `logout('logout-auto')`.

---

## 4. Endpoints del servidor de autenticación

Todos en el host `https://autenticacion.portaloas.udistrital.edu.co`:

| Endpoint                                     | Método | Propósito                            | Parámetros clave                                                        |
|----------------------------------------------|--------|--------------------------------------|-------------------------------------------------------------------------|
| `/oauth2/authorize`                          | GET    | Inicia el flujo OAuth2               | `client_id`, `redirect_uri`, `response_type`, `scope`, `state`, `nonce` |
| `/oidc/logout`                               | GET    | Cierra sesión en WSO2                | `id_token_hint`, `post_logout_redirect_uri`, `state`                    |
| `/apioas/autenticacion_mid/v1/token/userRol` | POST   | Obtiene roles/datos del usuario      | Header: `Authorization: Bearer <token>` — Body: `{ "user": "<email>" }` |
| `/apioas/agora_crud/v1/`                     | —      | API gateway hacia este microservicio | Header: `Authorization: Bearer <token>`                                 |

**Evidencia:** `sga_cliente/src/environments/environment.ts` líneas 21–29, 92.

---

## 5. Credenciales y configuración requeridas

### Variables en el frontend Angular

Definidas en `src/environments/environment.*.ts` bajo la clave `TOKEN`:

| Clave                   | Descripción                              | Ejemplo (dev — NO usar en prod)                                      |
|-------------------------|------------------------------------------|----------------------------------------------------------------------|
| `AUTORIZATION_URL`      | Endpoint OAuth2 de WSO2                  | `https://autenticacion.portaloas.udistrital.edu.co/oauth2/authorize` |
| `CLIENTE_ID`            | Client ID registrado en WSO2 para tu app | `[CLIENTE_ID]` (cada módulo usa el suyo)                             |
| `RESPONSE_TYPE`         | Tipo de respuesta OAuth2                 | `id_token token`                                                     |
| `SCOPE`                 | Scopes OIDC solicitados                  | `openid email role documento`                                        |
| `REDIRECT_URL`          | URL de callback post-login               | `http://localhost:4200/` (dev)                                       |
| `SIGN_OUT_URL`          | Endpoint OIDC logout de WSO2             | `https://autenticacion.portaloas.udistrital.edu.co/oidc/logout`      |
| `SIGN_OUT_REDIRECT_URL` | URL de redirección post-logout           | `http://localhost:4200/` (dev)                                       |
| `AUTENTICACION_MID`     | MID para obtener roles del usuario       | `https://.../autenticacion_mid/v1/token/userRol`                     |

> **IMPORTANTE:** Tu módulo de Ágora necesita un **`CLIENTE_ID` propio** registrado en WSO2
> para tu aplicación. El `CLIENTE_ID` de otros módulos NO debe ser reutilizado.
> Solicitar a la OAS el registro de un nuevo cliente OAuth2.

### Variables en el backend Go (agora_api_crud)

Definidas en `conf/app.conf` con valores desde variables de entorno:

| Variable de entorno       | Descripción                      |
|---------------------------|----------------------------------|
| `AGORA_API_CRUD_HTTPPORT` | Puerto HTTP del servicio         |
| `AGORA_API_CRUD_PGUSER`   | Usuario PostgreSQL               |
| `AGORA_API_CRUD_PGPASS`   | Contraseña PostgreSQL            |
| `AGORA_API_CRUD_PGHOST`   | Host PostgreSQL                  |
| `AGORA_API_CRUD_PGPORT`   | Puerto PostgreSQL                |
| `AGORA_API_CRUD_PGDB`     | Nombre de la base de datos       |
| `AGORA_API_CRUD_PGSCHEMA` | Esquema PostgreSQL               |
| `PARAMETER_STORE`         | Endpoint AWS SSM Parameter Store |

> `utils_oas/security` puede requerir variables adicionales para la validación JWT
> (p. ej. la URL del JWKS de WSO2). Verificar la documentación de
> `github.com/udistrital/utils_oas`.

---

## 6. Integración frontend — Angular / Single-SPA

### Cómo lo hace `sga_cliente` (referencia funcional)

**Archivos clave:**

| Archivo                                                 | Rol                                                                    |
|---------------------------------------------------------|------------------------------------------------------------------------|
| `src/app/@core/utils/implicit_autentication.service.ts` | Servicio central: inicia flujo, procesa callback, logout, autologout   |
| `src/app/@core/_Interceptor/auth.Interceptor.ts`        | HTTP Interceptor: añade `Authorization: Bearer <token>` a cada request |
| `src/app/@core/_guards/auth.guard.ts`                   | Route Guard: verifica que el rol del usuario tenga acceso a la ruta    |
| `src/app/app.module.ts`                                 | Registra el interceptor en `HTTP_INTERCEPTORS`                         |
| `src/environments/environment.ts`                       | Variables de configuración (URLs, client_id, scopes)                   |

**Flujo de integración en el módulo Angular:**

```typescript
// 1. En app.module.ts — registrar el interceptor
providers: [
    {provide: HTTP_INTERCEPTORS, useClass: AuthInterceptor, multi: true}
]

// 2. En el componente raíz o app.component.ts — verificar sesión
constructor(private
authService: ImplicitAutenticationService
)
{
    if (!authService.login(false)) {
        // Redirige automáticamente a WSO2 para login
    }
}

// 3. El interceptor añade el token a TODOS los requests automáticamente
// No hace falta modificar cada llamada HTTP individualmente
```

---

## 7. Guía de implementación en un módulo nuevo

### Prerequisitos

1. **Registrar un nuevo cliente OAuth2 en WSO2** (Ágora / OAS):
    - Tipo: `Public` (SPA sin secret)
    - Grant type: `Implicit` (o `Authorization Code + PKCE` si se moderniza)
    - Callback URLs: URLs de tu app (dev + prod)
    - Scopes: `openid email role documento`
    - Recibirás un `CLIENTE_ID` propio

2. **Asegurarse de que los usuarios de empresa existan en el user store de WSO2** que
   corresponde a Ágora. Este es el punto de integración entre WSO2 y la BD de usuarios
   de Ágora (legacy: `framework.usuario`).

### Paso a paso — Frontend Angular

```
1. Copiar implicit_autentication.service.ts de sga_cliente a tu módulo Angular
   (o instalarlo como librería npm si la OAS lo publica)

2. Copiar auth.Interceptor.ts

3. Copiar auth.guard.ts

4. Crear src/environments/environment.ts con la sección TOKEN:
   TOKEN: {
     AUTORIZATION_URL: 'https://autenticacion.portaloas.udistrital.edu.co/oauth2/authorize',
     CLIENTE_ID: '<TU_CLIENTE_ID_PARA_AGORA>',   // ← Pedirlo a la OAS
     RESPONSE_TYPE: 'id_token token',
     SCOPE: 'openid email role documento',
     REDIRECT_URL: '<URL_DE_TU_APP>/',
     SIGN_OUT_URL: 'https://autenticacion.portaloas.udistrital.edu.co/oidc/logout',
     SIGN_OUT_REDIRECT_URL: '<URL_DE_TU_APP>/',
     AUTENTICACION_MID: 'https://autenticacion.portaloas.udistrital.edu.co/apioas/autenticacion_mid/v1/token/userRol',
   }

5. En app.module.ts:
   - Importar HttpClientModule
   - Registrar AuthInterceptor en HTTP_INTERCEPTORS

6. En el componente raíz: inyectar ImplicitAutenticationService y llamar login(false)
   en el constructor

7. En rutas protegidas: añadir canActivate: [AuthGuard]
```

### Paso a paso — Backend Go (nuevo microservicio estilo agora_api_crud)

```
1. En go.mod: agregar dependencia github.com/udistrital/utils_oas

2. En main.go: copiar el mismo bloque de middlewares:
   - cors.Allow(...) con "authorization" en AllowHeaders
   - xray.InitXRay()
   - apistatus.Init()
   - auditoria.InitMiddleware()
   - security.SetSecurityHeaders()   ← Este valida el JWT

3. En conf/app.conf: usar variables de entorno para todas las credenciales

4. Si se necesita validación de roles a nivel de ruta, investigar si utils_oas/security
   expone un filtro por rol o si hay que implementarlo manualmente en cada controller
```

---

## 8. Diferencia: autenticación Ágora (empresas) vs. SGA (egresados)

| Aspecto                    | Ágora — Empresas                                     | SGA — Egresados                                        |
|----------------------------|------------------------------------------------------|--------------------------------------------------------|
| **Proveedor de identidad** | WSO2 (mismo servidor)                                | WSO2 (mismo servidor)                                  |
| **Diferencia**             | `CLIENTE_ID` diferente + user store de Ágora en WSO2 | `CLIENTE_ID` del SGA + user store del SGA              |
| **Protocolo**              | OAuth2 Implicit / OIDC                               | OAuth2 Implicit / OIDC                                 |
| **Frontend**               | Angular + `ImplicitAutenticationService` (igual)     | Angular + `ImplicitAutenticationService` (misma clase) |
| **Backend**                | Go/Beego + `utils_oas/security`                      | Go/Beego + `utils_oas/security` (mismo patrón)         |
| **Evidencia**              | `environment.ts:92` — `AGORA_SERVICE`                | `environment.ts:21–28` — todo el bloque TOKEN          |

> **Conclusión clave:** La diferencia entre autenticar empresas (Ágora) y egresados (SGA) es
> **solo el `CLIENTE_ID` y el user store configurado en WSO2**, no el protocolo ni el código.

---

## 9. Dudas abiertas y piezas faltantes para end-to-end

| # | Duda                                                                                                                                                                                   | Impacto                   | Cómo resolverla                                                          |
|---|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------|--------------------------------------------------------------------------|
| 1 | **¿Los usuarios de empresa (proveedores) ya están migrados al user store de WSO2?** Si aún solo existen en la BD PHP legacy (`framework.usuario`), el login OAuth2 fallará para ellos. | **CRÍTICO**               | Preguntar a la OAS si el user store de Ágora está configurado en WSO2    |
| 2 | **`CLIENTE_ID` para el módulo de Ágora** — no existe un client_id propio para nuevos módulos de empresas.                                                                              | **CRÍTICO**               | Registrar nuevo cliente OAuth2 en el WSO2 de la UD (OAS)                 |
| 3 | **¿Qué valida exactamente `utils_oas/security`?** — el código fuente no está en el módulo cache local. Puede validar firma JWT, expiración, scopes, o hacer introspección.             | Alto                      | Revisar el repo `github.com/udistrital/utils_oas` en GitHub              |
| 4 | **Roles de empresa en WSO2** — el scope `role` devuelve roles del token. ¿Existen roles específicos para empresas/proveedores en WSO2?                                                 | Alto                      | Verificar con la OAS qué roles están configurados para usuarios de Ágora |
| 5 | **`AUTENTICACION_MID`** — el endpoint `autenticacion_mid/v1/token/userRol` devuelve datos del usuario. ¿Funciona para usuarios de empresa o solo para usuarios académicos?             | Alto                      | Probar con un usuario de empresa de prueba                               |
| 6 | **URL del API gateway para el módulo nuevo** — `agora_api_crud` está expuesto en `/apioas/agora_crud/v1/`. Un módulo nuevo tendrá su propia ruta.                                      | Medio                     | Definir con la OAS la ruta del nuevo servicio en el API gateway          |
| 7 | **Implicit Flow vs Authorization Code + PKCE** — si el nuevo módulo es Angular, considerar migrar a PKCE (más seguro). WSO2 moderno lo soporta.                                        | Bajo (decisión de diseño) | Propuesta de arquitectura al equipo                                      |

---

## 10. Resumen ejecutivo

```
PROTOCOLO:        OAuth2 Implicit Flow + OIDC
PROVEEDOR:        WSO2 Identity Server en autenticacion.portaloas.udistrital.edu.co
TOKENS:           id_token (JWT con claims) + access_token (JWT Bearer)
SCOPES:           openid email role documento
FLUJO:            SPA → /oauth2/authorize → WSO2 login → redirect con tokens en hash URL
FRONTEND:         ImplicitAutenticationService (Angular) — disponible en sga_cliente/
INTERCEPTOR:      AuthInterceptor añade "Authorization: Bearer <token>" automáticamente
GUARD:            AuthGuard verifica permisos de ruta por rol
BACKEND:          utils_oas/security valida el token en cada request (main.go)
DIFERENCIA SGA:   Solo el CLIENTE_ID y el user store de WSO2 son distintos
PASOS FALTANTES:  (1) Registrar CLIENTE_ID en WSO2, (2) confirmar migración de usuarios
                  de empresa al user store de WSO2
```

---

## 11. Archivos de referencia por ruta exacta

| Archivo                                                                | Relevancia                                                       |
|------------------------------------------------------------------------|------------------------------------------------------------------|
| `main.go:37–48`                                                        | CORS que permite el header `Authorization`                       |
| `main.go:57`                                                           | `security.SetSecurityHeaders()` — validación JWT en backend      |
| `main.go:55`                                                           | `auditoria.InitMiddleware()` — auditoría de requests             |
| `conf/app.conf`                                                        | Variables de entorno del backend                                 |
| `go.mod:8`                                                             | Dependencia `utils_oas` (contiene el middleware de seguridad)    |
| `../sga_cliente/src/app/@core/utils/implicit_autentication.service.ts` | **Implementación completa del flujo OAuth2** — copiar/adaptar    |
| `../sga_cliente/src/app/@core/_Interceptor/auth.Interceptor.ts`        | HTTP Interceptor con Bearer token                                |
| `../sga_cliente/src/app/@core/_guards/auth.guard.ts`                   | Route guard por roles                                            |
| `../sga_cliente/src/environments/environment.ts:20–29`                 | Parámetros OAuth2 (URLs, client_id, scopes)                      |
| `../sga_cliente/src/environments/environment.ts:92`                    | `AGORA_SERVICE` — URL del servicio Ágora en el gateway           |
| `../agora_desarrollo/AUTENTICACION_AGORA.md`                           | Documentación del sistema legacy PHP (solo referencia histórica) |
