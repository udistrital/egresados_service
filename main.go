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
	// En prod solo se permiten orígenes *.udistrital.edu.co; en dev cualquiera.
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

	// Después de CORS para no interferir el preflight. Ver middleware/jwt.go.
	web.InsertFilter("/v1/*", web.BeforeRouter, middleware.ValidarJWTEntrante)

	apistatus.Init()
	auditoria.InitMiddleware()
	security.SetSecurityHeaders()
	xray.Init()
	web.ErrorController(&customerrorv2.CustomErrorController{})
}

func main() {
	web.Run()
}
