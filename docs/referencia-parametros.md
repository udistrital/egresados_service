# Referencia — Mecanismo institucional de parámetros

> **Estado del documento (2026-07-08):** exploración histórica de
> `parametros_crud` (2026-06-04), antes archivo `PARAMETROS.md`. La parte
> operativa quedó EJECUTADA el 2026-07-07 (tipos 174–179 y parámetros 7199–7230
> creados, D-2 cerrado); el contrato vigente está en
> `specs/system/parametros/spec.md`.

> **Estado:** Fase 1 — exploración completada. Sin modificaciones al código.
> **Fecha:** 2026-06-04
> **Autor:** Daniel Velandia (pasante)

---

## 1. Cómo funciona el mecanismo de parámetros en este repo

### 1.1 Tecnología y arquitectura

Este repositorio (`parametros_crud`) es un **API REST en Go** construida con el framework Beego y ORM propio de Beego contra PostgreSQL (esquema `parametros`). No hay UI, no hay seed scripts ni archivos YAML de configuración; **todo se registra mediante llamadas HTTP**.

**Archivos clave confirmados:**

| Archivo | Propósito |
|---|---|
| `models/area_tipo.go` | Struct Go + ORM del nivel 1 |
| `models/tipo_parametro.go` | Struct Go + ORM del nivel 2 |
| `models/parametro.go` | Struct Go + ORM del nivel 3 (valor real) |
| `controllers/parametro.go` | Handler HTTP — asigna fechas automáticamente |
| `controllers/tipo_parametro.go` | Handler HTTP |
| `controllers/area_tipo.go` | Handler HTTP |
| `routers/router.go` | Declaración de rutas bajo prefijo `/v1` |
| `database/scripts/20211220_191816_initial_from_sql.up.sql` | Migración SQL inicial — crea las tablas |

---

### 1.2 Jerarquía de tres niveles

El sistema usa **tres capas anidadas** (cada una FK a la anterior):

```
AreaTipo  (nivel 1 — agrupa módulos o sistemas)
 └── TipoParametro  (nivel 2 — categoría dentro del área)
      └── Parametro  (nivel 3 — el valor/ítem del catálogo)
               └── Parametro (opcional: hijo de otro Parametro vía parametro_padre_id)
```

**Ejemplo conceptual:**

```
AreaTipo: "Beneficios Egresados"
  ├── TipoParametro: "Estado Beneficio"
  │     ├── Parametro: "BORRADOR"
  │     ├── Parametro: "PUBLICADO"
  │     └── ...
  └── TipoParametro: "Categoría Beneficio"
        ├── Parametro: "Educación"
        └── ...
```

---

### 1.3 Modelos de datos exactos

#### AreaTipo (`models/area_tipo.go`, líneas 12-21)

```go
type AreaTipo struct {
    Id                int     // PK autoincrement
    Nombre            string  // requerido, max 100 chars
    Descripcion       string  // opcional, max 100 chars
    CodigoAbreviacion string  // opcional, max 20 chars
    Activo            bool    // requerido
    NumeroOrden       float64 // opcional, numeric(5,2)
    FechaCreacion     string  // timestamp — lo asigna el servidor
    FechaModificacion string  // timestamp — lo asigna el servidor
}
```

#### TipoParametro (`models/tipo_parametro.go`, líneas 12-22)

```go
type TipoParametro struct {
    Id                int       // PK autoincrement
    Nombre            string    // requerido, max 100 chars
    Descripcion       string    // opcional, max 100 chars
    CodigoAbreviacion string    // opcional, max 20 chars
    Activo            bool      // requerido
    NumeroOrden       float64   // opcional
    FechaCreacion     string    // asignado por el servidor
    FechaModificacion string    // asignado por el servidor
    AreaTipoId        *AreaTipo // FK requerida — objeto con campo "id"
}
```

#### Parametro (`models/parametro.go`, líneas 12-23)

```go
type Parametro struct {
    Id                int            // PK autoincrement
    Nombre            string         // requerido, max 100 chars
    Descripcion       string         // opcional, max 100 chars
    CodigoAbreviacion string         // opcional, max 20 chars
    Activo            bool           // requerido
    NumeroOrden       float64        // opcional
    FechaCreacion     string         // asignado por el servidor
    FechaModificacion string         // asignado por el servidor
    TipoParametroId   *TipoParametro // FK requerida — objeto con campo "id"
    ParametroPadreId  *Parametro     // FK opcional — para parámetros hijos
}
```

