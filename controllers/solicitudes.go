package controllers

import (
	"encoding/json"

	"github.com/beego/beego/v2/server/web"
	"github.com/udistrital/egresados_service/helpers"
	"github.com/udistrital/egresados_service/services"
)

type SolicitudesController struct{ web.Controller }

// @Title Crear
// @Description Crea una solicitud. Valida: límite activo (RN-010), cupo atómico (RN-002b), solicitud única por (egresado, beneficio) (RN-007), genera radicado (RN-RADICADO). El egresado_id del body debe ser el del token (anti-IDOR).
// @Param   body    body    string    true    "JSON con beneficio_id y demás datos de la solicitud"
// @Success 201 {object} helpers.APIResponse
// @Failure 400 body inválido
// @Failure 403 el egresado_id del body no coincide con el del token
// @Failure 422 regla de negocio violada (límite activo, cupo, solicitud duplicada)
// @router /solicitudes [post]
func (c *SolicitudesController) Crear() {
	var body map[string]interface{}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &body); err != nil {
		helpers.BadRequest(&c.Controller, "cuerpo de solicitud inválido")
		return
	}

	// El servicio valida que el egresado_id del body sea el del token (anti-IDOR).
	result, err := services.CrearSolicitud(c.Ctx.Input.Header("Authorization"), body)
	if err != nil {
		responderErrorNegocio(&c.Controller, err)
		return
	}
	helpers.Created(&c.Controller, result)
}

// @Title GetByEgresado
// @Description Lista de solicitudes del egresado con estado e historial.
// @Param   egresado_id    path    int    true    "id del egresado"
// @Success 200 {object} helpers.APIResponse
// @Failure 400 egresado_id inválido
// @Failure 403 el egresado_id no coincide con el del token
// @Failure 500 error interno
// @router /solicitudes/egresado/:egresado_id [get]
func (c *SolicitudesController) GetByEgresado() {
	egresadoId, err := c.GetInt(":egresado_id")
	if err != nil {
		helpers.BadRequest(&c.Controller, "egresado_id inválido")
		return
	}

	token := c.Ctx.Input.Header("Authorization")
	if err := services.VerificarEgresadoDelToken(token, egresadoId); err != nil {
		responderErrorAcceso(&c.Controller, err)
		return
	}

	result, err := services.GetSolicitudesByEgresado(token, egresadoId)
	if err != nil {
		helpers.InternalError(&c.Controller, err)
		return
	}
	helpers.Ok(&c.Controller, result)
}

// @Title Cancelar
// @Description Cancelar solicitud. Solo si estado = PENDIENTE o REQUIERE_INFO (RN-005). Devuelve cupo (RN-002c).
// @Param   id      path    int       true    "id de la solicitud"
// @Param   body    body    string    false   "JSON opcional con justificación"
// @Success 200 {object} helpers.APIResponse
// @Failure 400 id inválido
// @Failure 403 el usuario del token no es dueño de la solicitud
// @Failure 422 la solicitud no está en un estado cancelable
// @router /solicitudes/:id/cancelar [put]
func (c *SolicitudesController) Cancelar() {
	id, err := c.GetInt(":id")
	if err != nil {
		helpers.BadRequest(&c.Controller, "id inválido")
		return
	}

	token := c.Ctx.Input.Header("Authorization")
	if err := services.VerificarAccesoSolicitudEgresado(token, id); err != nil {
		responderErrorAcceso(&c.Controller, err)
		return
	}

	var body map[string]interface{}
	json.Unmarshal(c.Ctx.Input.RequestBody, &body)

	if err := services.CancelarSolicitud(token, id, body); err != nil {
		helpers.UnprocessableEntity(&c.Controller, err.Error())
		return
	}
	helpers.Ok(&c.Controller, "solicitud cancelada")
}

