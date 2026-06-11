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
  `ResolverParametroCodigo`). Tipos usados: `TIPO_USUARIO`, `ESTADO_EMPRESA`,
  `ESTADO_BENEFICIO`, `ESTADO_SOLICITUD`, `CATEGORIA_BENEFICIO`, `SECTOR_ECONOMICO`,
  `PARAMETRO_SISTEMA`.
- **Estado de solicitudes (C-4b):** el estado vigente se deriva del historial
  (`GET /historial_solicitud/solicitud/:id/vigente` del CRUD); la máquina de estados
  (RN-005) valida transiciones y cada cambio es un INSERT en el historial.
- **Radicados:** `POST /secuencia_radicado/siguiente/:anio` del CRUD; si la
  secuencia no responde, la creación falla explícitamente (sin radicados fantasma).
  Formato `BNF-YYYY-NNNNNN`.
- **Catálogo de beneficios (RN-008):** solo PUBLICADO, `fecha_fin >= hoy`,
  `cupos_disponibles > 0`; filtros por categoría/empresa y búsqueda por título,
  todo delegado al CRUD vía `query=`.

## Variables de entorno

| Variable | Default | Descripción |
|---|---|---|
| `BENEFICIOS_EGRESADOS_MID_CRUD_URL` | `http://localhost:8080/v1` | URL del CRUD del módulo |
| `BENEFICIOS_EGRESADOS_MID_PARAMETROS_URL` | `https://autenticacion.portaloas.udistrital.edu.co/apioas/parametros/v1` | Servicio institucional de parámetros |
| `BENEFICIOS_EGRESADOS_MID_AUTENTICACION_URL` | `https://autenticacion.portaloas.udistrital.edu.co/apioas/autenticacion_mid/v1` | autenticacion_mid (userRol) |
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
GET  /beneficios/:id                          detalle
POST /solicitudes                             crear solicitud (radicado + estado PENDIENTE)
GET  /solicitudes/egresado/:egresado_id       mis solicitudes (con estado vigente)
GET  /solicitudes/egresado/:egresado_id/resumen
PUT  /solicitudes/:id/cancelar                RN-005: solo PENDIENTE/REQUIERE_INFO

# Empresa
POST /empresas                                registro de empresa
GET  /empresas/:empresa_id/solicitudes        bandeja (datos mínimos del egresado, RNF-002b)
POST /empresas/:empresa_id/beneficios         publicar beneficio (RN-008b, empresa APROBADA)
PUT  /beneficios/:id                          editar beneficio
PUT  /solicitudes/:id/responder               aprobar / rechazar / requerir info
POST /solicitudes/:id/mensajes                mensajes (REQUIERE_INFO)
GET  /solicitudes/:id/mensajes

# Admin / catálogos
PUT  /empresas/:id/suspender
GET  /categorias-beneficio                    proxy del servicio de parámetros
GET  /sectores-economicos                     proxy del servicio de parámetros
```

## Pendientes conocidos

- RN-007 (solicitud única por egresado+beneficio) y RN-010 (límite de activas):
  TODOs en `solicitudes_service.go` — implementables con `query=` del CRUD.
- RN-002b/c (descuento/devolución atómica de cupos): pendiente endpoint dedicado.
- JIT provisioning (alta de `usuario`/`egresado`/`usuario_empresa` al primer login).
- Validación del JWT de WSO2 (`utils_oas`) en cada request.

## Contexto

Desarrollado en el marco de la pasantía de Ingeniería de Sistemas (2026) para la
Oficina Asesora de Sistemas (OAS) / OATI. Lineamientos: APIs separadas CRUD/MID,
plantillas `udistrital/plantilla_api_mid`, autenticación OAuth2/OIDC sobre WSO2.