> **Importante:** `FechaCreacion` y `FechaModificacion` **no se envían en el payload**; el servidor las asigna automáticamente con la hora de Bogotá (`controllers/parametro.go`, línea 40-41).

---

### 1.4 Endpoints REST disponibles

Todos los endpoints están bajo el prefijo `/v1` (`routers/router.go`, línea 17).

| Método | Ruta | Acción |
|--------|------|--------|
| `POST` | `/v1/area_tipo/` | Crear AreaTipo |
| `GET` | `/v1/area_tipo/` | Listar AreaTipos |
| `GET` | `/v1/area_tipo/:id` | Obtener uno por ID |
| `PUT` | `/v1/area_tipo/:id` | Actualizar |
| `DELETE` | `/v1/area_tipo/:id` | Eliminar |
| `POST` | `/v1/tipo_parametro/` | Crear TipoParametro |
| `GET` | `/v1/tipo_parametro/` | Listar TipoParametros |
| `GET` | `/v1/tipo_parametro/:id` | Obtener uno por ID |
| `POST` | `/v1/parametro/` | Crear Parametro |
| `GET` | `/v1/parametro/` | Listar Parametros |
| `GET` | `/v1/parametro/:id` | Obtener uno por ID |

**Parámetros de query** disponibles en todos los `GET` de lista:

```
?query=campo:valor   → filtrar por campo
?limit=50            → número de resultados (default: 10)
?offset=0            → paginación
?sortby=nombre&order=asc
?fields=id,nombre    → proyección de campos
```

---

### 1.5 Proceso de registro — paso a paso

Los parámetros se crean mediante **tres llamadas HTTP en orden secuencial**, ya que cada nivel necesita el `id` del nivel superior.

**Importante:** Los `id` son autogenerados por la base de datos. El flujo real es:

```
Paso 1: POST /v1/area_tipo/        → recibe id del AreaTipo creado
Paso 2: POST /v1/tipo_parametro/   → usa ese id en area_tipo_id
                                   → recibe id del TipoParametro creado
Paso 3: POST /v1/parametro/        → usa ese id en tipo_parametro_id
         (repetir por cada valor del catálogo)
```

---

## 2. Contrato exacto — payloads de ejemplo

### 2.1 Crear AreaTipo

```
POST /v1/area_tipo/
Content-Type: application/json
```

```json
{
  "nombre": "Nombre del área",
  "descripcion": "Descripción breve (máx 100 chars)",
  "codigo_abreviacion": "ABREV",
  "activo": true,
  "numero_orden": 1.0
}
```

**Respuesta exitosa (HTTP 201):**

```json
{
  "Success": true,
  "Status": "201",
  "Message": "Registration successful",
  "Data": {
    "Id": 42,
    "Nombre": "Nombre del área",
    "Descripcion": "Descripción breve",
    "CodigoAbreviacion": "ABREV",
    "Activo": true,
    "NumeroOrden": 1.0,
    "FechaCreacion": "2026-06-04T10:00:00-05:00",
    "FechaModificacion": "2026-06-04T10:00:00-05:00"
  }
}
```

### 2.2 Crear TipoParametro

```
POST /v1/tipo_parametro/
Content-Type: application/json
```

```json
{
  "nombre": "Nombre del tipo",
  "descripcion": "Descripción breve",
  "codigo_abreviacion": "TIPO_ABREV",
  "activo": true,
  "numero_orden": 1.0,
  "area_tipo_id": { "id": 42 }
}
```

> `area_tipo_id.id` es el `Id` recibido en el paso anterior.

### 2.3 Crear Parametro (valor individual del catálogo)

```
POST /v1/parametro/
Content-Type: application/json
```

```json
{
  "nombre": "NOMBRE_VALOR",
  "descripcion": "Descripción del valor",
  "codigo_abreviacion": "COD",
  "activo": true,
  "numero_orden": 1.0,
  "tipo_parametro_id": { "id": 17 },
  "parametro_padre_id": null
}
```

> `tipo_parametro_id.id` es el `Id` recibido del TipoParametro creado.

---

## 3. Plantilla para los 6 catálogos del submódulo Beneficios Egresados

> **Convención usada:** un único `AreaTipo` para el módulo completo, un `TipoParametro` por catálogo, y un `Parametro` por cada valor/estado.

---

### PASO 1 — Crear el AreaTipo del módulo

```
POST /v1/area_tipo/
```

```json
{
  "nombre": "Beneficios Egresados",
  "descripcion": "Módulo de gestión de beneficios para egresados UD",
  "codigo_abreviacion": "BEN_EGR",
  "activo": true,
  "numero_orden": 1.0
}
```

