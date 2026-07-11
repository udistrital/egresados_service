# egresados_service

API MID (lógica de negocio) del submódulo **Beneficios para Egresados** del Sistema
de Gestión Académica (SGA) de la Universidad Distrital Francisco José de Caldas.
Orquesta el CRUD del módulo y los servicios institucionales (parámetros, Ágora,
autenticación) y aplica las reglas de negocio.

Capas hermanas:
[`egresados_crud`](https://github.com/udistrital/egresados_crud)
(persistencia) y
[`egresados_cliente`](https://github.com/udistrital/egresados_cliente)
(micro-frontend).

## Especificaciones Técnicas

### Tecnologías Implementadas y Versiones
* Golang 1.22 (imagen de CI en `golang:1.25`, compatible hacia atrás)
* BeeGo v2.2 (`server/web`, sin acceso directo a base de datos: todo pasa por el
  CRUD del módulo o por servicios institucionales vía HTTP)

### Responsabilidades clave

- **Catálogos (C-1):** `services/parametros_service.go` centraliza el acceso al
  servicio institucional de parámetros (`GetParametrosPorTipo`, `ResolverParametroId`,
  `ResolverParametroCodigo`). Tipos usados: `ESTADO_EMPRESA`, `ESTADO_BENEFICIO`,
  `ESTADO_SOLICITUD`, `CATEGORIA_BENEFICIO`, `SECTOR_ECONOMICO`, `PARAMETRO_SISTEMA`
  (creados en el servicio institucional el 2026-07-07, área EGR).
- **Estado de solicitudes (C-4b):** el estado vigente se deriva del historial
  (`GET /historial-solicitud/solicitud/:id/vigente` del CRUD); la máquina de estados
  (RN-005) valida transiciones y cada cambio es un INSERT en el historial.
- **Radicados:** los genera la base de datos al insertar la solicitud
  (`fn_siguiente_radicado()`, C-5); el MID no envía radicado y lo devuelve leído
  de la solicitud creada. Formato `BNF-YYYY-NNNNNN`.
- **Catálogo de beneficios (RN-008):** solo PUBLICADO, `fecha_fin >= hoy`,
  `cupos_disponibles > 0`; filtros por categoría/empresa y búsqueda por título,
  todo delegado al CRUD vía `query=`.

### Variables de Entorno

Definidas en [`conf/app.conf`](conf/app.conf) vía `${VAR||default}` (expansión nativa
de Beego); las claves `*Service` quedan disponibles en `web.AppConfig`.

| Variable | Clave en app.conf | Default | Descripción |
|---|---|---|---|
| `EGRESADOS_SERVICE_CRUD_URL` | `CrudService` | `http://localhost:8080/v1` | URL del CRUD del módulo (`egresados_crud`) |
| `EGRESADOS_SERVICE_AUTENTICACION_URL` | `AutenticacionService` | `.../apioas/autenticacion_mid/v1` | autenticacion_mid (userRol) |
| `EGRESADOS_SERVICE_PARAMETROS_URL` | `ParametrosService` | `.../apioas/parametros/v1` | Servicio institucional de parámetros |
| `EGRESADOS_SERVICE_AMAZON_URL` | `AmazonService` | `.../apioas/administrativa_amazon_api/v1` | Datos de proveedor/empresa (C-2b) |
| `EGRESADOS_SERVICE_USERINFO_URL` | `Wso2UserService` | `.../oauth2/userinfo` | OIDC userinfo — identidad del dueño del token |
| `EGRESADOS_SERVICE_JWKS_URL` | `Wso2JwksService` | `.../oauth2/jwks` | JWKS de WSO2 para validar la firma RS256 del JWT entrante |
| `EGRESADOS_SERVICE_TERCEROS_URL` | `TercerosService` | `.../apioas/terceros_crud/v1` | Identidad institucional del egresado (C-2a) |
| `EGRESADOS_SERVICE_SGA_MID_URL` | `SgaMidService` | `.../apioas/sga_mid/v1` | `consultar_persona` (C-2a) |
| `EGRESADOS_SERVICE_GESTOR_DOCUMENTAL_URL` | `GestorDocumentalService` | `.../apioas/gestor_documental_mid/v1` | Gestor documental institucional (Nuxeo): subir/consultar/eliminar los PDFs de solicitudes. El cliente Angular nunca lo llama directo, solo el MID (`IdTipoDocumento=167` fijo) |
| `EGRESADOS_SERVICE_VALIDAR_JWT` | `ValidarJWT` | `true` | **Solo dev**: `false` desactiva la validación del JWT entrante (sin conectividad al JWKS/userinfo) |
| `EGRESADOS_SERVICE_PARAMETROS_LOCAL` | `ParametrosLocal` | `false` | **Solo dev/offline**: si `true`, resuelve los parámetros desde un catálogo EN MEMORIA (`parametros_service.go`), sin token ni servicio institucional. El seed local usa los MISMOS ids institucionales (7199+), así que modo local y real son intercambiables. |
| `EGRESADOS_SERVICE_HTTP_PORT` | `httpport` | `8081` | Puerto HTTP (distinto del `8080` del CRUD para poder correr ambos en local) |
| `EGRESADOS_SERVICE_RUN_MODE` | `runmode` | `prod` | Modo de ejecución de Beego (`prod`\|`dev`) |
| `PARAMETER_STORE` | `parameterStore` | _(vacío)_ | Endpoint del AWS SSM Parameter Store institucional |

### Ejecución del Proyecto
```shell
# 1. Clonar el repositorio
git clone -b develop https://github.com/udistrital/egresados_service.git

# 2. Moverse a la carpeta del repositorio
cd egresados_service

# 3. Moverse a la rama **develop**
git pull origin develop && git checkout develop

# 4. Requiere el CRUD corriendo (ver README de egresados_crud) y configurar
#    las variables de entorno que se necesiten (ver tabla arriba)
export EGRESADOS_SERVICE_RUN_MODE=dev
export EGRESADOS_SERVICE_PARAMETROS_LOCAL=true   # si el catálogo institucional aún no existe

# 5. Ejecutar el proyecto
bee run
```

### Ejecución Dockerfile
```shell
# El Dockerfile está implementado para el despliegue mediante
# el sistema de integración continua (CI).

# 1. Construir la imagen
docker build -t egresados_service .

# 2. Ejecutar el contenedor
docker run --name egresados_service \
  -e EGRESADOS_SERVICE_RUN_MODE=dev \
  -e EGRESADOS_SERVICE_CRUD_URL=http://host.docker.internal:8080/v1 \
  -e EGRESADOS_SERVICE_PARAMETROS_LOCAL=true \
  -p 8081:8081 \
  egresados_service

# 3. Comprobar que el contenedor esté en ejecución
docker ps
```

### Ejecución docker-compose
```shell
# El stack local completo (Postgres + CRUD + MID) se levanta con el
# docker-compose.yml del directorio PADRE (raíz del workspace Pasantias),
# que construye este repo con Dockerfile.local (multi-stage: compila
# dentro del contenedor, no requiere Go en el host).
# Requiere el .env de este repo (gitignorado) con las URLs institucionales;
# EGRESADOS_SERVICE_CRUD_URL la fija el compose (http://crud:8080/v1).

cd ..
docker compose up -d --build
```

## Estado CI

| Develop | Master |
| -- | -- |
| [![Build Status](https://hubci.portaloas.udistrital.edu.co/api/badges/udistrital/egresados_service/status.svg?ref=refs/heads/develop)](https://hubci.portaloas.udistrital.edu.co/udistrital/egresados_service) | [![Build Status](https://hubci.portaloas.udistrital.edu.co/api/badges/udistrital/egresados_service/status.svg?ref=refs/heads/master)](https://hubci.portaloas.udistrital.edu.co/udistrital/egresados_service) |

> Sin Sonar por ahora: el `.drone.yml` de este repo no incluye el step de
> `sonar-scanner` (a diferencia de `egresados_crud`).

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

Documentación interactiva (Swagger UI, generada con `bee generate docs`):
[`swagger/`](swagger/), servida en `/swagger/` cuando `EnableDocs = true`.

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
- Pendientes: ver `specs/logica-negocio/tasks.md`. No hay pruebas automatizadas
  todavía (`go test ./...` no tiene casos definidos).

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

## Licencia

This file is part of egresados_service.

egresados_service is free software: you can redistribute it and/or modify it under the
terms of the GNU General Public License as published by the Free Software
Foundation, either version 3 of the License, or (at your option) any later version.

egresados_service is distributed in the hope that it will be useful, but WITHOUT ANY
WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A
PARTICULAR PURPOSE. See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with
egresados_service. If not, see https://www.gnu.org/licenses/.
