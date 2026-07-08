# Spec — Visión general del sistema

> **Última actualización:** 2026-07-08 · **Estado:** módulo funcionando end-to-end
> para ambos perfiles contra servicios y BD reales (verificado 2026-07-02).
> Reemplaza a `CONTEXTO_PROYECTO.md`.

## Objetivo

Centralizar en el SGA la oferta de beneficios y convenios para egresados de la
Universidad Distrital, hoy dispersa en canales no integrados. Empresas aliadas
publican beneficios; los egresados los consultan, solicitan y hacen seguimiento;
todo queda trazado (radicados, historial de estados, auditoría).

## Alcance

**In scope (MVP):**
- Catálogo de beneficios con filtros y paginación (RF-002).
- Solicitud de beneficio con radicado `BNF-YYYY-NNNNNN` y descuento atómico de cupo (RF-003).
- Seguimiento de solicitudes: estado, línea de tiempo, hilo de mensajes, cancelación (RF-008).
- Bandeja de la empresa y respuesta a solicitudes: aprobar / rechazar / requerir información (RF-006, RF-007).
- Publicación, edición y retiro de beneficios por la empresa (RF-005).
- Aprovisionamiento de identidad (JIT) para egresados y empresas al primer login (reemplaza a los "registros" RF-001/RF-004: no hay formulario de registro propio, la identidad se deriva del token WSO2 y de los sistemas institucionales).
- Límite de solicitudes activas (RF-012 / RN-010) y resumen de actividad (RF-013).
- Documentos requeridos por beneficio: subida de PDFs al gestor documental institucional, comentarios de la empresa y comprobante de aprobación.

**Out of scope:**
- RF-009 (redención con certificado/QR) — fase posterior al MVP.
- Vista de administrador UD (el endpoint de suspender empresa existe pero espera los roles de WSO2, D-8).
- Notificaciones por correo/push de cambios de estado (candidato `notificacion_mid`, no integrado).

## Repos involucrados

| Repo | Stack | Rol |
|---|---|---|
| `sga_crud_beneficios_egresados` | Go + Beego ORM | CRUD genérico + PostgreSQL (schema `beneficios_egresados`) |
| `egresados_service` | Go + Beego | Lógica de negocio, reglas RN-*, orquestación e integración institucional |
| `sga_cliente_beneficios_egresados_mf` | Angular 16.2 + Single-SPA | Micro-frontend con las vistas de egresado y de empresa |

El frontend **solo** habla con el MID; el MID consume el CRUD local y los
servicios institucionales (parámetros, terceros, sga_mid, Ágora/amazon,
gestor documental, autenticación). Ver `specs/system/autenticacion/spec.md` y
`specs/system/parametros/spec.md` para esos contratos.

## Actores

1. **Egresado** — consulta el catálogo, solicita beneficios, sube documentos, hace seguimiento.
2. **Empresa (representante)** — publica/edita/retira beneficios y gestiona la bandeja de solicitudes. Un representante puede tener **varias empresas** (caso 1:N real, selector multiempresa).
3. **Administrador UD** — suspende empresas (pendiente de roles D-8; sin UI aún).
4. **Sistema** — validaciones automáticas: cupos, límites, máquina de estados, radicados.

## Requisitos funcionales

| RF | Nombre | Estado |
|---|---|---|
| RF-001 | Identidad de egresado (JIT al login, deriva del token) | ✅ e2e 2026-07-02 |
| RF-002 | Catálogo de beneficios con filtros y paginación | ✅ |
| RF-003 | Solicitud de beneficio con radicado y cupo atómico | ✅ e2e (radicado `BNF-2026-000003` real) |
| RF-004 | Identidad de empresa (JIT al login vía Ágora) | ✅ e2e 2026-07-02 |
| RF-005 | Publicar / editar / retirar beneficio (empresa ACTIVA) | ✅ (editar/retirar 2026-07-07) |
| RF-006 | Bandeja de solicitudes recibidas (minimización RNF-002b) | ✅ (enriquecer datos del egresado: pendiente) |
| RF-007 | Responder solicitud + hilo de mensajes | ✅ (modelo ping-pong RN-005) |
| RF-008 | Seguimiento: estados, historial, cancelar | ✅ |
| RF-009 | Redención (certificado/QR) | ⛔ fuera del MVP |
| RF-012 | Límite de solicitudes activas | ✅ (RN-010, default 5) |
| RF-013 | Resumen de actividad por estado | ✅ |

