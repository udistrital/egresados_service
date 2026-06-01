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

	result, err := services.CrearSolicitud(body)
	if err != nil {
		helpers.UnprocessableEntity(&c.Controller, err.Error())
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

	result, err := services.GetSolicitudesByEgresado(egresadoId)
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

	// El egresado que cancela debe venir en el body o del token JWT
	var body map[string]interface{}
	json.Unmarshal(c.Ctx.Input.RequestBody, &body)

	if err := services.CancelarSolicitud(id, body); err != nil {
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

	result, err := services.GetResumenEgresado(egresadoId)
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

	var body map[string]interface{}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &body); err != nil {
		helpers.BadRequest(&c.Controller, "cuerpo de solicitud inválido")
		return
	}

	if err := services.ResponderSolicitud(id, body); err != nil {
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

	var body map[string]interface{}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &body); err != nil {
		helpers.BadRequest(&c.Controller, "cuerpo de solicitud inválido")
		return
	}

	result, err := services.EnviarMensaje(id, body)
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

	result, err := services.GetMensajes(id)
	if err != nil {
		helpers.InternalError(&c.Controller, err)
		return
	}
	helpers.Ok(&c.Controller, result)
}
