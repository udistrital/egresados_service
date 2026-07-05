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

	// ── Documentos requeridos / subidos (gestor_documental_mid, IdTipoDocumento=167) ──
	// GET    /v1/beneficios/:id/documentos-requeridos   qué documentos exige la empresa
	// GET    /v1/solicitudes/:id/documentos             requeridos vs. subidos (egresado y empresa)
	// POST   /v1/solicitudes/:id/documentos             el egresado sube/reemplaza un PDF
	// DELETE /v1/solicitudes/:id/documentos/:doc_id     el egresado quita un documento
	// PUT    /v1/documentos/:doc_id/comentario          la empresa comenta un documento
	// GET    /v1/documentos/:doc_id/archivo             ver/descargar (proxy de solo lectura)
	web.Router("/v1/beneficios/:id/documentos-requeridos", &controllers.BeneficiosController{}, "get:GetDocumentosRequeridos")
	web.Router("/v1/solicitudes/:id/documentos", &controllers.SolicitudesController{}, "get:GetDocumentos;post:SubirDocumento")
	web.Router("/v1/solicitudes/:id/documentos/:doc_id", &controllers.SolicitudesController{}, "delete:EliminarDocumento")
	web.Router("/v1/documentos/:doc_id/comentario", &controllers.SolicitudesController{}, "put:ComentarDocumento")
	web.Router("/v1/documentos/:doc_id/archivo", &controllers.SolicitudesController{}, "get:GetArchivoDocumento")

	// GET /v1/solicitudes/:id/comprobante — comprobante OPCIONAL que la empresa
	// adjunta al aprobar (ver PUT /solicitudes/:id/responder, body.comprobante)
	web.Router("/v1/solicitudes/:id/comprobante", &controllers.SolicitudesController{}, "get:GetComprobante")

	// ── Egresados ─────────────────────────────────────────────────────────────
	// POST /v1/egresados/provision                 C-2a JIT provisioning al login
	web.Router("/v1/egresados/provision", &controllers.EgresadosController{}, "post:Provisionar")

	// ── Empresas ──────────────────────────────────────────────────────────────
	// POST /v1/empresas/provision                  C-2b/c JIT provisioning al login
	// GET  /v1/empresas/:id                        perfil público (detalle beneficio)
	// GET  /v1/usuarios/:usuario_id/empresas       selector multiempresa (caso 1:N)
	// POST /v1/empresas/:empresa_id/beneficios     RF-005 publicar beneficio
	// GET  /v1/empresas/:empresa_id/beneficios     gestión: mis beneficios (dueño)
	// PUT  /v1/beneficios/:id                      RF-005 editar beneficio
	web.Router("/v1/empresas/provision", &controllers.EmpresasController{}, "post:Provisionar")
	web.Router("/v1/empresas/:id", &controllers.EmpresasController{}, "get:GetPerfil")
	web.Router("/v1/usuarios/:usuario_id/empresas", &controllers.EmpresasController{}, "get:GetEmpresasDeUsuario")
	web.Router("/v1/empresas/:empresa_id/beneficios", &controllers.BeneficiosController{}, "post:Publicar;get:GetByEmpresa")
	web.Router("/v1/beneficios/:id", &controllers.BeneficiosController{}, "put:Editar")

	// ── Administrador ─────────────────────────────────────────────────────────
	web.Router("/v1/empresas/:id/suspender", &controllers.AdminController{}, "put:SuspenderEmpresa")

	// ── Catálogos (read-only) ─────────────────────────────────────────────────
	web.Router("/v1/categorias-beneficio", &controllers.CatalogosController{}, "get:GetCategorias")
	web.Router("/v1/sectores-economicos", &controllers.CatalogosController{}, "get:GetSectores")
}
