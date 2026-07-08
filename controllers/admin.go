package controllers

import (
	"github.com/beego/beego/v2/server/web"
	"github.com/udistrital/egresados_service/helpers"
	"github.com/udistrital/egresados_service/services"
)

type AdminController struct{ web.Controller }

// @Title SuspenderEmpresa
// @Description Suspende una empresa (pasa de ACTIVA a SUSPENDIDA).
// @Param   id    path    int    true    "id de la empresa"
// @Success 200 {object} helpers.APIResponse
// @Failure 400 id inválido
// @Failure 422 la empresa no se puede suspender en su estado actual
// @router /empresas/:id/suspender [put]
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