> Guardar el `id` resultante — se llama **`{ID_AREA}`** en los bloques siguientes.

---

### PASO 2 — Crear los 6 TipoParametros

#### 2.1 — tipo_usuario

```
POST /v1/tipo_parametro/
```

```json
{
  "nombre": "Tipo Usuario",
  "descripcion": "Clasificación de los usuarios del módulo de egresados",
  "codigo_abreviacion": "TIPO_USR",
  "activo": true,
  "numero_orden": 1.0,
  "area_tipo_id": { "id": "{ID_AREA}" }
}
```

#### 2.2 — estado_empresa

```
POST /v1/tipo_parametro/
```

```json
{
  "nombre": "Estado Empresa",
  "descripcion": "Estados del ciclo de vida de una empresa registrada",
  "codigo_abreviacion": "EST_EMP",
  "activo": true,
  "numero_orden": 2.0,
  "area_tipo_id": { "id": "{ID_AREA}" }
}
```

#### 2.3 — estado_beneficio

```
POST /v1/tipo_parametro/
```

```json
{
  "nombre": "Estado Beneficio",
  "descripcion": "Estados del ciclo de vida de un beneficio publicado",
  "codigo_abreviacion": "EST_BEN",
  "activo": true,
  "numero_orden": 3.0,
  "area_tipo_id": { "id": "{ID_AREA}" }
}
```

#### 2.4 — estado_solicitud

```
POST /v1/tipo_parametro/
```

```json
{
  "nombre": "Estado Solicitud",
  "descripcion": "Estados del flujo de aprobación de una solicitud de beneficio",
  "codigo_abreviacion": "EST_SOL",
  "activo": true,
  "numero_orden": 4.0,
  "area_tipo_id": { "id": "{ID_AREA}" }
}
```

#### 2.5 — categoria_beneficio

```
POST /v1/tipo_parametro/
```

```json
{
  "nombre": "Categoría Beneficio",
  "descripcion": "Categorías temáticas para clasificar los beneficios ofertados",
  "codigo_abreviacion": "CAT_BEN",
  "activo": true,
  "numero_orden": 5.0,
  "area_tipo_id": { "id": "{ID_AREA}" }
}
```

#### 2.6 — sector_economico

```
POST /v1/tipo_parametro/
```

```json
{
  "nombre": "Sector Económico",
  "descripcion": "Sectores económicos para clasificar las empresas registradas",
  "codigo_abreviacion": "SEC_ECO",
  "activo": true,
  "numero_orden": 6.0,
  "area_tipo_id": { "id": "{ID_AREA}" }
}
```

---

### PASO 3 — Crear los Parametros (valores) por catálogo

> En cada bloque, reemplazar `{ID_TIPO_USR}`, `{ID_EST_EMP}`, etc. con los `id` recibidos en el Paso 2.

---

#### 3.1 — Valores de tipo_usuario

```
POST /v1/parametro/   (repetir por cada fila)
```

| numero_orden | nombre | descripcion | codigo_abreviacion |
|---|---|---|---|
| 1 | Egresado | Usuario egresado de la Universidad Distrital | EGRESADO |
| 2 | Empresa | Empresa u organización registrada como aliada | EMPRESA |
| 3 | Administrador | Usuario administrador del módulo de egresados | ADMIN |

**Ejemplo para "Egresado":**

```json
{
  "nombre": "Egresado",
  "descripcion": "Usuario egresado de la Universidad Distrital",
  "codigo_abreviacion": "EGRESADO",
  "activo": true,
  "numero_orden": 1.0,
  "tipo_parametro_id": { "id": "{ID_TIPO_USR}" },
  "parametro_padre_id": null
}
```

---

#### 3.2 — Valores de estado_empresa

| numero_orden | nombre | descripcion | codigo_abreviacion |
|---|---|---|---|
| 1 | En Revisión | Empresa en proceso de verificación y aprobación | EN_REVISION |
| 2 | Aprobada | Empresa verificada y habilitada para publicar beneficios | APROBADA |
| 3 | Rechazada | Empresa rechazada tras el proceso de revisión | RECHAZADA |
| 4 | Suspendida | Empresa suspendida temporalmente por incumplimiento | SUSPENDIDA |

**Ejemplo para "En Revisión":**

```json
{
  "nombre": "En Revisión",
  "descripcion": "Empresa en proceso de verificación y aprobación",
  "codigo_abreviacion": "EN_REVISION",
  "activo": true,
  "numero_orden": 1.0,
  "tipo_parametro_id": { "id": "{ID_EST_EMP}" },
  "parametro_padre_id": null
}
```

