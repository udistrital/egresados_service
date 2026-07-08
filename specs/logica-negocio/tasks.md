# Tasks — API MID

> Estado al 2026-07-08.

## Completadas (hitos)

1. [x] Propagación del Bearer a todas las llamadas salientes. — 2026-07-01
2. [x] JIT de empresa (`POST /empresas/provision`) + selector multiempresa. — 2026-07-01/02
3. [x] JIT de egresado (`POST /egresados/provision`). — 2026-07-02 (e2e)
4. [x] RN-002b/c cupos atómicos con compensación; RN-007; RN-010; RF-013. — 2026-07-01
5. [x] Máquina RN-005 modelo ping-pong + nota de info como mensaje. — 2026-07-02
6. [x] Perfil público de empresa + vista de gestión del dueño. — 2026-07-02
7. [x] Anti-IDOR completo (`autorizacion_service.go`, 403). — 2026-07-07
8. [x] Validación del JWT entrante (`middleware/jwt.go`, 401). — 2026-07-07
9. [x] Historial de solicitud (`GET /solicitudes/:id/historial`). — 2026-07-07
10. [x] Editar/retirar beneficio con regla RN-008b de edición. — 2026-07-07
11. [x] C-1 cerrado: parámetros institucionales reales, sin fallback local. — 2026-07-07
12. [x] JIT de empresa: fallback por NIT + reactivación/sincronización de empresas existentes (fix del merge de Johan). — 2026-07-08

## Pendientes

1. [ ] **Notificaciones de cambio de estado (RN-005):** evaluar e integrar `notificacion_mid` / `notificaciones_crud` (contexts confirmados en el API Store). El pendiente funcional más visible.
2. [ ] **Enriquecer la bandeja (RF-006):** programa/facultad (cadena C-2a con caché por documento dentro del request) y correo del egresado — el correo condicionado a validación RNF-002b con el revisor. Ver `plan.md`.
3. [ ] **RN-010 configurable:** hoy default 5; acordar `numero_orden` del parámetro institucional como portador del valor y leerlo.
4. [ ] Optimizar N+1 de `getEstadoActual` con `v_solicitud_estado_vigente` (cuando el volumen lo amerite).
5. [ ] Verificaciones con token vivo: rama real del middleware JWT (¿access_token JWT u opaco?), query de Ágora por id (`informacion_proveedor?query=id:{id}`), `GetAmazon` con valor url-escapado.
6. [ ] Pruebas automatizadas (hoy no existe ninguna): `transicionValida` (tabla de casos), `aJSONB`/`desdeJSONB`, `normalizarListaVacia`, helpers puros.
7. [ ] Backfill de `usuario.id_externo` para usuarios pre-JIT (si aparecen 403 en pruebas).

## Bloqueadas por OATI

1. [ ] D-6: CLIENTE_ID de producción.
2. [ ] D-8: roles WSO2 + menú — habilita `PUT /empresas/:id/suspender` y el perfil admin.
3. [ ] Validación con OATI del alcance de autorización/minimización de datos de los servicios institucionales consumidos (gestión por canal interno).
4. [ ] Confirmar con OATI la política de verificación de correo en el auto-registro de WSO2.
