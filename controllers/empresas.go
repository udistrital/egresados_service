package controllers

import (
	"encoding/json"

	"github.com/beego/beego/v2/server/web"
	"github.com/udistrital/sga_mid_beneficios_egresados/helpers"
	"github.com/udistrital/sga_mid_beneficios_egresados/services"
)

type EmpresasController struct{ web.Controller }

// Registrar POST /v1/empresas
// Registrar empresa. Estado inicial = EN_REVISION. Valida contra Ágora.
func (c *EmpresasController) Registrar() {
	var body map[string]interface{}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &body); err != nil {
		helpers.BadRequest(&c.Controller, "cuerpo de solicitud inválido")
		return
	}

	result, err := services.RegistrarEmpresa(body)
	if err != nil {
		helpers.UnprocessableEntity(&c.Controller, err.Error())
		return
	}
	helpers.Created(&c.Controller, result)
}

// GetBandeja GET /v1/empresas/:empresa_id/solicitudes
// Bandeja de solicitudes recibidas. Solo campos mínimos del egresado (RNF-002b / Ley 1581).
func (c *EmpresasController) GetBandeja() {
	empresaId, err := c.GetInt(":empresa_id")
	if err != nil {
		helpers.BadRequest(&c.Controller, "empresa_id inválido")
		return
	}

	result, err := services.GetBandejaEmpresa(empresaId)
	if err != nil {
		helpers.InternalError(&c.Controller, err)
		return
	}
	helpers.Ok(&c.Controller, result)
}
