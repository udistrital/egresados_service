package controllers

import (
	"github.com/beego/beego/v2/server/web"
	"github.com/udistrital/egresados_service/helpers"
	"github.com/udistrital/egresados_service/services"
)

type EgresadosController struct{ web.Controller }

// @Title Provisionar
// @Description JIT provisioning del egresado al primer login (contraparte del de empresa): resuelve su identidad desde el token (OIDC userinfo → userRol → terceros_crud) y da de alta usuario/egresado locales. No recibe body: la identidad sale del token.
// @Success 200 {object} helpers.APIResponse
// @Failure 422 no se pudo resolver la identidad del egresado
// @router /egresados/provision [post]
func (c *EgresadosController) Provisionar() {
	result, err := services.ProvisionarEgresado(c.Ctx.Input.Header("Authorization"))
	if err != nil {
		helpers.UnprocessableEntity(&c.Controller, err.Error())
		return
	}
	helpers.Ok(&c.Controller, result)
}
