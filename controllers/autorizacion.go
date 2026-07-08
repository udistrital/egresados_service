package controllers

import (
	"errors"

	"github.com/beego/beego/v2/server/web"
	"github.com/udistrital/egresados_service/helpers"
	"github.com/udistrital/egresados_service/services"
)

// responderErrorAcceso traduce los errores de las verificaciones anti-IDOR:
// ErrAccesoDenegado → 403; cualquier otro fallo (userinfo caído, CRUD caído) → 500.
func responderErrorAcceso(c *web.Controller, err error) {
	if errors.Is(err, services.ErrAccesoDenegado) {
		helpers.Forbidden(c, err.Error())
		return
	}
	helpers.InternalError(c, err)
}

// responderErrorNegocio es para errores de servicios que validan autorización
// ADEMÁS de reglas de negocio (p. ej. CrearSolicitud): ErrAccesoDenegado → 403;
// cualquier otro → 422 (validación).
func responderErrorNegocio(c *web.Controller, err error) {
	if errors.Is(err, services.ErrAccesoDenegado) {
		helpers.Forbidden(c, err.Error())
		return
	}
	helpers.UnprocessableEntity(c, err.Error())
}
