package main

import (
	"github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/filter/cors"
	"github.com/udistrital/egresados_service/middleware"
	_ "github.com/udistrital/egresados_service/routers"
	apistatus "github.com/udistrital/utils_oas/v2/apiStatusLib"
	"github.com/udistrital/utils_oas/v2/auditoria"
	customerrorv2 "github.com/udistrital/utils_oas/v2/customerror"
	"github.com/udistrital/utils_oas/v2/security"
	"github.com/udistrital/utils_oas/v2/xray"
)

func init() {
	// httpport/runmode se resuelven solos desde conf/app.conf (Beego los aplica al
	// parsear el ini); ya no hace falta leerlos a mano por os.Getenv.

	// CORS (lineamiento institucional de configuracion_ci_cd.md, adaptado a Beego v2):
	// en prod solo se permiten orígenes *.udistrital.edu.co; en dev cualquiera, para no
	// pelear con localhost:4200/4201 durante el desarrollo. AllowOrigins (no
	// AllowAllOrigins) + AllowCredentials es la combinación correcta para credentials
	// con wildcard: beego v2 hace echo del Origin real de la request, no manda "*"
	// literal (ver cors.Options.Header en beego/v2/server/web/filter/cors).
	allowedOrigins := []string{"*.udistrital.edu.co"}
	if web.BConfig.RunMode == web.DEV {
		allowedOrigins = []string{"*"}
	}
	web.InsertFilter("*", web.BeforeRouter, cors.Allow(&cors.Options{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"DELETE", "GET", "OPTIONS", "PATCH", "POST", "PUT"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type", "User-Agent", "X-Amzn-Trace-Id"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Validación del JWT ENTRANTE (después de CORS para no interferir el preflight):
	// firma RS256 contra el JWKS de WSO2 o userinfo para tokens opacos. Sin token
	// válido → 401 antes de tocar cualquier controller. Ver middleware/jwt.go.
	// (auditoria.InitMiddleware(), más abajo, valida el token por su cuenta para el log
	// de auditoría, pero nunca corta el request — la única que aplica control de acceso
	// real es esta.)
	web.InsertFilter("/v1/*", web.BeforeRouter, middleware.ValidarJWTEntrante)

	// EnableDocs (conf/app.conf) solo activa la bandera; Beego v2 no sirve
	// swagger/ automáticamente, hay que exponerla como estática (bee generate docs).
	// Se lee directo del config genérico (no de BConfig.WebConfig.EnableDocs, que
	// Beego v2 marcó deprecated: "Beego didn't use it anymore" — sigue funcionando
	// hoy, pero podrían quitarlo en una versión futura).
	if web.AppConfig.DefaultBool("EnableDocs", false) {
		web.SetStaticPath("/swagger", "swagger")
	}

	// Integraciones institucionales (configuracion_ci_cd.md). Habilitadas ahora que
	// utils_oas/v2 (v2.0.0-beta.1, 2026-07) migró a Beego v2 — ya no choca con nuestro
	// stack (antes: panic: flag redefined: graceful, por astaxie/beego transitivo).
	apistatus.Init()              // GET / — healthcheck institucional ({"status":"ok"})
	auditoria.InitMiddleware()    // log de auditoría por request (aquí no hay ORM/SQL que loguear)
	security.SetSecurityHeaders() // headers CSP/HSTS/X-Frame-Options/etc.
	xray.Init()                   // tracing AWS X-Ray (no-op si PARAMETER_STORE no está seteado)
	web.ErrorController(&customerrorv2.CustomErrorController{})
}

func main() {
	web.Run()
}
