# Spec — API MID (`egresados_service`)

> **Última actualización:** 2026-07-08 · **Estado:** implementado y probado e2e.
> Deriva del `BACKEND_SPEC.md` original, actualizado al contrato real (router
> verificado). Auth/JIT/autorización: ver `specs/system/autenticacion/spec.md`.
> Catálogos: ver `specs/system/parametros/spec.md`.

## Objetivo

Implementar la lógica de negocio del módulo (reglas RN-*), orquestando el CRUD
local y los servicios institucionales. Es la única API que consume el frontend.

## Alcance

**In scope:** endpoints de catálogo, solicitudes, bandeja, beneficios de empresa,
documentos, provision, perfil de empresa; reglas RN-*; integración con gestor
documental; seguridad de entrada/salida.
**Out of scope:** persistencia (CRUD), catálogos (servicio institucional),
notificaciones (pendiente, ver `tasks.md`).

## Repos involucrados

- `egresados_service` (este) — Go + Beego.
- Consume: CRUD local, Parametros, autenticacion_mid, oauth2/userinfo, administrativa_amazon_api, terceros_crud, sga_mid (consultar_persona), gestor_documental_mid.

## Endpoints (contrato vigente, prefijo `/v1`, todos con Bearer)

### Catálogo y detalle (egresado)
| Método/Ruta | Descripción |
|---|---|
| `GET /beneficios` | RF-002: catálogo paginado con filtros; PUBLICADO vigente + AGOTADO (la UI lo marca "Sin cupos"); oculta borrador/vencido/retirado |
| `GET /beneficios/:id` | RF-003: detalle + `total_solicitudes` (social proof) |
| `GET /beneficios/:id/documentos-requeridos` | Documentos que exige la empresa |
| `GET /categorias-beneficio`, `GET /sectores-economicos` | Catálogos resueltos (C-1) |

### Solicitudes (egresado)
| Método/Ruta | Descripción |
|---|---|
| `POST /solicitudes` | RF-003: valida RN-007/RN-010 → reserva cupo (RN-002b, con compensación si algo falla) → INSERT (radicado lo genera la BD) → historial PENDIENTE (RN-004); devuelve `radicado` |
| `GET /solicitudes/egresado/:egresado_id` | RF-008: mis solicitudes con estado vigente resuelto |
| `PUT /solicitudes/:id/cancelar` | RF-008: solo estados en curso; devuelve cupo (RN-002c) |
| `GET /solicitudes/egresado/:egresado_id/resumen` | RF-013: contadores por estado vigente |
| `GET /solicitudes/:id/historial` | Bitácora con códigos de estado resueltos; actor solo `usuario{id}` (RNF-002b) |

### Bandeja y respuesta (empresa)
| Método/Ruta | Descripción |
|---|---|
| `GET /empresas/:empresa_id/solicitudes` | RF-006: bandeja con datos mínimos del egresado + `datos_complementarios` |
| `PUT /solicitudes/:id/responder` | RF-007: aprobar (comprobante opcional) / rechazar (justificación ≥ 20 chars, RN-003, devuelve cupo) / requerir info (nota publicada como mensaje) |
| `POST|GET /solicitudes/:id/mensajes` | Hilo (orden cronológico); mensaje del egresado en REQUIERE_INFO auto-transiciona a EN_REVISION |

### Documentos (gestor documental institucional)
| Método/Ruta | Descripción |
|---|---|
| `GET|POST /solicitudes/:id/documentos` | Requeridos vs. subidos; subir/reemplazar PDF (magic number validado) |
| `DELETE /solicitudes/:id/documentos/:doc_id` | El egresado quita un documento |
| `PUT /documentos/:doc_id/comentario` | La empresa comenta un documento |
| `GET /documentos/:doc_id/archivo` | Proxy de solo lectura |
| `GET /solicitudes/:id/comprobante` | Comprobante opcional de la aprobación |

### Identidad y empresa
| Método/Ruta | Descripción |
|---|---|
| `POST /egresados/provision` · `POST /empresas/provision` | JIT (ver spec de autenticación) |
| `GET /empresas/:id` | Perfil público: local + Ágora on-demand (whitelist RNF-002b) + métricas |
| `GET /usuarios/:usuario_id/empresas` | Selector multiempresa (lee la BD local) |
| `POST|GET /empresas/:empresa_id/beneficios` | RF-005 publicar (empresa ACTIVA, RN-008b) / vista de gestión del dueño (todos los estados + métricas) |
| `PUT /beneficios/:id` · `PUT /beneficios/:id/retirar` | Editar (merge whitelist; regla RN-008b de edición) / retirar (→ RETIRADO, no devuelve cupos) |
| `PUT /empresas/:id/suspender` | Admin (espera roles D-8) |

