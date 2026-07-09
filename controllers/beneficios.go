package controllers

import (
	"encoding/json"

	"github.com/beego/beego/v2/server/web"
	"github.com/udistrital/egresados_service/helpers"
	"github.com/udistrital/egresados_service/services"
)

type BeneficiosController struct{ web.Controller }

// @Title GetCatalogo
// @Description Catálogo paginado de beneficios. Solo PUBLICADO, fecha_fin >= hoy, cupos_disponibles > 0 (RN-008).
// @Param   page           query   int     false   "página (default 1)"
// @Param   limit          query   int     false   "tamaño de página (default 20)"
// @Param   categoria_id   query   int     false   "filtro por categoría"
// @Param   empresa_id     query   int     false   "filtro por empresa"
// @Param   q              query   string  false   "búsqueda por título"
// @Success 200 {object} helpers.APIResponse
// @Failure 500 error interno (CRUD o servicio de parámetros caído)
// @router /beneficios [get]
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

// @Title GetOne
// @Description Detalle de un beneficio.
// @Param   id    path    int    true    "id del beneficio"
// @Success 200 {object} helpers.APIResponse
// @Failure 400 id inválido
// @Failure 404 no encontrado
// @router /beneficios/:id [get]
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

// @Title GetByEmpresa
// @Description Vista de gestión del dueño: TODOS sus beneficios (cualquier estado) con métricas de solicitudes.
// @Param   empresa_id    path    int    true    "id de la empresa"
// @Success 200 {object} helpers.APIResponse
// @Failure 400 empresa_id inválido
// @Failure 403 el usuario del token no tiene acceso a esa empresa
// @Failure 500 error interno
// @router /empresas/:empresa_id/beneficios [get]
func (c *BeneficiosController) GetByEmpresa() {
	empresaId, err := c.GetInt(":empresa_id")
	if err != nil {
		helpers.BadRequest(&c.Controller, "empresa_id inválido")
		return
	}
	token := c.Ctx.Input.Header("Authorization")
	if err := services.VerificarAccesoEmpresa(token, empresaId); err != nil {
		responderErrorAcceso(&c.Controller, err)
		return
	}

	result, err := services.GetBeneficiosDeEmpresa(token, empresaId)
	if err != nil {
		helpers.InternalError(&c.Controller, err)
		return
	}
	helpers.Ok(&c.Controller, result)
}

// @Title Publicar
// @Description Publicar un beneficio. Solo si empresa = ACTIVA. Valida RN-008b.
// @Param   empresa_id    path    int                       true    "id de la empresa"
// @Param   body          body    string   true    "JSON con los campos del beneficio a publicar"
// @Success 201 {object} helpers.APIResponse
// @Failure 400 empresa_id o body inválido
// @Failure 403 el usuario del token no tiene acceso a esa empresa
// @Failure 422 la empresa no está ACTIVA u otra regla de negocio (RN-008b)
// @router /empresas/:empresa_id/beneficios [post]
func (c *BeneficiosController) Publicar() {
	empresaId, err := c.GetInt(":empresa_id")
	if err != nil {
		helpers.BadRequest(&c.Controller, "empresa_id inválido")
		return
	}

	token := c.Ctx.Input.Header("Authorization")
	if err := services.VerificarAccesoEmpresa(token, empresaId); err != nil {
		responderErrorAcceso(&c.Controller, err)
		return
	}

	var body map[string]interface{}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &body); err != nil {
		helpers.BadRequest(&c.Controller, "cuerpo de solicitud inválido")
		return
	}

	result, err := services.PublicarBeneficio(token, empresaId, body)
	if err != nil {
		helpers.UnprocessableEntity(&c.Controller, err.Error())
		return
	}
	helpers.Created(&c.Controller, result)
}

// @Title GetDocumentosRequeridos
// @Description Documentos que la empresa exige para postularse (definidos al publicar).
// @Param   id    path    int    true    "id del beneficio"
// @Success 200 {object} helpers.APIResponse
// @Failure 400 id inválido
// @Failure 500 error interno
// @router /beneficios/:id/documentos-requeridos [get]
func (c *BeneficiosController) GetDocumentosRequeridos() {
	id, err := c.GetInt(":id")
	if err != nil {
		helpers.BadRequest(&c.Controller, "id inválido")
		return
	}

	result, err := services.GetDocumentosRequeridos(c.Ctx.Input.Header("Authorization"), id)
	if err != nil {
		helpers.InternalError(&c.Controller, err)
		return
	}
	helpers.Ok(&c.Controller, result)
}

// @Title Editar
// @Description Editar beneficio (solo borradores o publicados sin solicitudes activas).
// @Param   id      path    int                       true    "id del beneficio"
// @Param   body    body    string   true    "JSON con los campos a actualizar"
// @Success 200 {object} helpers.APIResponse
// @Failure 400 id o body inválido
// @Failure 403 el usuario del token no tiene acceso a ese beneficio
// @Failure 422 el beneficio no se puede editar en su estado actual
// @router /beneficios/:id [put]
func (c *BeneficiosController) Editar() {
	id, err := c.GetInt(":id")
	if err != nil {
		helpers.BadRequest(&c.Controller, "id inválido")
		return
	}

	token := c.Ctx.Input.Header("Authorization")
	if err := services.VerificarAccesoBeneficio(token, id); err != nil {
		responderErrorAcceso(&c.Controller, err)
		return
	}

	var body map[string]interface{}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &body); err != nil {
		helpers.BadRequest(&c.Controller, "cuerpo de solicitud inválido")
		return
	}

	if err := services.EditarBeneficio(token, id, body); err != nil {
		helpers.UnprocessableEntity(&c.Controller, err.Error())
		return
	}
	helpers.Ok(&c.Controller, "beneficio actualizado")
}

// @Title Retirar
// @Description El "cerrar" de la empresa: pasa el beneficio a RETIRADO (sale del catálogo).
// @Param   id    path    int    true    "id del beneficio"
// @Success 200 {object} helpers.APIResponse
// @Failure 400 id inválido
// @Failure 403 el usuario del token no tiene acceso a ese beneficio
// @Failure 422 el beneficio no se puede retirar en su estado actual
// @router /beneficios/:id/retirar [put]
func (c *BeneficiosController) Retirar() {
	id, err := c.GetInt(":id")
	if err != nil {
		helpers.BadRequest(&c.Controller, "id inválido")
		return
	}

	token := c.Ctx.Input.Header("Authorization")
	if err := services.VerificarAccesoBeneficio(token, id); err != nil {
		responderErrorAcceso(&c.Controller, err)
		return
	}

	if err := services.RetirarBeneficio(token, id); err != nil {
		helpers.UnprocessableEntity(&c.Controller, err.Error())
		return
	}
	helpers.Ok(&c.Controller, "beneficio retirado")
}
