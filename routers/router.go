// @APIVersion 1.0.0
// @Title egresados_service — API MID Beneficios Egresados
// @Description Lógica de negocio del módulo Beneficios para Egresados. Todo /v1/* exige
// @Description un JWT Bearer de WSO2 (middleware/jwt.go), salvo EGRESADOS_SERVICE_VALIDAR_JWT=false (solo dev).
// @SecurityDefinition Bearer apiKey Authorization header "Pegar el token con el prefijo 'Bearer '. Ej: Bearer eyJhbGciOi..."
// @Security Bearer
package routers

import (
	"github.com/beego/beego/v2/server/web"
	"github.com/udistrital/egresados_service/controllers"
)

func init() {
	// Rutas registradas vía NSInclude a partir de las anotaciones @router de cada
	// controller (routers/commentsRouter.go, generado con `bee generate routers`).
	// A diferencia del CRUD, aquí una misma ruta REST cruza varios controllers (p. ej.
	// /empresas/:empresa_id/beneficios vive en BeneficiosController, no en
	// EmpresasController), así que todos los controllers comparten un solo NSInclude
	// bajo /v1 en vez de un NSNamespace por controller.
	ns := web.NewNamespace("/v1",
		web.NSInclude(
			&controllers.BeneficiosController{},
			&controllers.SolicitudesController{},
			&controllers.EmpresasController{},
			&controllers.EgresadosController{},
			&controllers.AdminController{},
			&controllers.CatalogosController{},
		),
	)
	web.AddNamespace(ns)
}
