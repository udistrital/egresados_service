package controllers

import (
	"github.com/beego/beego/v2/server/web"
	"github.com/udistrital/sga_mid_beneficios_egresados/helpers"
	"github.com/udistrital/sga_mid_beneficios_egresados/services"
)

type EgresadosController struct{ web.Controller }

// Provisionar POST /v1/egresados/provision
// JIT provisioning del egresado al primer login (contraparte del de empresa):
// resuelve su identidad desde el token (OIDC userinfo → userRol → terceros_crud) y da
// de alta usuario/egresado locales. No recibe body: la identidad sale del token.
func (c *EgresadosController) Provisionar() {
	result, err := services.ProvisionarEgresado(c.Ctx.Input.Header("Authorization"))
	if err != nil {
		helpers.UnprocessableEntity(&c.Controller, err.Error())
		return
	}
	helpers.Ok(&c.Controller, result)
}
