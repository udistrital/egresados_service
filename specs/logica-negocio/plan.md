# Plan — Trabajo pendiente del MID

> Última actualización: 2026-07-08. Cubre los pendientes 1 y 2 de `tasks.md`
> (los que tienen diseño no trivial); el resto son tareas directas.

## 1. Enriquecer la bandeja de empresa (RF-006)

**Meta:** que cada ítem de la bandeja traiga programa/facultad (y correo, si el
revisor lo aprueba) del egresado, además del nombre+código actuales.

**Enfoque:**
1. En `GetBandejaEmpresa` (`solicitudes_service.go`), por cada egresado DISTINTO de la página: resolver `TerceroId` por documento (`terceros_crud/datos_identificacion?query=Activo:true,Numero:{doc}`) → `sga_mid/consultar_persona/{id}` → si trae `Codigos[]`, tomar proyecto del código activo=false.
2. **Caché por documento dentro del request** (map local): una bandeja con N solicitudes del mismo egresado no repite la cadena.
3. **Degradación:** si la cadena falla o viene plana (sin `Codigos`), derivar carrera/facultad de los dígitos [5:8] del `codigo_institucional` (portar el diccionario `programas-ud.ts` del front o dejar que el front siga aplicando su fallback — decisión: dejarlo en el front, el código ya viaja en el ítem; el MID solo agrega lo institucional).
4. **Correo:** viene de `usuario.correo` local (cero llamadas). NO exponerlo hasta validar RNF-002b con el revisor; el hilo de mensajes ya cubre la comunicación dentro del módulo.

**Riesgo:** +2 llamadas institucionales por egresado distinto; aceptable con caché
y `limit` de página. Verificar el tiempo de respuesta con la bandeja real.

## 2. Notificaciones de cambio de estado (RN-005)

**Meta:** avisar al egresado cuando la empresa responde (aprobada/rechazada/
requiere info) y a la empresa cuando llega solicitud o respuesta del egresado.

**Enfoque:**
1. Evaluar `NotificacionesMid` (`/notificacion_mid/v1`) vs `Notificaciones` (`/notificaciones_crud/v1`) en el API Store: contrato, canales (correo/portal), y si aceptan el Bearer de usuario propagado (mismo patrón del resto del gateway).
2. Punto de integración: un solo helper `notificar(token, evento, solicitud)` llamado desde `ResponderSolicitud`, `EnviarMensaje` y `CrearSolicitud` — best-effort (el fallo de la notificación NUNCA revierte la transición).
3. Plantillas mínimas: radicado + beneficio + nuevo estado + enlace profundo (`/solicitudes?radicado=`, el front ya soporta el deep-link).

**Bloqueante potencial:** que el servicio exija un rol/scope que no tengamos
(relacionado con D-8). Verificar con un token vivo antes de diseñar más.
