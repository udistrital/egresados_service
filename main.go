package main

import (
	"github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/filter/cors"
	"github.com/udistrital/egresados_service/middleware"
	_ "github.com/udistrital/egresados_service/routers"
)

func init() {
	// httpport/runmode se resuelven solos desde conf/app.conf (Beego los aplica al
	// parsear el ini); ya no hace falta leerlos a mano por os.Getenv.

	// CORS: el micro-frontend (localhost:4200) llama al MID cross-origin. Se permiten
	// todos los orígenes (API pública tras el gateway); Authorization va en AllowHeaders
	// para que el AuthInterceptor pueda anexar el Bearer. Maneja también el preflight OPTIONS.
	web.InsertFilter("*", web.BeforeRouter, cors.Allow(&cors.Options{
		AllowAllOrigins: true,
		AllowMethods:    []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:    []string{"Origin", "Authorization", "Content-Type", "Accept"},
		ExposeHeaders:   []string{"Content-Length"},
	}))

	// Validación del JWT ENTRANTE (después de CORS para no interferir el preflight):
	// firma RS256 contra el JWKS de WSO2 o userinfo para tokens opacos. Sin token
	// válido → 401 antes de tocar cualquier controller. Ver middleware/jwt.go.
	web.InsertFilter("/v1/*", web.BeforeRouter, middleware.ValidarJWTEntrante)

	// EnableDocs (conf/app.conf) solo activa la bandera; Beego v2 no sirve
	// swagger/ automáticamente, hay que exponerla como estática (bee generate docs).
	if web.BConfig.WebConfig.EnableDocs {
		web.SetStaticPath("/swagger", "swagger")
	}
}

func main() {
	web.Run()
}
