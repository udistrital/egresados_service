package routers

import (
	"github.com/beego/beego/v2/server/web"
	"github.com/udistrital/sga_mid_beneficios_egresados/controllers"
)

func init() {
	// ── Catálogo público (egresados) ──────────────────────────────────────────
	// GET  /v1/beneficios           RF-002 catálogo paginado con filtros
	// GET  /v1/beneficios/:id       RF-003 detalle de un beneficio
	web.Router("/v1/beneficios", &controllers.BeneficiosController{}, "get:GetCatalogo")
	web.Router("/v1/beneficios/:id", &controllers.BeneficiosController{}, "get:GetOne")

	// ── Solicitudes (egresado) ────────────────────────────────────────────────
	// POST /v1/solicitudes                                RF-003 crear solicitud
	// GET  /v1/solicitudes/egresado/:egresado_id          RF-008 mis solicitudes
	// PUT  /v1/solicitudes/:id/cancelar                   RF-008 cancelar
	// GET  /v1/solicitudes/egresado/:egresado_id/resumen  RF-013 resumen
	web.Router("/v1/solicitudes", &controllers.SolicitudesController{}, "post:Crear")
	web.Router("/v1/solicitudes/egresado/:egresado_id", &controllers.SolicitudesController{}, "get:GetByEgresado")
	web.Router("/v1/solicitudes/:id/cancelar", &controllers.SolicitudesController{}, "put:Cancelar")
	web.Router("/v1/solicitudes/egresado/:egresado_id/resumen", &controllers.SolicitudesController{}, "get:Resumen")

	// ── Bandeja empresa ───────────────────────────────────────────────────────
	// GET  /v1/empresas/:empresa_id/solicitudes    RF-006 bandeja de solicitudes
	// PUT  /v1/solicitudes/:id/responder           RF-007 aprobar/rechazar/requiere info
	// POST /v1/solicitudes/:id/mensajes            RF-007 enviar mensaje
	// GET  /v1/solicitudes/:id/mensajes            RF-007 historial mensajes
	web.Router("/v1/empresas/:empresa_id/solicitudes", &controllers.EmpresasController{}, "get:GetBandeja")
	web.Router("/v1/solicitudes/:id/responder", &controllers.SolicitudesController{}, "put:Responder")
	web.Router("/v1/solicitudes/:id/mensajes", &controllers.SolicitudesController{}, "post:EnviarMensaje;get:GetMensajes")

	// ── Empresas ──────────────────────────────────────────────────────────────
	// POST /v1/empresas                            RF-004 registrar empresa
	// POST /v1/empresas/:empresa_id/beneficios     RF-005 publicar beneficio
	// PUT  /v1/beneficios/:id                      RF-005 editar beneficio
	web.Router("/v1/empresas", &controllers.EmpresasController{}, "post:Registrar")
	web.Router("/v1/empresas/:empresa_id/beneficios", &controllers.BeneficiosController{}, "post:Publicar")
	web.Router("/v1/beneficios/:id", &controllers.BeneficiosController{}, "put:Editar")

	// ── Administrador ─────────────────────────────────────────────────────────
	web.Router("/v1/empresas/:id/suspender", &controllers.AdminController{}, "put:SuspenderEmpresa")

	// ── Catálogos (read-only) ─────────────────────────────────────────────────
	web.Router("/v1/categorias-beneficio", &controllers.CatalogosController{}, "get:GetCategorias")
	web.Router("/v1/sectores-economicos", &controllers.CatalogosController{}, "get:GetSectores")
}
