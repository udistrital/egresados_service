package main

import (
	"github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/filter/cors"
	"github.com/udistrital/egresados_service/middleware"
	_ "github.com/udistrital/egresados_service/routers"
	// Pendiente (ver nota "Integraciones institucionales" en init(), más abajo):
	// estos 5 imports rompen el build hoy por el choque Beego v1/v2. Descomentar
	// junto con el bloque de abajo cuando utils_oas soporte Beego v2.
	// apistatus "github.com/udistrital/utils_oas/apiStatusLib"
	// "github.com/udistrital/utils_oas/auditoria"
	// "github.com/udistrital/utils_oas/customerrorv2"
	// "github.com/udistrital/utils_oas/security"
	// "github.com/udistrital/utils_oas/xray"
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
	web.InsertFilter("/v1/*", web.BeforeRouter, middleware.ValidarJWTEntrante)

	// EnableDocs (conf/app.conf) solo activa la bandera; Beego v2 no sirve
	// swagger/ automáticamente, hay que exponerla como estática (bee generate docs).
	if web.BConfig.WebConfig.EnableDocs {
		web.SetStaticPath("/swagger", "swagger")
	}

	// ── Integraciones institucionales pendientes (configuracion_ci_cd.md) ──────────
	// Decisión con el Ingeniero (2026-07-09): este repo se queda en Beego v2 por ahora.
	// utils_oas se actualizará a v2 más adelante (de su lado o del nuestro); hasta que
	// eso pase, NO descomentar lo siguiente: apiStatusLib/auditoria/security/xray/
	// customerrorv2 importan github.com/astaxie/beego (v1), que registra la misma flag
	// "graceful" que beego/v2/server/web/grace — el binario hace panic al arrancar si
	// conviven los dos (confirmado: "panic: flag redefined: graceful").
	//
	// Cuando utils_oas soporte Beego v2 (o se decida migrar este repo a v1): descomentar
	// esto + los 5 imports de arriba, correr
	//   go get github.com/udistrital/utils_oas@latest && go mod tidy
	// y validar que compile y arranque (incluye el CORS, el ruteo y el swagger, que hoy
	// ya están probados y funcionando sin estas piezas) antes de desplegar.
	//
	// apistatus.Init()
	// auditoria.InitMiddleware()
	// security.SetSecurityHeaders()
	// xray.Init()
	// TODO: revisar si customerrorv2.CustomErrorController tiene equivalente para
	// web.ErrorController (v2) o si customerrorv2 en sí ya es agnóstico de versión.
	// web.ErrorController(&customerrorv2.CustomErrorController{})
}

func main() {
	web.Run()
}
