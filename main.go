package main

import (
	"os"
	"strconv"

	"github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/filter/cors"
	"github.com/udistrital/sga_mid_beneficios_egresados/middleware"
	_ "github.com/udistrital/sga_mid_beneficios_egresados/routers"
)

func init() {
	if port := os.Getenv("BENEFICIOS_EGRESADOS_MID_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			web.BConfig.Listen.HTTPPort = p
		}
	}
	if runmode := os.Getenv("BENEFICIOS_EGRESADOS_MID_RUNMODE"); runmode != "" {
		web.BConfig.RunMode = runmode
	}

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
}

func main() {
	web.Run()
}
