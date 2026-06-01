package helpers

import (
	"fmt"

	"github.com/beego/beego/v2/server/web"
)

// APIResponse es el envelope estándar de respuesta OATI.
type APIResponse struct {
	Status  string      `json:"Status"`
	Success bool        `json:"Success"`
	Body    interface{} `json:"Body,omitempty"`
	Message string      `json:"Message,omitempty"`
}

// Ok responde 200 con el body indicado.
func Ok(c *web.Controller, body interface{}) {
	c.Data["json"] = APIResponse{Status: "200", Success: true, Body: body}
	c.ServeJSON()
}

// Created responde 201 con el body indicado.
func Created(c *web.Controller, body interface{}) {
	c.Ctx.Output.SetStatus(201)
	c.Data["json"] = APIResponse{Status: "201", Success: true, Body: body}
	c.ServeJSON()
}

// BadRequest responde 400 con el mensaje indicado.
func BadRequest(c *web.Controller, msg string) {
	c.Ctx.Output.SetStatus(400)
	c.Data["json"] = APIResponse{Status: "400", Success: false, Message: msg}
	c.ServeJSON()
}

// Unauthorized responde 401.
func Unauthorized(c *web.Controller) {
	c.Ctx.Output.SetStatus(401)
	c.Data["json"] = APIResponse{Status: "401", Success: false, Message: "No autorizado"}
	c.ServeJSON()
}

// Forbidden responde 403 con el mensaje indicado.
func Forbidden(c *web.Controller, msg string) {
	c.Ctx.Output.SetStatus(403)
	c.Data["json"] = APIResponse{Status: "403", Success: false, Message: msg}
	c.ServeJSON()
}

// NotFound responde 404.
func NotFound(c *web.Controller, resource string) {
	c.Ctx.Output.SetStatus(404)
	c.Data["json"] = APIResponse{Status: "404", Success: false, Message: fmt.Sprintf("%s no encontrado", resource)}
	c.ServeJSON()
}

// InternalError responde 500 con el error.
func InternalError(c *web.Controller, err error) {
	c.Ctx.Output.SetStatus(500)
	c.Data["json"] = APIResponse{Status: "500", Success: false, Message: err.Error()}
	c.ServeJSON()
}

// UnprocessableEntity responde 422 con el mensaje de validación.
func UnprocessableEntity(c *web.Controller, msg string) {
	c.Ctx.Output.SetStatus(422)
	c.Data["json"] = APIResponse{Status: "422", Success: false, Message: msg}
	c.ServeJSON()
}
