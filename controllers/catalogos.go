package controllers

import (
	"github.com/beego/beego/v2/server/web"
	"github.com/udistrital/sga_mid_beneficios_egresados/helpers"
	"github.com/udistrital/sga_mid_beneficios_egresados/services"
)

type CatalogosController struct{ web.Controller }

// GetCategorias GET /v1/categorias-beneficio
func (c *CatalogosController) GetCategorias() {
	result, err := services.GetCategoriasBeneficio(c.Ctx.Input.Header("Authorization"))
	if err != nil {
		helpers.InternalError(&c.Controller, err)
		return
	}
	helpers.Ok(&c.Controller, result)
}

// GetSectores GET /v1/sectores-economicos
func (c *CatalogosController) GetSectores() {
	result, err := services.GetSectoresEconomicos(c.Ctx.Input.Header("Authorization"))
	if err != nil {
		helpers.InternalError(&c.Controller, err)
		return
	}
	helpers.Ok(&c.Controller, result)
}