---

#### 3.3 — Valores de estado_beneficio

| numero_orden | nombre | descripcion | codigo_abreviacion |
|---|---|---|---|
| 1 | Borrador | Beneficio en edición, no visible para egresados | BORRADOR |
| 2 | Publicado | Beneficio activo y visible para egresados | PUBLICADO |
| 3 | Agotado | Beneficio sin cupos disponibles | AGOTADO |
| 4 | Vencido | Beneficio cuya fecha de vigencia expiró | VENCIDO |
| 5 | Retirado | Beneficio retirado manualmente por la empresa | RETIRADO |

**Ejemplo para "Borrador":**

```json
{
  "nombre": "Borrador",
  "descripcion": "Beneficio en edición, no visible para egresados",
  "codigo_abreviacion": "BORRADOR",
  "activo": true,
  "numero_orden": 1.0,
  "tipo_parametro_id": { "id": "{ID_EST_BEN}" },
  "parametro_padre_id": null
}
```

---

#### 3.4 — Valores de estado_solicitud

| numero_orden | nombre | descripcion | codigo_abreviacion |
|---|---|---|---|
| 1 | Pendiente | Solicitud recibida, en espera de revisión | PENDIENTE |
| 2 | En Revisión | Solicitud siendo evaluada por la empresa | EN_REVISION |
| 3 | Requiere Información | Solicitud requiere datos adicionales del egresado | REQUIERE_INFO |
| 4 | Aprobada | Solicitud aprobada por la empresa | APROBADA |
| 5 | Rechazada | Solicitud rechazada por la empresa | RECHAZADA |
| 6 | Cancelada | Solicitud cancelada por el egresado | CANCELADA |

**Ejemplo para "Pendiente":**

```json
{
  "nombre": "Pendiente",
  "descripcion": "Solicitud recibida, en espera de revisión",
  "codigo_abreviacion": "PENDIENTE",
  "activo": true,
  "numero_orden": 1.0,
  "tipo_parametro_id": { "id": "{ID_EST_SOL}" },
  "parametro_padre_id": null
}
```

---

#### 3.5 — Valores de categoria_beneficio

| numero_orden | nombre | descripcion | codigo_abreviacion |
|---|---|---|---|
| 1 | Educación | Becas, cursos, diplomados y formación académica | EDUCACION |
| 2 | Salud | Servicios médicos, seguros y bienestar | SALUD |
| 3 | Recreación | Actividades culturales, deportivas y de ocio | RECREACION |
| 4 | Empleo | Ofertas laborales y bolsa de empleo | EMPLEO |
| 5 | Descuentos | Descuentos en productos y servicios comerciales | DESCUENTOS |

**Ejemplo para "Educación":**

```json
{
  "nombre": "Educación",
  "descripcion": "Becas, cursos, diplomados y formación académica",
  "codigo_abreviacion": "EDUCACION",
  "activo": true,
  "numero_orden": 1.0,
  "tipo_parametro_id": { "id": "{ID_CAT_BEN}" },
  "parametro_padre_id": null
}
```

---

#### 3.6 — Valores de sector_economico

| numero_orden | nombre | descripcion | codigo_abreviacion |
|---|---|---|---|
| 1 | Tecnología | Empresas de tecnología, software y telecomunicaciones | TECNOLOGIA |
| 2 | Salud | Empresas del sector salud y servicios médicos | SALUD |
| 3 | Educación | Instituciones educativas y editoriales | EDUCACION |
| 4 | Comercio | Comercio al por mayor y menor | COMERCIO |
| 5 | Industria | Manufactura e industria en general | INDUSTRIA |
| 6 | Servicios | Servicios profesionales, consultoría y outsourcing | SERVICIOS |
| 7 | Agroindustria | Sector agrícola, pecuario y agroindustrial | AGROINDUSTRIA |
| 8 | Construcción | Obras civiles, arquitectura e ingeniería | CONSTRUCCION |
| 9 | Financiero | Bancos, aseguradoras y servicios financieros | FINANCIERO |
| 10 | Entretenimiento | Medios, entretenimiento y arte | ENTRETENIMIENTO |

**Ejemplo para "Tecnología":**

```json
{
  "nombre": "Tecnología",
  "descripcion": "Empresas de tecnología, software y telecomunicaciones",
  "codigo_abreviacion": "TECNOLOGIA",
  "activo": true,
  "numero_orden": 1.0,
  "tipo_parametro_id": { "id": "{ID_SEC_ECO}" },
  "parametro_padre_id": null
}
```

