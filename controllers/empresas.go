package controllers

import (
	"fmt"

	"github.com/beego/beego/v2/server/web"
	"github.com/udistrital/sga_mid_beneficios_egresados/helpers"
	"github.com/udistrital/sga_mid_beneficios_egresados/services"
)

type EmpresasController struct{ web.Controller }

// Provisionar POST /v1/empresas/provision
// JIT provisioning del usuario de empresa al primer login (C-2b/c): resuelve su
// identidad (desde el token vía OIDC userinfo) y sus proveedores en Ágora, y da de alta
// usuario/empresa/usuario_empresa. No recibe body: la identidad sale del token.
func (c *EmpresasController) Provisionar() {
	result, err := services.ProvisionarEmpresa(c.Ctx.Input.Header("Authorization"))
	if err != nil {
		helpers.UnprocessableEntity(&c.Controller, err.Error())
		return
	}
	helpers.Ok(&c.Controller, result)
}

// GetEmpresasDeUsuario GET /v1/usuarios/:usuario_id/empresas
// Empresas a las que el usuario tiene acceso (selector multiempresa, caso 1:N).
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

// GetPerfil GET /v1/empresas/:id
// Perfil público de la empresa (razón social, descripción/web/dirección de Ágora
// on-demand, métricas de beneficios). Whitelist RNF-002b — sin NIT ni datos bancarios.
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

// GetBandeja GET /v1/empresas/:empresa_id/solicitudes
// Bandeja de solicitudes recibidas. Solo campos mínimos del egresado (RNF-002b / Ley 1581).
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
