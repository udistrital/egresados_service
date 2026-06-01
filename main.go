package main

import (
	"os"
	"strconv"

	"github.com/beego/beego/v2/server/web"
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
}

func main() {
	web.Run()
}
