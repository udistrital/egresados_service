package controllers

import (
	"encoding/json"

	"github.com/beego/beego/v2/server/web"
	"github.com/udistrital/sga_mid_beneficios_egresados/helpers"
	"github.com/udistrital/sga_mid_beneficios_egresados/services"
)

type SolicitudesController struct{ web.Controller }

// Crear POST /v1/solicitudes
// Crea una solicitud. Valida: límite activo (RN-010), cupo atómico (RN-002b),
// solicitud única por (egresado, beneficio) (RN-007), genera radicado (RN-RADICADO).
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

// GetByEgresado GET /v1/solicitudes/egresado/:egresado_id
// Lista de solicitudes del egresado con estado e historial.
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

// Cancelar PUT /v1/solicitudes/:id/cancelar
// Cancelar solicitud. Solo si estado = PENDIENTE o REQUIERE_INFO (RN-005). Devuelve cupo (RN-002c).
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

// Resumen GET /v1/solicitudes/egresado/:egresado_id/resumen
// Contadores por estado (activas, aprobadas, rechazadas, canceladas).
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

// Responder PUT /v1/solicitudes/:id/responder
// Aprobar / Rechazar / Requiere información (empresa).
// Rechazar requiere justificacion >= 20 chars (RN-003).
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

// EnviarMensaje POST /v1/solicitudes/:id/mensajes
// Enviar mensaje (solo cuando estado = REQUIERE_INFO).
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

// GetMensajes GET /v1/solicitudes/:id/mensajes
// Historial de mensajes de la solicitud.
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

// GetHistorial GET /v1/solicitudes/:id/historial
// Bitácora de estados (C-4b) para el drawer de detalle, de ambas partes.
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

// GetDocumentos GET /v1/solicitudes/:id/documentos
// Documentos requeridos del beneficio vs. subidos por el egresado (merge), para
// que tanto el egresado (qué le falta) como la empresa (qué revisar) vean lo mismo.
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

// SubirDocumento POST /v1/solicitudes/:id/documentos
// El egresado sube (o reemplaza) el PDF de un documento requerido. Body:
// { documento_requerido_id, nombre_archivo, file (base64, PDF) }.
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

// EliminarDocumento DELETE /v1/solicitudes/:id/documentos/:doc_id
// El egresado quita un documento que había subido.
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

// ComentarDocumento PUT /v1/documentos/:doc_id/comentario
// La empresa deja una observación sobre un documento subido por el egresado.
// Body: { comentario }.
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

// GetComprobante GET /v1/solicitudes/:id/comprobante
// Comprobante opcional que la empresa adjuntó al aprobar. { tiene_comprobante,
// nombre_archivo?, file? (base64) }.
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

// GetArchivoDocumento GET /v1/documentos/:doc_id/archivo
// Proxy de solo lectura hacia el gestor documental: el cliente nunca llama a ese
// servicio directamente. Devuelve { nombre_archivo, file (base64) }.
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