// @Title Resumen
// @Description Contadores por estado (activas, aprobadas, rechazadas, canceladas).
// @Param   egresado_id    path    int    true    "id del egresado"
// @Success 200 {object} helpers.APIResponse
// @Failure 400 egresado_id inválido
// @Failure 403 el egresado_id no coincide con el del token
// @Failure 500 error interno
// @router /solicitudes/egresado/:egresado_id/resumen [get]
func (c *SolicitudesController) Resumen() {
	egresadoId, err := c.GetInt(":egresado_id")
	if err != nil {
		helpers.BadRequest(&c.Controller, "egresado_id inválido")
		return
	}

	token := c.Ctx.Input.Header("Authorization")
	if err := services.VerificarEgresadoDelToken(token, egresadoId); err != nil {
		responderErrorAcceso(&c.Controller, err)
		return
	}

	result, err := services.GetResumenEgresado(token, egresadoId)
	if err != nil {
		helpers.InternalError(&c.Controller, err)
		return
	}
	helpers.Ok(&c.Controller, result)
}

// @Title Responder
// @Description Aprobar / Rechazar / Requiere información (empresa). Rechazar requiere justificacion >= 20 chars (RN-003).
// @Param   id      path    int       true    "id de la solicitud"
// @Param   body    body    string    true    "JSON con la decisión (aprobar/rechazar/requiere_info) y justificación"
// @Success 200 {object} helpers.APIResponse
// @Failure 400 id o body inválido
// @Failure 403 el usuario del token no es dueño de la empresa de la solicitud
// @Failure 422 justificación insuficiente u otra regla de negocio (RN-003)
// @router /solicitudes/:id/responder [put]
func (c *SolicitudesController) Responder() {
	id, err := c.GetInt(":id")
	if err != nil {
		helpers.BadRequest(&c.Controller, "id inválido")
		return
	}

	token := c.Ctx.Input.Header("Authorization")
	if err := services.VerificarAccesoSolicitudEmpresa(token, id); err != nil {
		responderErrorAcceso(&c.Controller, err)
		return
	}

	var body map[string]interface{}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &body); err != nil {
		helpers.BadRequest(&c.Controller, "cuerpo de solicitud inválido")
		return
	}

	if err := services.ResponderSolicitud(token, id, body); err != nil {
		helpers.UnprocessableEntity(&c.Controller, err.Error())
		return
	}
	helpers.Ok(&c.Controller, "solicitud actualizada")
}

// @Title EnviarMensaje
// @Description Enviar mensaje (solo cuando estado = REQUIERE_INFO).
// @Param   id      path    int       true    "id de la solicitud"
// @Param   body    body    string    true    "JSON con el texto del mensaje"
// @Success 201 {object} helpers.APIResponse
// @Failure 400 id o body inválido
// @Failure 403 el usuario del token no participa de la solicitud
// @Failure 422 la solicitud no admite mensajes en su estado actual
// @router /solicitudes/:id/mensajes [post]
func (c *SolicitudesController) EnviarMensaje() {
	id, err := c.GetInt(":id")
	if err != nil {
		helpers.BadRequest(&c.Controller, "id inválido")
		return
	}

	token := c.Ctx.Input.Header("Authorization")
	if err := services.VerificarParticipanteSolicitud(token, id); err != nil {
		responderErrorAcceso(&c.Controller, err)
		return
	}

	var body map[string]interface{}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &body); err != nil {
		helpers.BadRequest(&c.Controller, "cuerpo de solicitud inválido")
		return
	}

	result, err := services.EnviarMensaje(token, id, body)
	if err != nil {
		helpers.UnprocessableEntity(&c.Controller, err.Error())
		return
	}
	helpers.Created(&c.Controller, result)
}

// @Title GetMensajes
// @Description Historial de mensajes de la solicitud.
// @Param   id    path    int    true    "id de la solicitud"
// @Success 200 {object} helpers.APIResponse
// @Failure 400 id inválido
// @Failure 403 el usuario del token no participa de la solicitud
// @Failure 500 error interno
// @router /solicitudes/:id/mensajes [get]
func (c *SolicitudesController) GetMensajes() {
	id, err := c.GetInt(":id")
	if err != nil {
		helpers.BadRequest(&c.Controller, "id inválido")
		return
	}

	token := c.Ctx.Input.Header("Authorization")
	if err := services.VerificarParticipanteSolicitud(token, id); err != nil {
		responderErrorAcceso(&c.Controller, err)
		return
	}

	result, err := services.GetMensajes(token, id)
	if err != nil {
		helpers.InternalError(&c.Controller, err)
		return
	}
	helpers.Ok(&c.Controller, result)
}

