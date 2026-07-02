package controllers

import (
	"encoding/json"

	"github.com/beego/beego/v2/server/web"
	"github.com/udistrital/sga_mid_beneficios_egresados/helpers"
	"github.com/udistrital/sga_mid_beneficios_egresados/services"
)

type BeneficiosController struct{ web.Controller }

// GetCatalogo GET /v1/beneficios
// Catálogo paginado con filtros. Solo PUBLICADO, fecha_fin >= hoy, cupos > 0 (RN-008).
// Params: page, limit, categoria_id, empresa_id, q
func (c *BeneficiosController) GetCatalogo() {
	page, _ := c.GetInt("page", 1)
	limit, _ := c.GetInt("limit", 20)
	categoriaId, _ := c.GetInt("categoria_id", 0)
	empresaId, _ := c.GetInt("empresa_id", 0)
	q := c.GetString("q")

	token := c.Ctx.Input.Header("Authorization")
	result, err := services.GetCatalogoBeneficios(token, page, limit, categoriaId, empresaId, q)
	if err != nil {
		helpers.InternalError(&c.Controller, err)
		return
	}
	helpers.Ok(&c.Controller, result)
}

// GetOne GET /v1/beneficios/:id
// Detalle de un beneficio.
func (c *BeneficiosController) GetOne() {
	id, err := c.GetInt(":id")
	if err != nil {
		helpers.BadRequest(&c.Controller, "id inválido")
		return
	}
	result, err := services.GetBeneficioDetalle(c.Ctx.Input.Header("Authorization"), id)
	if err != nil {
		helpers.NotFound(&c.Controller, "beneficio")
		return
	}
	helpers.Ok(&c.Controller, result)
}

// GetByEmpresa GET /v1/empresas/:empresa_id/beneficios
// Vista de gestión del dueño: TODOS sus beneficios (cualquier estado) con
// métricas de solicitudes.
func (c *BeneficiosController) GetByEmpresa() {
	empresaId, err := c.GetInt(":empresa_id")
	if err != nil {
		helpers.BadRequest(&c.Controller, "empresa_id inválido")
		return
	}
	result, err := services.GetBeneficiosDeEmpresa(c.Ctx.Input.Header("Authorization"), empresaId)
	if err != nil {
		helpers.InternalError(&c.Controller, err)
		return
	}
	helpers.Ok(&c.Controller, result)
}

// Publicar POST /v1/empresas/:empresa_id/beneficios
// Publicar un beneficio. Solo si empresa = APROBADA. Valida RN-008b.
func (c *BeneficiosController) Publicar() {
	empresaId, err := c.GetInt(":empresa_id")
	if err != nil {
		helpers.BadRequest(&c.Controller, "empresa_id inválido")
		return
	}

	var body map[string]interface{}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &body); err != nil {
		helpers.BadRequest(&c.Controller, "cuerpo de solicitud inválido")
		return
	}

	result, err := services.PublicarBeneficio(c.Ctx.Input.Header("Authorization"), empresaId, body)
	if err != nil {
		helpers.UnprocessableEntity(&c.Controller, err.Error())
		return
	}
	helpers.Created(&c.Controller, result)
}

// Editar PUT /v1/beneficios/:id
// Editar beneficio (solo borradores o publicados sin solicitudes activas).
func (c *BeneficiosController) Editar() {
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

	if err := services.EditarBeneficio(c.Ctx.Input.Header("Authorization"), id, body); err != nil {
		helpers.UnprocessableEntity(&c.Controller, err.Error())
		return
	}
	helpers.Ok(&c.Controller, "beneficio actualizado")
}
