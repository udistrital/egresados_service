package routers

import (
	beego "github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context/param"
)

func init() {

	beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:AdminController"] = append(beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:AdminController"],
		beego.ControllerComments{
			Method:           "SuspenderEmpresa",
			Router:           `/empresas/:id/suspender`,
			AllowHTTPMethods: []string{"put"},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

	beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:BeneficiosController"] = append(beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:BeneficiosController"],
		beego.ControllerComments{
			Method:           "GetCatalogo",
			Router:           `/beneficios`,
			AllowHTTPMethods: []string{"get"},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

	beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:BeneficiosController"] = append(beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:BeneficiosController"],
		beego.ControllerComments{
			Method:           "GetOne",
			Router:           `/beneficios/:id`,
			AllowHTTPMethods: []string{"get"},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

	beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:BeneficiosController"] = append(beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:BeneficiosController"],
		beego.ControllerComments{
			Method:           "Editar",
			Router:           `/beneficios/:id`,
			AllowHTTPMethods: []string{"put"},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

	beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:BeneficiosController"] = append(beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:BeneficiosController"],
		beego.ControllerComments{
			Method:           "GetDocumentosRequeridos",
			Router:           `/beneficios/:id/documentos-requeridos`,
			AllowHTTPMethods: []string{"get"},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

	beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:BeneficiosController"] = append(beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:BeneficiosController"],
		beego.ControllerComments{
			Method:           "Retirar",
			Router:           `/beneficios/:id/retirar`,
			AllowHTTPMethods: []string{"put"},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

	beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:BeneficiosController"] = append(beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:BeneficiosController"],
		beego.ControllerComments{
			Method:           "GetByEmpresa",
			Router:           `/empresas/:empresa_id/beneficios`,
			AllowHTTPMethods: []string{"get"},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

	beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:BeneficiosController"] = append(beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:BeneficiosController"],
		beego.ControllerComments{
			Method:           "Publicar",
			Router:           `/empresas/:empresa_id/beneficios`,
			AllowHTTPMethods: []string{"post"},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

	beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:CatalogosController"] = append(beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:CatalogosController"],
		beego.ControllerComments{
			Method:           "GetCategorias",
			Router:           `/categorias-beneficio`,
			AllowHTTPMethods: []string{"get"},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

	beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:CatalogosController"] = append(beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:CatalogosController"],
		beego.ControllerComments{
			Method:           "GetSectores",
			Router:           `/sectores-economicos`,
			AllowHTTPMethods: []string{"get"},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

	beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:EgresadosController"] = append(beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:EgresadosController"],
		beego.ControllerComments{
			Method:           "Provisionar",
			Router:           `/egresados/provision`,
			AllowHTTPMethods: []string{"post"},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

	beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:EmpresasController"] = append(beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:EmpresasController"],
		beego.ControllerComments{
			Method:           "GetBandeja",
			Router:           `/empresas/:empresa_id/solicitudes`,
			AllowHTTPMethods: []string{"get"},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

	beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:EmpresasController"] = append(beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:EmpresasController"],
		beego.ControllerComments{
			Method:           "GetPerfil",
			Router:           `/empresas/:id`,
			AllowHTTPMethods: []string{"get"},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

	beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:EmpresasController"] = append(beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:EmpresasController"],
		beego.ControllerComments{
			Method:           "Provisionar",
			Router:           `/empresas/provision`,
			AllowHTTPMethods: []string{"post"},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

	beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:EmpresasController"] = append(beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:EmpresasController"],
		beego.ControllerComments{
			Method:           "GetEmpresasDeUsuario",
			Router:           `/usuarios/:usuario_id/empresas`,
			AllowHTTPMethods: []string{"get"},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

	beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:SolicitudesController"] = append(beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:SolicitudesController"],
		beego.ControllerComments{
			Method:           "GetArchivoDocumento",
			Router:           `/documentos/:doc_id/archivo`,
			AllowHTTPMethods: []string{"get"},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

	beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:SolicitudesController"] = append(beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:SolicitudesController"],
		beego.ControllerComments{
			Method:           "ComentarDocumento",
			Router:           `/documentos/:doc_id/comentario`,
			AllowHTTPMethods: []string{"put"},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

	beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:SolicitudesController"] = append(beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:SolicitudesController"],
		beego.ControllerComments{
			Method:           "Crear",
			Router:           `/solicitudes`,
			AllowHTTPMethods: []string{"post"},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

	beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:SolicitudesController"] = append(beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:SolicitudesController"],
		beego.ControllerComments{
			Method:           "Cancelar",
			Router:           `/solicitudes/:id/cancelar`,
			AllowHTTPMethods: []string{"put"},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

	beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:SolicitudesController"] = append(beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:SolicitudesController"],
		beego.ControllerComments{
			Method:           "GetComprobante",
			Router:           `/solicitudes/:id/comprobante`,
			AllowHTTPMethods: []string{"get"},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

	beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:SolicitudesController"] = append(beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:SolicitudesController"],
		beego.ControllerComments{
			Method:           "SubirDocumento",
			Router:           `/solicitudes/:id/documentos`,
			AllowHTTPMethods: []string{"post"},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

	beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:SolicitudesController"] = append(beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:SolicitudesController"],
		beego.ControllerComments{
			Method:           "GetDocumentos",
			Router:           `/solicitudes/:id/documentos`,
			AllowHTTPMethods: []string{"get"},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

	beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:SolicitudesController"] = append(beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:SolicitudesController"],
		beego.ControllerComments{
			Method:           "EliminarDocumento",
			Router:           `/solicitudes/:id/documentos/:doc_id`,
			AllowHTTPMethods: []string{"delete"},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

	beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:SolicitudesController"] = append(beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:SolicitudesController"],
		beego.ControllerComments{
			Method:           "GetHistorial",
			Router:           `/solicitudes/:id/historial`,
			AllowHTTPMethods: []string{"get"},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

	beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:SolicitudesController"] = append(beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:SolicitudesController"],
		beego.ControllerComments{
			Method:           "GetMensajes",
			Router:           `/solicitudes/:id/mensajes`,
			AllowHTTPMethods: []string{"get"},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

	beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:SolicitudesController"] = append(beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:SolicitudesController"],
		beego.ControllerComments{
			Method:           "EnviarMensaje",
			Router:           `/solicitudes/:id/mensajes`,
			AllowHTTPMethods: []string{"post"},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

	beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:SolicitudesController"] = append(beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:SolicitudesController"],
		beego.ControllerComments{
			Method:           "Responder",
			Router:           `/solicitudes/:id/responder`,
			AllowHTTPMethods: []string{"put"},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

	beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:SolicitudesController"] = append(beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:SolicitudesController"],
		beego.ControllerComments{
			Method:           "GetByEgresado",
			Router:           `/solicitudes/egresado/:egresado_id`,
			AllowHTTPMethods: []string{"get"},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

	beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:SolicitudesController"] = append(beego.GlobalControllerRouter["github.com/udistrital/egresados_service/controllers:SolicitudesController"],
		beego.ControllerComments{
			Method:           "Resumen",
			Router:           `/solicitudes/egresado/:egresado_id/resumen`,
			AllowHTTPMethods: []string{"get"},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

}