// @Title GetHistorial
// @Description Bitácora de estados (C-4b) para el drawer de detalle, de ambas partes.
// @Param   id    path    int    true    "id de la solicitud"
// @Success 200 {object} helpers.APIResponse
// @Failure 400 id inválido
// @Failure 403 el usuario del token no participa de la solicitud
// @Failure 500 error interno
// @router /solicitudes/:id/historial [get]
func (c *SolicitudesController) GetHistorial() {
	id, err := c.GetInt(":id")
	if err != nil {
		helpers.BadRequest(&c.Controller, "id inválido")
		return
	}

	token := c.Ctx.Input.Header("Authorization")
	if err := services.VerificarParticipanteSolicitud(token, id); err != nil {
		responderErrorAcceso(&c.Controller, err)
		return
	}

	result, err := services.GetHistorialSolicitud(token, id)
	if err != nil {
		helpers.InternalError(&c.Controller, err)
		return
	}
	helpers.Ok(&c.Controller, result)
}

// @Title GetDocumentos
// @Description Documentos requeridos del beneficio vs. subidos por el egresado (merge), para que tanto el egresado (qué le falta) como la empresa (qué revisar) vean lo mismo.
// @Param   id    path    int    true    "id de la solicitud"
// @Success 200 {object} helpers.APIResponse
// @Failure 400 id inválido
// @Failure 403 el usuario del token no participa de la solicitud
// @Failure 500 error interno
// @router /solicitudes/:id/documentos [get]
func (c *SolicitudesController) GetDocumentos() {
	id, err := c.GetInt(":id")
	if err != nil {
		helpers.BadRequest(&c.Controller, "id inválido")
		return
	}

	token := c.Ctx.Input.Header("Authorization")
	if err := services.VerificarParticipanteSolicitud(token, id); err != nil {
		responderErrorAcceso(&c.Controller, err)
		return
	}

	result, err := services.GetDocumentosDeSolicitud(token, id)
	if err != nil {
		helpers.InternalError(&c.Controller, err)
		return
	}
	helpers.Ok(&c.Controller, result)
}

// @Title SubirDocumento
// @Description El egresado sube (o reemplaza) el PDF de un documento requerido.
// @Param   id      path    int       true    "id de la solicitud"
// @Param   body    body    string    true    "JSON { documento_requerido_id, nombre_archivo, file (base64, PDF) }"
// @Success 201 {object} helpers.APIResponse
// @Failure 400 id o body inválido
// @Failure 403 el usuario del token no es dueño de la solicitud
// @Failure 422 archivo inválido u otra regla de negocio
// @router /solicitudes/:id/documentos [post]
func (c *SolicitudesController) SubirDocumento() {
	id, err := c.GetInt(":id")
	if err != nil {
		helpers.BadRequest(&c.Controller, "id inválido")
		return
	}

	token := c.Ctx.Input.Header("Authorization")
	if err := services.VerificarAccesoSolicitudEgresado(token, id); err != nil {
		responderErrorAcceso(&c.Controller, err)
		return
	}

	var body map[string]interface{}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &body); err != nil {
		helpers.BadRequest(&c.Controller, "cuerpo de solicitud inválido")
		return
	}

	result, err := services.SubirDocumentoSolicitud(token, id, body)
	if err != nil {
		helpers.UnprocessableEntity(&c.Controller, err.Error())
		return
	}
	helpers.Created(&c.Controller, result)
}

