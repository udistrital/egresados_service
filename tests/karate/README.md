# Pruebas de API con Karate — `sga_mid_beneficios_egresados`

Suite de pruebas funcionales de la API del MID usando
[Karate](https://github.com/karatelabs/karate) (v1.4.1, Java 11+). Prueba el
servicio **de caja negra por HTTP**, contra el stack local real
(MID + CRUD + PostgreSQL) y con los **servicios institucionales simulados**.

## Diseño

```
┌──────────┐   HTTP    ┌─────────┐   HTTP    ┌──────────┐      ┌────────────┐
│  Karate   │ ────────► │   MID   │ ────────► │   CRUD   │ ───► │ PostgreSQL │
│ (mvn test)│           │  :8081  │           │  :8080   │      │  :5432     │
└─────┬────┘           └────┬────┘           └──────────┘      └────────────┘
      │ levanta             │ userinfo / userRol / Ágora /
      ▼                     ▼ terceros / gestor documental
┌──────────────────────────────────┐
│ Mock institucional (Karate) :8090 │
└──────────────────────────────────┘
```

- **Los servicios institucionales se mockean** (`src/test/resources/mocks/institucional-mock.feature`)
  con los contratos reales verificados contra producción (2026-07-01/02):
  `userinfo`, `userRol` (Estado `E`/vacío), `informacion_proveedor` (array, caso
  1:N, **incluye** los campos bancarios para verificar que el MID los descarta),
  `terceros_crud`, `consultar_persona` y el gestor documental (`document/upload`
  responde objeto con `res.Enlace`). Así la suite es **offline y repetible**:
  no depende de tokens vivos (~1h) ni de la red de la UD, y sirve para CI.
- **El middleware de token del MID queda ACTIVO**: los tokens ficticios de la
  suite son opacos y el MID los valida contra el `userinfo` del mock — la misma
  rama de código (`validarOpaco`) que corre en producción. Un token desconocido
  produce el 401 real del middleware.
- **Identidades de prueba** (definidas en `karate-config.js` y el mock):

  | Token | Identidad | Uso |
  |---|---|---|
  | `token-egresado` | sub `SGA-2` → usuario 2 / egresado 1 del seed | flujos de egresado |
  | `token-empresa` | sub `AGORA-1` → usuario 1, vínculo con empresa 1 | flujos de empresa |
  | `token-egresado-nuevo` | sin provisionar, con documento y código | JIT de egresado, RN-010 |
  | `token-empresa-nueva` | sin provisionar, correo con **2 proveedores** en Ágora | JIT de empresa (caso 1:N) |
  | `token-intruso` | autenticado pero sin ningún vínculo local | pruebas anti-IDOR (403) |

## Cobertura

| Feature | Qué valida |
|---|---|
| `01-seguridad-autenticacion` | 401 del middleware (sin token / token rechazado) y 403 anti-IDOR en las tres familias (egresado, empresa, bidireccional) |
| `02-catalogos` | Catálogos C-1 (categorías/sectores) con los ids institucionales |
| `03-provision-jit` | JIT de egresado y empresa: idempotencia, caso 1:N, rechazo cruzado (egresado↔empresa), whitelist RNF-002b (sin datos bancarios/documento) |
| `04-beneficios-empresa` | RF-005: publicar (RN-008b), editar (delta de cupos), retirar, vista del dueño con métricas, catálogo/búsqueda, perfil público con enriquecimiento Ágora |
| `05-solicitudes-flujo` | Flujo completo: radicado `BNF-YYYY-NNNNNN`, cupo atómico RN-002b, RN-007, máquina RN-005 con ping-pong (REQUIERE_INFO→EN_REVISION), aprobación con comprobante PDF, historial C-4b, resumen RF-013 |
| `06-solicitudes-cancelar-rechazar` | RN-002c (devolución de cupo al cancelar/rechazar), rechazo sin justificación → texto "sin perjuicio" publicado en el hilo |
| `07-documentos-solicitud` | Documentos requeridos vs. subidos, validación %PDF, comentario de empresa, proxy de descarga, permisos por rol |
| `08-rn010-limite-activas` | Límite de 5 solicitudes activas; las terminales no cuentan; el cupo no se descuenta cuando RN-010 rechaza |

## Cómo ejecutar

**Prerrequisitos:** Go, Java 11+, Maven, PostgreSQL corriendo, y el repo
`sga_crud_beneficios_egresados` clonado como hermano de este.

```powershell
cd tests\karate
.\run_pruebas.ps1
```

El script usa una **BD exclusiva de pruebas** (`beneficios_egresados_pruebas`),
que crea con `db/schema.sql` si no existe y re-siembra con `db/seed_pruebas.sql`
en cada corrida — la BD de desarrollo (`beneficios_egresados`) **no se toca**.
Luego compila y levanta CRUD y MID con las variables de entorno apuntando a esa
BD y al mock, corre `mvn test` y apaga todo. El reporte HTML queda en
`target/karate-reports/karate-summary.html`.

Si CRUD y MID ya están corriendo (con las env del script), basta con:

```powershell
mvn test
```

> **Windows Smart App Control:** si está activo puede bloquear los `.exe` de Go
> recién compilados (hash sin firma). Ver la nota en la bitácora del proyecto.

## Decisiones y limitaciones

- La suite **no** prueba la validación de firma RS256 de JWT (requiere tokens
  firmados por el WSO2 real); esa rama se probó manualmente el 2026-07-07.
  Aquí se cubre la rama de tokens opacos, que es la misma puerta 401.
- Las corridas re-siembran la BD para que los ids del seed (empresa 1,
  egresado 1, usuarios 1/2) sean deterministas. Los features crean el resto de
  sus datos por la propia API (no hay INSERTs por fuera del contrato).
- Los features se ejecutan en orden fijo y un solo hilo (comparten BD).
- El endpoint `PUT /v1/empresas/:id/suspender` (admin) no se prueba: espera los
  roles de D-8.
