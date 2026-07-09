package controllers

import (
	"github.com/beego/beego/v2/server/web"
	"github.com/udistrital/egresados_service/helpers"
	"github.com/udistrital/egresados_service/services"
)

type CatalogosController struct{ web.Controller }

// @Title GetCategorias
// @Description Catálogo de categorías de beneficio (servicio institucional de parámetros, o local si ParametrosLocal=true).
// @Success 200 {object} helpers.APIResponse
// @Failure 500 error interno
// @router /categorias-beneficio [get]
func (c *CatalogosController) GetCategorias() {
	result, err := services.GetCategoriasBeneficio(c.Ctx.Input.Header("Authorization"))
	if err != nil {
		helpers.InternalError(&c.Controller, err)
		return
	}
	helpers.Ok(&c.Controller, result)
}

// @Title GetSectores
// @Description Catálogo de sectores económicos (servicio institucional de parámetros, o local si ParametrosLocal=true).
// @Success 200 {object} helpers.APIResponse
// @Failure 500 error interno
// @router /sectores-economicos [get]
func (c *CatalogosController) GetSectores() {
	result, err := services.GetSectoresEconomicos(c.Ctx.Input.Header("Authorization"))
	if err != nil {
		helpers.InternalError(&c.Controller, err)
		return
	}
	helpers.Ok(&c.Controller, result)
}
