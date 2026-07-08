# Spec — Catálogos vía servicio institucional de parámetros (C-1)

> **Última actualización:** 2026-07-08 · **Estado:** CERRADO end-to-end
> (2026-07-07: parámetros creados en el servicio real y módulo probado sin
> fallback local). Referencia ampliada del mecanismo:
> `docs/referencia-parametros.md`.

## Objetivo

El módulo no mantiene tablas de catálogo propias (corrección C-1 del revisor):
estados, categorías y parámetros de configuración se resuelven contra la API
institucional `Parametros` (`/apioas/parametros/v1`). Esta spec fija el
contrato, los códigos y los ids institucionales asignados.

## Alcance

**In scope:** contrato de consulta, códigos/ids definitivos, regla de acceso
centralizado en el MID, fallback local de desarrollo.
**Out of scope:** administración de los parámetros (la hace OATI); catálogo
`SECTOR_ECONOMICO` para empresas (se decidió usar CIIU de Ágora on-demand).

## Repos involucrados

- `egresados_service` — único consumidor (`services/parametros_service.go`).
- `sga_crud_beneficios_egresados` — guarda **ids planos** de parámetro (`*int` si nullable); sin FK local de catálogo (C-6: `parametro` virtualizado).
- Frontend — nunca llama al servicio de parámetros; recibe códigos resueltos por el MID.

## Requisitos

1. Todo acceso a catálogos pasa por `parametros_service.go`: `GetParametrosPorTipo`, `ResolverParametroId`, `ResolverParametroCodigo`. Prohibido crear catálogos locales nuevos.
2. La resolución es por `codigo_abreviacion` (del tipo y del valor), nunca por id quemado en código.
3. Validación de pertenencia (patrón D-3 del SGA): anclar el tipo en la misma query de desreferencia — `parametro?query=TipoParametroId__Id:{tipo},Id:{id}`.
4. El MID no confía en ids de catálogo enviados por el frontend para derivar datos sensibles: re-consulta el parámetro.

## Contrato de integración

```
GET {PARAMETROS}/parametro/?query=tipo_parametro_id.codigo_abreviacion:{CODIGO}&limit=0
Header: Authorization: Bearer {token del request entrante}

Respuesta: { "Success": true, "Status": "200", "Message": "...", "Data": [ ... ] }
```

- Dot-notation del query se traduce a `__` (contrato Beego del SGA).
- `fecha_creacion`/`fecha_modificacion` las asigna el servidor (no van en POST).
- Env: `EGRESADOS_SERVICE_PARAMETROS_URL`
  (default `https://autenticacion.portaloas.udistrital.edu.co/apioas/parametros/v1`).
- OJO homónimos: la API correcta es `Parametros` (`/parametros/v1`), NO `ParametrosGobierno`.

## Ids institucionales asignados (creados 2026-07-07, sin colisiones — D-2 cerrado)

Jerarquía: `AreaTipo → TipoParametro → Parametro`. Área del módulo: **EGR, id 32**.

| TipoParametro (id 174–179) | Valores (`parametro` ids 7199–7230) |
|---|---|
| `ESTADO_EMPRESA` | `ACTIVA` (7199), `SUSPENDIDA` (7200) — **no existe APROBADA**: la empresa nace operativa (Ágora ya la verificó), sin flujo de aprobación en login |
| `ESTADO_BENEFICIO` | BORRADOR, PUBLICADO, AGOTADO, VENCIDO, RETIRADO |
| `ESTADO_SOLICITUD` | PENDIENTE (7206), EN_REVISION, REQUIERE_INFO, APROBADA, RECHAZADA, CANCELADA |
| `CATEGORIA_BENEFICIO` | Educación, Salud, Recreación, Empleo, Descuentos, … |
| `SECTOR_ECONOMICO` | (reservado; para empresas se usa CIIU de Ágora) |
| `PARAMETRO_SISTEMA` | `LIMITE_SOLIC_ACTIVAS` (7228), `PAGINACION_DEFAULT` (7229), `JUSTIF_RECHAZO_MIN` (7230) — códigos recortados: `codigo_abreviacion` institucional es varchar(20) |

- `TIPO_USUARIO` **no** se creó en el servicio (no se usa; los tipos EGR/EMP son constantes del MID).
- La tabla institucional **no tiene columna de valor**: `LIMITE_SOLIC_ACTIVAS` usa el default 5 del MID (RN-010). Si se quiere configurable, acordar `numero_orden` como portador y leerlo.

## Modo local de desarrollo (fallback offline)

`EGRESADOS_SERVICE_PARAMETROS_LOCAL=true` resuelve los catálogos desde un
seed en memoria del MID, **con los mismos ids institucionales (7199+)** — la BD
dev se migró a esos ids (2026-07-07, 97 filas), así que modo local y real son
intercambiables. Desde el 2026-07-07 el MID de dev se levanta SIN esta env
(queda solo como modo offline).

## Criterios de aceptación

1. Con token vivo y sin `PARAMETROS_LOCAL`: catálogo, categorías, provision, mis-solicitudes (18 estados resueltos), historial y resumen responden 200 (verificado 2026-07-07).
2. Un id de parámetro de otro tipo (o inexistente) es rechazado por la validación anclada al tipo.
3. Los códigos usados en código (`ACTIVA`, `PUBLICADO`, `PENDIENTE`, …) existen en el servicio institucional con esos `codigo_abreviacion` exactos.

## Casos borde

- Servicio de parámetros caído o 401 → el MID responde error explícito (no hay caché de catálogos aún; el fallback local es solo dev).
- Cambio futuro de ids institucionales → solo afecta a la BD (ids persistidos); el código resuelve por código, no por id.