// @Title EliminarDocumento
// @Description El egresado quita un documento que había subido.
// @Param   id        path    int    true    "id de la solicitud"
// @Param   doc_id    path    int    true    "id del documento"
// @Success 200 {object} helpers.APIResponse
// @Failure 400 id o doc_id inválido
// @Failure 403 el usuario del token no es dueño de la solicitud
// @Failure 422 el documento no se puede eliminar en el estado actual
// @router /solicitudes/:id/documentos/:doc_id [delete]
func (c *SolicitudesController) EliminarDocumento() {
	id, err := c.GetInt(":id")
	if err != nil {
		helpers.BadRequest(&c.Controller, "id inválido")
		return
	}
	docId, err := c.GetInt(":doc_id")
	if err != nil {
		helpers.BadRequest(&c.Controller, "doc_id inválido")
		return
	}

	token := c.Ctx.Input.Header("Authorization")
	if err := services.VerificarAccesoSolicitudEgresado(token, id); err != nil {
		responderErrorAcceso(&c.Controller, err)
		return
	}

	if err := services.EliminarDocumentoSolicitud(token, id, docId); err != nil {
		helpers.UnprocessableEntity(&c.Controller, err.Error())
		return
	}
	helpers.Ok(&c.Controller, "documento eliminado")
}

// @Title ComentarDocumento
// @Description La empresa deja una observación sobre un documento subido por el egresado.
// @Param   doc_id    path    int       true    "id del documento"
// @Param   body      body    string    true    "JSON { comentario }"
// @Success 200 {object} helpers.APIResponse
// @Failure 400 doc_id o body inválido
// @Failure 403 el usuario del token no tiene acceso a ese documento
// @Failure 422 error de negocio
// @router /documentos/:doc_id/comentario [put]
func (c *SolicitudesController) ComentarDocumento() {
	docId, err := c.GetInt(":doc_id")
	if err != nil {
		helpers.BadRequest(&c.Controller, "doc_id inválido")
		return
	}

	token := c.Ctx.Input.Header("Authorization")
	if err := services.VerificarAccesoDocumentoEmpresa(token, docId); err != nil {
		responderErrorAcceso(&c.Controller, err)
		return
	}

	var body map[string]interface{}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &body); err != nil {
		helpers.BadRequest(&c.Controller, "cuerpo de solicitud inválido")
		return
	}
	comentario, _ := body["comentario"].(string)

	if err := services.ComentarDocumento(token, docId, comentario); err != nil {
		helpers.UnprocessableEntity(&c.Controller, err.Error())
		return
	}
	helpers.Ok(&c.Controller, "comentario guardado")
}

// @Title GetComprobante
// @Description Comprobante opcional que la empresa adjuntó al aprobar. { tiene_comprobante, nombre_archivo?, file? (base64) }.
// @Param   id    path    int    true    "id de la solicitud"
// @Success 200 {object} helpers.APIResponse
// @Failure 400 id inválido
// @Failure 403 el usuario del token no participa de la solicitud
// @Failure 500 error interno
// @router /solicitudes/:id/comprobante [get]
func (c *SolicitudesController) GetComprobante() {
	id, err := c.GetInt(":id")
	if err != nil {
		helpers.BadRequest(&c.Controller, "id inválido")
		return
	}

	token := c.Ctx.Input.Header("Authorization")
	if err := services.VerificarParticipanteSolicitud(token, id); err != nil {
		responderErrorAcceso(&c.Controller, err)
		return
	}

	result, err := services.GetComprobanteSolicitud(token, id)
	if err != nil {
		helpers.InternalError(&c.Controller, err)
		return
	}
	helpers.Ok(&c.Controller, result)
}

// @Title GetArchivoDocumento
// @Description Proxy de solo lectura hacia el gestor documental: el cliente nunca llama a ese servicio directamente. Devuelve { nombre_archivo, file (base64) }.
// @Param   doc_id    path    int    true    "id del documento"
// @Success 200 {object} helpers.APIResponse
// @Failure 400 doc_id inválido
// @Failure 403 el usuario del token no participa del documento
// @Failure 500 error interno
// @router /documentos/:doc_id/archivo [get]
func (c *SolicitudesController) GetArchivoDocumento() {
	docId, err := c.GetInt(":doc_id")
	if err != nil {
		helpers.BadRequest(&c.Controller, "doc_id inválido")
		return
	}

	token := c.Ctx.Input.Header("Authorization")
	if err := services.VerificarParticipanteDocumento(token, docId); err != nil {
		responderErrorAcceso(&c.Controller, err)
		return
	}

	result, err := services.GetArchivoDocumento(token, docId)
	if err != nil {
		helpers.InternalError(&c.Controller, err)
		return
	}
	helpers.Ok(&c.Controller, result)
}
