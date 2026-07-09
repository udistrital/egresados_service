package controllers

import (
	"fmt"

	"github.com/beego/beego/v2/server/web"
	"github.com/udistrital/egresados_service/helpers"
	"github.com/udistrital/egresados_service/services"
)

type EmpresasController struct{ web.Controller }

// @Title Provisionar
// @Description JIT provisioning del usuario de empresa al primer login (C-2b/c): resuelve su identidad (desde el token vía OIDC userinfo) y sus proveedores en Ágora, y da de alta usuario/empresa/usuario_empresa. No recibe body: la identidad sale del token.
// @Success 200 {object} helpers.APIResponse
// @Failure 422 no se pudo resolver la identidad o el proveedor en Ágora
// @router /empresas/provision [post]
func (c *EmpresasController) Provisionar() {
	result, err := services.ProvisionarEmpresa(c.Ctx.Input.Header("Authorization"))
	if err != nil {
		helpers.UnprocessableEntity(&c.Controller, err.Error())
		return
	}
	helpers.Ok(&c.Controller, result)
}

// @Title GetEmpresasDeUsuario
// @Description Empresas a las que el usuario tiene acceso (selector multiempresa, caso 1:N).
// @Param   usuario_id    path    int    true    "id del usuario"
// @Success 200 {object} helpers.APIResponse
// @Failure 400 usuario_id inválido
// @Failure 403 el usuario_id no coincide con el del token
// @Failure 500 error interno
// @router /usuarios/:usuario_id/empresas [get]
func (c *EmpresasController) GetEmpresasDeUsuario() {
	usuarioId, err := c.GetInt(":usuario_id")
	if err != nil {
		helpers.BadRequest(&c.Controller, "usuario_id inválido")
		return
	}

	token := c.Ctx.Input.Header("Authorization")
	if err := services.VerificarUsuarioDelToken(token, usuarioId); err != nil {
		responderErrorAcceso(&c.Controller, err)
		return
	}

	result, err := services.GetEmpresasDeUsuario(token, usuarioId)
	if err != nil {
		helpers.InternalError(&c.Controller, err)
		return
	}
	helpers.Ok(&c.Controller, result)
}

// @Title GetPerfil
// @Description Perfil público de la empresa (razón social, descripción/web/dirección de Ágora on-demand, métricas de beneficios). Whitelist RNF-002b — sin NIT ni datos bancarios.
// @Param   id    path    int    true    "id de la empresa"
// @Success 200 {object} helpers.APIResponse
// @Failure 400 id inválido
// @Failure 404 no encontrada
// @router /empresas/:id [get]
func (c *EmpresasController) GetPerfil() {
	id, err := c.GetInt(":id")
	if err != nil {
		helpers.BadRequest(&c.Controller, "id inválido")
		return
	}
	result, err := services.GetPerfilEmpresa(c.Ctx.Input.Header("Authorization"), id)
	if err != nil {
		// NotFound ya añade "no encontrado"; pasar solo el recurso.
		helpers.NotFound(&c.Controller, fmt.Sprintf("empresa %d", id))
		return
	}
	helpers.Ok(&c.Controller, result)
}

// @Title GetBandeja
// @Description Bandeja de solicitudes recibidas. Solo campos mínimos del egresado (RNF-002b / Ley 1581).
// @Param   empresa_id    path    int    true    "id de la empresa"
// @Success 200 {object} helpers.APIResponse
// @Failure 400 empresa_id inválido
// @Failure 403 el usuario del token no tiene acceso a esa empresa
// @Failure 500 error interno
// @router /empresas/:empresa_id/solicitudes [get]
func (c *EmpresasController) GetBandeja() {
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

	result, err := services.GetBandejaEmpresa(token, empresaId)
	if err != nil {
		helpers.InternalError(&c.Controller, err)
		return
	}
	helpers.Ok(&c.Controller, result)
}
