# Referencia — Servicios backend del ecosistema SGA

> **Estado del documento (2026-07-08):** catálogo de los servicios que consume el
> `sga_cliente` ORIGINAL del SGA (no nuestro módulo), antes archivo
> `SERVICIOS_BACKEND.md`. Útil como mapa del ecosistema; los servicios que
> nuestro módulo consume realmente están documentados en
> `specs/system/autenticacion/spec.md` y `specs/logica-negocio/spec.md`.

Documentación de todos los servicios de backend consumidos por ese frontend Angular.  
Todos los servicios pasan por el `RequestManager` (`/src/app/managers/requestManager.ts`), que adjunta automáticamente el Bearer token del `localStorage` y extrae la propiedad `Body` de las respuestas cuando está presente.

---

## Tabla de contenidos

1. [Configuración general](#configuración-general)
2. [Autenticación / OAuth](#autenticación--oauth)
3. [Servicios CRUD](#servicios-crud)
4. [Servicios Middleware (MID)](#servicios-middleware-mid)
5. [Servicios especiales](#servicios-especiales)
6. [Comunicación en tiempo real](#comunicación-en-tiempo-real)
7. [Pasarela de pagos](#pasarela-de-pagos)
8. [Patrones HTTP comunes](#patrones-http-comunes)

---

## Configuración general

| Entorno | Dominio base |
|---------|-------------|
| Producción | `https://autenticacion.portaloas.udistrital.edu.co/apioas/` |
| Desarrollo (local) | `http://localhost:8095` / `http://localhost:8119` |
| Pruebas | `http://pruebasapi.intranetoas.udistrital.edu.co` |

Los archivos de entorno están en `src/environments/`.

---

## Autenticación / OAuth

| Campo | Valor |
|-------|-------|
| **Protocolo** | OpenID Connect (OIDC) / OAuth 2.0 |
| **URL de autorización** | `https://autenticacion.portaloas.udistrital.edu.co/oauth2/authorize` |
| **URL de logout** | `https://autenticacion.portaloas.udistrital.edu.co/oidc/logout` |
| **Endpoint roles** | `POST /apioas/autenticacion_mid/v1/token/userRol` |
| **Archivo** | `src/app/@core/utils/implicit_autentication.service.ts` |

### Flujo
1. El cliente redirige al usuario a la URL de autorización con `ClientID`, `Scope` y `RedirectURL`.
2. Recibe `id_token` y `access_token` en el hash de la redirección.
3. Tokens almacenados en `localStorage`; se incluyen como `Bearer` en cada request.
4. Se consulta el endpoint de roles con `{ user: email }` para obtener los roles del usuario.

### Resolución de usuario
- Por documento: `datos_identificacion?query=Activo:true,Numero:{documento}`
- Por email: `tercero?query=Activo:True,UsuarioWSO2:{email}`

---

## Servicios CRUD

### 1. `agora_crud` — Agora

| Campo | Detalle |
|-------|---------|
| **URL base** | `…/agora_crud/v1/` |
| **Archivo** | `src/app/@core/data/agora.service.ts` |
| **Métodos** | GET, POST, PUT, DELETE |
| **Descripción** | CRUD para el foro académico / gestión de discusiones (Agora). |
| **Respuesta** | Objeto o lista de registros del foro. |

---

### 2. `cidc_crud` / `siciud_crud` — CIDC

| Campo | Detalle |
|-------|---------|
| **URL base (prod)** | `…/siciud_crud/v1/` |
| **URL base (dev)** | `http://200.69.103.88:3114/api/v1/` |
| **Archivo** | `src/app/@core/data/cidc.service.ts` |
| **Métodos** | GET, POST, PUT, DELETE |
| **Descripción** | Gestión de información ciudadana / civil (CIDC). |
| **Respuesta** | Registros de identificación / entidad. |

---

### 3. `configuracion_crud_api` — Configuración del sistema

| Campo | Detalle |
|-------|---------|
| **URL base** | `…/configuracion_crud_api/v1/` |
| **Archivo** | `src/app/@core/data/configuracion.service.ts` |
| **Métodos** | GET, POST, PUT, DELETE |
| **Descripción** | Parámetros globales del sistema y árbol de menús (`menu_opcion_padre/ArbolMenus/`). |
| **Respuesta** | Configuraciones clave-valor; árbol jerárquico de opciones de menú. |

---

### 4. `core_crud` — Core académico

| Campo | Detalle |
|-------|---------|
| **URL base** | `…/core_crud/v2/` (prod) / `v1/` (alternativo) |
| **Archivo** | `src/app/@core/data/core.service.ts` |
| **Métodos** | GET, POST, PUT, DELETE |
| **Descripción** | Gestión del núcleo académico: programas, prerequisitos, habilitación de periodos. |
| **Respuesta** | Objetos de programa, período, prerequisito, etc. |

---

### 5. `dependencias_api` — Dependencias

| Campo | Detalle |
|-------|---------|
| **URL base** | `…/dependencias_api/v1/` |
| **Archivo** | `src/app/@core/data/dependencias.service.ts` |
| **Métodos** | GET (SOAP) |
| **Descripción** | Unidades organizacionales / dependencias institucionales. Usa cabeceras `multipart/form-data`. |
| **Respuesta** | Lista de dependencias / unidades. |

---

### 6. `matriculas_descuentos_crud` — Descuentos académicos

| Campo | Detalle |
|-------|---------|
| **URL base** | `…/matriculas_descuentos_crud/v2/` |
| **Archivo** | `src/app/@core/data/descuento_academico.service.ts` |
| **Métodos** | GET, POST, PUT, DELETE |
| **Descripción** | Descuentos y exoneraciones sobre derechos de matrícula. |
| **Respuesta** | Registros de descuento con porcentaje, tipo y estado. |

---

### 7. `documento_crud` — Documentos

| Campo | Detalle |
|-------|---------|
| **URL base** | `…/documento_crud/v2/` |
| **Archivo** | `src/app/@core/data/documento.service.ts` |
| **Métodos** | GET, POST, PUT, DELETE |
| **Descripción** | Metadatos de documentos institucionales. |
| **Respuesta** | Metadatos del documento (nombre, tipo, fecha, etc.). |

---

### 8. `documento_programa_crud` — Documentos de programa

| Campo | Detalle |
|-------|---------|
| **URL base (prod)** | `https://autenticacion.udistrital.edu.co/apioas/documento_programa_crud/v1/` |
| **URL base (dev)** | `http://api.planestic.udistrital.edu.co:9014/v1/` |
| **Archivo** | `src/app/@core/data/documento_programa.service.ts` |
| **Métodos** | GET, POST, PUT, DELETE |
| **Descripción** | Documentos asociados a planes de estudio y programas académicos. |
| **Respuesta** | Documentos vinculados a un programa con su versión y estado. |

---

### 9. `ente_crud` — Entidades

| Campo | Detalle |
|-------|---------|
| **URL base** | `…/ente_crud/v1/` |
| **Archivo** | `src/app/@core/data/ente.service.ts` |
| **Métodos** | GET, POST, PUT, DELETE |
| **Descripción** | Gestión de entidades / instituciones. |
| **Respuesta** | Registros de entidades con nombre, tipo y estado. |

---

### 10. `espacios_academicos_crud` — Espacios académicos

| Campo | Detalle |
|-------|---------|
| **URL base** | `…/espacios_academicos_crud/v1/` |
| **Archivo** | `src/app/@core/data/espacios_academicos.service.ts` |
| **Métodos** | GET, POST, PUT, DELETE |
| **Descripción** | Aulas, laboratorios y espacios físicos para clases. |
| **Respuesta** | Datos del espacio: nombre, capacidad, edificio, disponibilidad. |

---

### 11. `evaluacion_inscripcion_crud` — Evaluación de inscripción

| Campo | Detalle |
|-------|---------|
| **URL base (prod)** | `…/evaluacion_inscripcion_crud/v2/` |
| **URL base (dev)** | `http://pruebasapi2.intranetoas.udistrital.edu.co:8118/v1/` |
| **Archivo** | `src/app/@core/data/evaluacion_inscripcion.service.ts` |
| **Métodos** | GET, POST, PUT, DELETE |
| **Descripción** | Evaluación y calificación del proceso de inscripción de aspirantes. |
| **Respuesta** | Resultados de evaluación con puntajes y estados. |

---

### 12. `sesiones_crud` — Eventos / Sesiones

| Campo | Detalle |
|-------|---------|
| **URL base (prod)** | `…/sesiones_crud/v2/` |
| **URL base (dev)** | `http://pruebasapi2.intranetoas.udistrital.edu.co:8107/v1/` |
| **Archivo** | `src/app/@core/data/evento.service.ts` |
| **Métodos** | GET, POST, PUT, DELETE |
| **Descripción** | Gestión de eventos y sesiones de clase. También usado por `oferta_academica.service.ts`. |
| **Respuesta** | Datos de sesión: fecha, hora, docente, espacio, asignatura. |

---

### 13. `experiencia_laboral_crud` — Experiencia laboral

| Campo | Detalle |
|-------|---------|
| **URL base (prod)** | `…/experiencia_laboral_crud/v1/` |
| **URL base (dev)** | `http://api.planestic.udistrital.edu.co:8099/v1/` |
| **Archivo** | `src/app/@core/data/experiencia.service.ts` |
| **Métodos** | GET, POST, PUT, DELETE |
| **Descripción** | Hoja de vida laboral de docentes y funcionarios. |
| **Respuesta** | Registros de experiencia: empresa, cargo, fechas, descripción. |

---

### 14. `formacion_academica_crud` — Formación académica

| Campo | Detalle |
|-------|---------|
| **URL base (prod)** | `…/formacion_academica_crud/v1/` |
| **URL base (dev)** | `http://api.planestic.udistrital.edu.co:8098/v1/` |
| **Archivo** | `src/app/@core/data/formacion_academica.service.ts` |
| **Métodos** | GET, POST, PUT, DELETE |
| **Descripción** | Historial de títulos y estudios académicos. |
| **Respuesta** | Títulos obtenidos: institución, nivel, año de grado. |

---

### 15. `idiomas_crud` — Idiomas

| Campo | Detalle |
|-------|---------|
| **URL base** | `…/idiomas_crud/v2/` |
| **Archivo** | `src/app/@core/data/idioma.service.ts` |
| **Métodos** | GET, POST, PUT, DELETE |
| **Descripción** | Competencias en idiomas de personas (docentes / estudiantes). |
| **Respuesta** | Registros de idioma con nivel de suficiencia. |

---

### 16. `inscripcion_crud` — Inscripciones

| Campo | Detalle |
|-------|---------|
| **URL base (prod)** | `…/inscripcion_crud/v2/` |
| **URL base (dev)** | `http://pruebasapi2.intranetoas.udistrital.edu.co:8208/v1/` |
| **Archivo** | `src/app/@core/data/inscripcion.service.ts` |
| **Métodos** | GET, POST, PUT, DELETE |
| **Descripción** | Inscripción de aspirantes a programas académicos. |
| **Respuesta** | Registro de inscripción con estado, fechas y programa. |

---

### 17. `oikos_crud_api` — Oikos (lugares)

| Campo | Detalle |
|-------|---------|
| **URL base (prod)** | `…/oikos_crud_api/v1/` |
| **URL base (dev)** | `http://api.intranetoas.udistrital.edu.co:8087/v2/` |
| **Archivo** | `src/app/@core/data/oikos.service.ts` |
| **Métodos** | GET, POST, PUT, DELETE |
| **Descripción** | Estructura geográfica y organizacional: sedes, facultades, edificios. |
| **Respuesta** | Nodos del árbol organizacional con jerarquía. |

---

### 18. `organizacion_crud` — Organización

| Campo | Detalle |
|-------|---------|
| **URL base** | `…/organizacion_crud/v1/` |
| **Archivo** | `src/app/@core/data/organizacion.service.ts` |
| **Métodos** | GET, POST, PUT, DELETE |
| **Descripción** | Estructura departamental y organizacional de la institución. |
| **Respuesta** | Unidades organizacionales con jerarquía y tipo. |

---

### 19. `parametros` — Parámetros del sistema

| Campo | Detalle |
|-------|---------|
| **URL base (prod)** | `…/parametros/v1/` |
| **URL base (dev)** | `http://pruebasapi.intranetoas.udistrital.edu.co:8205/v1/` |
| **Archivo** | `src/app/@core/data/parametros.service.ts` |
| **Métodos** | GET, POST, PUT, DELETE |
| **Descripción** | Tablas de parámetros globales: tipos, estados, categorías, etc. |
| **Respuesta** | Pares clave-valor o listas de opciones de parámetro. |

---

### 20. `personas_crud` — Personas

| Campo | Detalle |
|-------|---------|
| **URL base (prod)** | `…/personas_crud/v1/` |
| **URL base (dev)** | `http://api.intranetoas.udistrital.edu.co:8083/v1` |
| **Archivo** | `src/app/@core/data/persona.service.ts` |
| **Métodos** | GET, POST, PUT, DELETE |
| **Descripción** | Información personal de estudiantes, docentes y funcionarios. |
| **Respuesta** | Objeto persona con datos de identificación, contacto y tipo. |

---

### 21. `plan_trabajo_docente_crud` — Plan de trabajo docente

| Campo | Detalle |
|-------|---------|
| **URL base** | `…/plan_trabajo_docente_crud/v1/` |
| **Archivo** | `src/app/@core/data/plan_trabajo_docente.service.ts` |
| **Métodos** | GET, POST, PUT, DELETE |
| **Descripción** | Planes de trabajo semestral de los docentes. |
| **Respuesta** | Plan con actividades, horas asignadas y estado de aprobación. |

---

### 22. `produccion_academica_crud` — Producción académica

| Campo | Detalle |
|-------|---------|
| **URL base (prod)** | `…/produccion_academica_crud/v2/` |
| **URL base (dev)** | `http://api.intranetoas.udistrital.edu.co:8121/v1/` |
| **Archivo** | `src/app/@core/data/produccion_academica.service.ts` |
| **Métodos** | GET, POST, PUT, DELETE |
| **Descripción** | Publicaciones, investigaciones y productos académicos de docentes. |
| **Respuesta** | Registros de producción con tipo, título, año y coautores. |

---

### 23. `proyecto_academico_crud` — Proyectos académicos

| Campo | Detalle |
|-------|---------|
| **URL base (prod)** | `…/proyecto_academico_crud/v1/` |
| **URL base (dev)** | `http://pruebasapi.intranetoas.udistrital.edu.co:8116/v1/` |
| **Archivo** | `src/app/@core/data/proyecto_academico.service.ts` |
| **Métodos** | GET, POST, PUT, DELETE |
| **Descripción** | Proyectos curriculares y planes de estudio por programa. |
| **Respuesta** | Datos del proyecto: nombre, código, facultad, créditos, estado. |

---

### 24. `terceros_crud` — Terceros

| Campo | Detalle |
|-------|---------|
| **URL base (prod)** | `…/terceros_crud/v1/` |
| **URL base (dev)** | `http://pruebasapi.intranetoas.udistrital.edu.co:8121/v1/` |
| **Archivo** | `src/app/@core/data/terceros.service.ts` |
| **Métodos** | GET, POST, PUT, DELETE |
| **Descripción** | Gestión de terceros: empresas, entidades externas, proveedores. |
| **Endpoints clave** | `datos_identificacion?query=Activo:true,Numero:{doc}` · `tercero?query=Activo:True,UsuarioWSO2:{email}` |
| **Respuesta** | Objeto de tercero con identificación, tipo y datos de contacto. |

---

### 25. `ubicaciones_crud` — Ubicaciones

| Campo | Detalle |
|-------|---------|
| **URL base (prod)** | `…/ubicaciones_crud/v2/` |
| **URL base (dev)** | `http://pruebasapi2.intranetoas.udistrital.edu.co:8085/v1/` |
| **Archivo** | `src/app/@core/data/ubicacion.service.ts` |
| **Métodos** | GET, POST, PUT, DELETE |
| **Descripción** | Edificios, bloques y ubicaciones físicas del campus. |
| **Respuesta** | Datos de ubicación con jerarquía (sede → edificio → aula). |

---

## Servicios Middleware (MID)

### 26. `sga_mid` — Middleware principal SGA

| Campo | Detalle |
|-------|---------|
| **URL base (prod)** | `…/sga_mid/v1/` |
| **URL base (dev)** | `http://localhost:8119/v1/` / `http://pruebasapi.intranetoas.udistrital.edu.co:8119/v1/` |
| **Archivo** | `src/app/@core/data/sga_mid.service.ts` |
| **Métodos** | GET, POST, POST_FILE, PUT, DELETE |
| **Descripción** | Middleware central del SGA. Orquesta lógica de negocio compleja que no corresponde a un solo CRUD. Incluye gestión de prácticas académicas. |
| **Respuesta** | Varía según endpoint; generalmente objetos de proceso con estado. |

---

### 27. `campus_mid` — Middleware de campus

| Campo | Detalle |
|-------|---------|
| **URL base (dev)** | `http://localhost:8095/v1/` |
| **Archivo** | `src/app/@core/data/campus_mid.service.ts` |
| **Métodos** | GET, POST, PUT, DELETE |
| **Descripción** | Middleware de gestión del campus físico. |
| **Respuesta** | Datos de campus procesados y combinados. |

---

### 28. `google_mid` — Middleware de Google

| Campo | Detalle |
|-------|---------|
| **URL base (prod)** | `…/google_mid/v1/` |
| **URL base (dev)** | `http://pruebasapi.intranetoas.udistrital.edu.co:8514/v1/` |
| **Archivo** | `src/app/@core/data/google.service.ts` |
| **Métodos** | GET, POST, POST_FILE, PUT, DELETE |
| **Descripción** | Integración con servicios de Google (Calendar, Forms, etc.). |
| **Respuesta** | Objetos de Google Calendar / Forms con IDs y URLs de acceso. |

---

## Servicios especiales

### 29. Nuxeo — Gestión documental

| Campo | Detalle |
|-------|---------|
| **URL directa** | `https://documental.portaloas.udistrital.edu.co/nuxeo/` |
| **URL middleware** | `…/gestor_documental_mid/v1` |
| **Archivos** | `src/app/@core/utils/nuxeo.service.ts` · `src/app/@core/utils/new_nuxeo.service.ts` |
| **Autenticación** | Basic Auth (usuario/contraseña en entorno) |
| **Operaciones** | `Document.Create`, `Blob.AttachOnDocument`, `Batch Upload` |
| **Endpoints clave** | `/document` · `/document/{uuid}` · `/document/uploadAnyFormat` |
| **Descripción** | Repositorio de archivos institucional. Permite subir, consultar y eliminar documentos. |
| **Respuesta** | UUID del documento, URL de descarga, metadatos del archivo. |

---

### 30. `recibo` — Recibos de pago

| Campo | Detalle |
|-------|---------|
| **URL base** | `http://api.planestic.udistrital.edu.co:9017/v1/` |
| **Archivo** | `src/app/@core/data/recibo.service.ts` |
| **Métodos** | GET, POST, PUT, DELETE |
| **Descripción** | Generación y consulta de recibos de pago de derechos pecuniarios. |
| **Respuesta** | Recibo con número, valor, fecha de vencimiento y estado. |

---

### 31. `pago` — Pagos internos

| Campo | Detalle |
|-------|---------|
| **URL base** | `http://prueba.campusvirtual.udistrital.edu.co/pagos/` |
| **Archivo** | `src/app/@core/data/pago.service.ts` |
| **Métodos** | GET, POST, PUT, DELETE |
| **Descripción** | Registro y gestión interna de pagos. |
| **Respuesta** | Estado del pago con referencia y confirmación. |

---

### 32. SpagoBI / Knowage — Business Intelligence

| Campo | Detalle |
|-------|---------|
| **URL base** | `https://inteligenciainstitucional.portaloas.udistrital.edu.co:443/knowage` |
| **Descripción** | Generación de reportes y analítica institucional. |
| **Notas** | Credenciales configuradas en el entorno. Solo para visualización de reportes. |

---

## Comunicación en tiempo real

### 33. WebSocket — Notificaciones

| Campo | Detalle |
|-------|---------|
| **URL** | `wss://pruebasapi.portaloas.udistrital.edu.co:8116/ws?id={userId}&profiles={roles}` |
| **Archivo** | `src/app/@core/utils/notificaciones.service.ts` |
| **Protocolo** | WebSocket (wss://) |
| **Descripción** | Entrega de notificaciones en tiempo real al usuario. Mantiene la conexión viva con un ping cada 30 segundos. |
| **Endpoints REST asociados** | |
| `GET /notificacion_estado_usuario` | Consulta notificaciones con filtros |
| `POST /notificacion_estado_usuario/changeStateNoView/{user}` | Marca notificaciones como vistas |
| `PUT /notificacion_estado_usuario` | Actualiza estado de notificación |

---

## Pasarela de pagos

### 34. PSE — Botón de pago

| Campo | Detalle |
|-------|---------|
| **URL (prod)** | `https://funcionarios.portaloas.udistrital.edu.co/botonPago/index.php?` |
| **URL (test)** | `https://pruebasfuncionarios.portaloas.udistrital.edu.co/botonPago/index.php?` |
| **Tipo** | Redirección HTTP (no servicio Angular) |
| **Descripción** | Pasarela de pago PSE para Colombia. El frontend redirige al usuario a esta URL con los parámetros del recibo. |

---

## Patrones HTTP comunes

### Métodos del `RequestManager`

| Método | Firma | Notas |
|--------|-------|-------|
| `get(endpoint)` | `Observable<any>` | Extrae `Body` si existe en la respuesta |
| `post(endpoint, obj)` | `Observable<any>` | Envía JSON |
| `post_file(endpoint, file)` | `Observable<any>` | Multipart upload |
| `put(endpoint, obj)` | `Observable<any>` | Añade `obj.Id` al path automáticamente |
| `delete(endpoint, id)` | `Observable<any>` | Añade `id` al path |
| `get_soap(endpoint)` | `Observable<any>` | Cabeceras `multipart/form-data` |

### Filtros de consulta (query string)

```
?query=Activo:true,Campo:valor
&sortby=FechaCreacion&order=desc
&limit=100&offset=0
```

### Servicio genérico — `AnyService`

`src/app/@core/data/any.service.ts`  
Cliente HTTP genérico para paths dinámicos. Útil cuando el servicio de destino no tiene un archivo `.service.ts` propio. Métodos: `get()`, `getp()` (con progreso), `post()`, `put()`, `delete()`.

---

## Resumen

| Estadística | Valor |
|-------------|-------|
| Servicios de backend distintos | 34+ |
| Métodos HTTP soportados | GET, POST, PUT, DELETE, WebSocket |
| Formato de datos | JSON (principalmente), multipart (archivos) |
| Autenticación | OAuth 2.0 OIDC + Bearer token |
| Dominios principales | `portaloas.udistrital.edu.co`, `planestic.udistrital.edu.co`, `intranetoas.udistrital.edu.co` |
| Configuración por entorno | Sí (dev / test / prod) |