---

## 4. Cómo se "envía" al equipo — resumen operativo

Basado en el código del repositorio, **no existe ningún seed script, archivo JSON de importación ni mecanismo batch** para cargar parámetros. El sistema es puramente REST.

El proceso de entrega al equipo será, con alta probabilidad, una de estas dos opciones (confirmar con ellos):

| Opción | Descripción | Indicios en el repo |
|---|---|---|
| **A — El equipo aplica los POSTs** | Tú entregas los payloads (como este documento), ellos los ejecutan contra el ambiente de desarrollo/staging usando Postman, curl o un script | Es la única forma documentada en el código |
| **B — Inserción directa en BD** | Tú entregas una migración SQL tipo `INSERT INTO parametros.parametro ...` | No hay ningún ejemplo de esto en el repo, pero es técnicamente posible |

**La opción A es la convención vigente** según la arquitectura del repositorio (API REST puro, sin seeds).

---

## 5. Dudas abiertas — confirmar con el equipo

Antes de enviarles los parámetros, confirmar los siguientes puntos:

1. **¿Existe ya un AreaTipo para el SGA/Egresados?**
   Pregunta: `GET /v1/area_tipo/?limit=100` — si ya hay un área de "Egresados" o "Bienestar", tus TipoParametros deben colgar de ahí en vez de crear uno nuevo.

2. **¿Cuál es la URL base del servicio en el ambiente compartido?**
   El `conf/app.conf` usa variables de entorno; no hay URL fija en el código. Necesitas saber si es `http://localhost:8080` (local) o una URL de staging.

3. **¿Hay autenticación en los endpoints POST?**
   El código no muestra middleware de auth en `routers/router.go`, pero puede estar en el API Gateway o en el nivel de infraestructura.

4. **¿Los IDs de TipoParametro se te devuelven o debes consultarlos luego?**
   Por la arquitectura REST, sí se devuelven en la respuesta `201`. Solo confirma que el equipo te dará acceso de escritura o que ellos aplican los POSTs y te devuelven los IDs para que los referencie en tu microservicio.

5. **¿La `descripcion` tiene restricciones adicionales de vocabulario?**
   La BD acepta hasta 100 caracteres. Revisar si hay alguna guía de estilo interna.

6. **¿Los valores de `codigo_abreviacion` tienen que seguir algún patrón?**
   No hay validación en el código (`null`able), pero puede haber una convención institucional (ej. todo mayúsculas, sin tildes).

7. **¿Se permite agregar más valores a `categoria_beneficio` y `sector_economico` en el futuro?**
   Técnicamente sí (el API lo soporta), pero confirmar si hay un proceso de aprobación para nuevos parámetros.

---

## 6. Diagrama entidad-relación (tablas reales)

```
parametros.area_tipo
  PK id
     nombre            VARCHAR(100) NOT NULL
     descripcion       VARCHAR(100)
     codigo_abreviacion VARCHAR(20)
     activo            BOOLEAN NOT NULL
     numero_orden      NUMERIC(5,2)
     fecha_creacion    TIMESTAMP NOT NULL
     fecha_modificacion TIMESTAMP NOT NULL

parametros.tipo_parametro
  PK id
     nombre            VARCHAR(100) NOT NULL
     descripcion       VARCHAR(100)
     codigo_abreviacion VARCHAR(20)
     activo            BOOLEAN NOT NULL
     numero_orden      NUMERIC(5,2)
     fecha_creacion    TIMESTAMP NOT NULL
     fecha_modificacion TIMESTAMP NOT NULL
  FK area_tipo_id → area_tipo.id

parametros.parametro
  PK id
     nombre            VARCHAR(100) NOT NULL
     descripcion       VARCHAR(100)
     codigo_abreviacion VARCHAR(20)
     activo            BOOLEAN NOT NULL
     numero_orden      NUMERIC(5,2)
     fecha_creacion    TIMESTAMP NOT NULL
     fecha_modificacion TIMESTAMP NOT NULL
  FK tipo_parametro_id → tipo_parametro.id   (requerido)
  FK parametro_padre_id → parametro.id        (opcional, para jerarquía)
```

---

*Fuentes: `models/parametro.go`, `models/tipo_parametro.go`, `models/area_tipo.go`, `controllers/parametro.go`, `routers/router.go`, `database/scripts/20211220_191816_initial_from_sql.up.sql`*