## Máquina de estados de solicitud (RN-005, modelo vigente)

Modelo **"de quién es la pelota"**: `REQUIERE_INFO` = esperando al egresado;
`EN_REVISION` = esperando a la empresa.

```
PENDIENTE      → EN_REVISION | REQUIERE_INFO | APROBADA | RECHAZADA | CANCELADA
EN_REVISION    → APROBADA | RECHAZADA | REQUIERE_INFO | CANCELADA
REQUIERE_INFO  → EN_REVISION (auto, al responder el egresado) | APROBADA | RECHAZADA | CANCELADA
APROBADA / RECHAZADA / CANCELADA → estados finales
```

- El egresado puede **cancelar** en cualquier estado en curso (PENDIENTE / REQUIERE_INFO / EN_REVISION).
- La nota de "pedir información" se publica **como mensaje del hilo** (la justificación del historial no es visible para el egresado).
- Mensaje del egresado en REQUIERE_INFO ⇒ auto-transición a EN_REVISION (ping-pong).
- El estado vigente **solo** se deriva del historial (`historial_solicitud`); no existe campo de estado en la solicitud (decisión C-4b).

## Criterios de aceptación (nivel sistema)

1. Un egresado real (login WSO2) entra al catálogo, solicita un beneficio y recibe radicado con formato `^BNF-\d{4}-\d{6}$`; el cupo del beneficio baja en 1 de forma atómica.
2. Un usuario de empresa real entra por el MISMO login, queda vinculado a su(s) empresa(s) de Ágora sin formulario, y ve la bandeja con solicitudes reales.
3. Rechazar una solicitud exige justificación ≥ 20 caracteres y devuelve el cupo.
4. Ningún endpoint expone datos bancarios, anexos de Ágora ni datos personales fuera de la whitelist (RNF-002b, Ley 1581/2012).
5. Un usuario autenticado no puede operar recursos de otro egresado/empresa (anti-IDOR, verificado con 403 el 2026-07-07).

## Casos borde conocidos

- **Correo con 2+ proveedores en Ágora** (caso real verificado contra el servicio institucional): la sesión guarda todas las empresas y la UI ofrece selector.
- **Usuario de empresa sin documento** (self-signup WSO2): la identidad se keya por `sub` de WSO2, no por cédula.
- **Beneficio agotado**: visible en el catálogo como "Sin cupos"; vencidos/retirados/borradores ocultos al egresado.
- **Retirar un beneficio** no devuelve cupos ni cancela solicitudes en curso: siguen respondibles desde la bandeja.

## Plan de entrega (histórico) y estado

- Sprint 1 (semanas 1–4): discovery, diseño, autenticación, catálogo — ✅.
- Sprint 2 (semanas 5–8): solicitudes, bandeja, respuestas, seguimiento, límites, resumen — ✅.
- Post-MVP en curso: notificaciones, enriquecimiento de bandeja, integración single-spa, roles de producción (ver `tasks.md` de cada repo).

## Glosario

- **SGA** — Sistema de Gestión Académica institucional.
- **OATI/OAS** — Oficina Asesora de Tecnologías de la Información (antes de Sistemas).
- **MF** — micro-frontend (Single-SPA).
- **MID / CRUD** — API de lógica de negocio / API de acceso a datos (Go + Beego).
- **Ágora** — banco de proveedores institucional (fuente de datos de empresas).
- **JIT provisioning** — creación de la identidad local en el primer login, derivada del token.
- **Radicado** — identificador único de solicitud (`BNF-YYYY-NNNNNN`).
