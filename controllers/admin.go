package controllers

import (
	"github.com/beego/beego/v2/server/web"
	"github.com/udistrital/sga_mid_beneficios_egresados/helpers"
	"github.com/udistrital/sga_mid_beneficios_egresados/services"
)

type AdminController struct{ web.Controller }

// SuspenderEmpresa PUT /v1/empresas/:id/suspender
func (c *AdminController) SuspenderEmpresa() {
	id, err := c.GetInt(":id")
	if err != nil {
		helpers.BadRequest(&c.Controller, "id inválido")
		return
	}

	if err := services.SuspenderEmpresa(c.Ctx.Input.Header("Authorization"), id); err != nil {
		helpers.UnprocessableEntity(&c.Controller, err.Error())
		return
	}
	helpers.Ok(&c.Controller, "empresa suspendida")
}