## Reglas de negocio (estado real)

| RN | Implementación |
|---|---|
| RN-002b/c | Cupo reservado ANTES del INSERT vía endpoint atómico del CRUD, con compensación/devolución si falla un paso posterior; devolución al cancelar y al rechazar |
| RN-003 | Justificación obligatoria ≥ 20 caracteres al rechazar |
| RN-004 / C-4b | Toda transición = INSERT en `historial_solicitud`; el estado vigente se deriva (no hay PUT de estado) |
| RN-005 | Máquina "de quién es la pelota" (ver `specs/system/vision-general/spec.md`); ping-pong automático; nota de info publicada como mensaje |
| RN-007 | Una sola solicitud EN CURSO por (egresado, beneficio); terminales no bloquean |
| RN-008 | Catálogo: PUBLICADO vigente (+ AGOTADO visible sin CTA) |
| RN-008b | Publicar: campos obligatorios; editar: solo BORRADOR o PUBLICADO sin solicitudes en curso; si cambia `cupos_total`, `cupos_disponibles` se mueve con el delta (piso 0) |
| RN-010 | Límite de solicitudes activas (default 5; parámetro institucional sin columna de valor) |
| RNF-002b | Whitelists en bandeja, historial, perfil de empresa y proveedor; datos bancarios/anexos de Ágora jamás se deserializan |

## Contrato de integración

- **Hacia el frontend:** envelope OATI `{Status, Success, Body|Message}`; errores de negocio 422; acceso denegado 403; token inválido 401.
- **Hacia el CRUD:** helpers `GetCRUD/PostCRUD/PutCRUD` (Bearer propagado, status validado, `[{}]`→`[]` normalizado). Relaciones como `{id}`; ids de parámetro planos.
- **Gestor documental** (`gestor_documental_mid`): `POST /document/upload` con body `[{IdTipoDocumento: 167, file: base64, nombre, descripcion, metadatos: {}}]` → respuesta **objeto** `{Status, res:{Enlace: uid}}` (NO array; endpoint correcto es `upload`, no `uploadAnyFormat`; `metadatos` SIEMPRE `{}` — con contenido el servicio da 422/timeout). Consultar `GET /document/{uid}`; eliminar `DELETE /document/{uid}`. La relación solicitud↔documento vive en la tabla local `documento_solicitud`.
- **Env:** `EGRESADOS_SERVICE_{CRUD,PARAMETROS,AUTENTICACION,AMAZON,USERINFO,TERCEROS,SGA_MID,GESTOR_DOCUMENTAL,JWKS}_URL`, `_PORT` (dev: 8081), `_RUNMODE`, `_VALIDAR_JWT` (solo dev), `_PARAMETROS_LOCAL` (solo offline).

## Criterios de aceptación

1. Crear solicitud descuenta cupo y devuelve radicado real; si el INSERT falla, el cupo se compensa (verificado en logs 2026-07-02).
2. RN-007/RN-010 rechazan con mensaje claro ANTES de tocar el cupo.
3. Rechazar sin justificación suficiente → 422; rechazar válido devuelve el cupo.
4. La nota de "requiere información" le llega al egresado como mensaje del hilo.
5. Todos los endpoints protegidos devuelven 403 ante un recurso ajeno y 401 sin token válido.
6. `go build` + `go vet` limpios en cada cambio.

## Casos borde

- Beego dev pretty-printa el `[{}]` de lista vacía → siempre normalizar por JSON compactado (bug histórico de findOrCreate con id 0).
- Token institucional expira ~1h: operaciones largas pueden fallar a mitad — el MID no re-autentica, propaga el 401.
- Pedir información estando ya en REQUIERE_INFO no es transición: publica mensaje adicional.
- Retirar un beneficio no toca solicitudes en curso (siguen respondibles).
- `getEstadoActual` es N+1 sobre el historial (acotado por RN-010); optimizable con `v_solicitud_estado_vigente` si crece.
