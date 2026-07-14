package helpers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/beego/beego/v2/server/web"
	"github.com/udistrital/utils_oas/v2/request"
)

// authorizationContextKey debe ser un string plano: utils_oas/v2/request busca el
// token con ctx.Value("Authorization") usando ese mismo tipo/valor.
const authorizationContextKey = "Authorization"

// Las URLs se alimentan por conf/app.conf (env vars), sin defaults quemados: si
// falta la variable, la llamada falla explícito en vez de pegarle a un servicio real.
var (
	crudURL       = web.AppConfig.DefaultString("CrudService", "")
	authURL       = web.AppConfig.DefaultString("AutenticacionService", "")
	parametrosURL = web.AppConfig.DefaultString("ParametrosService", "")
	// Datos de proveedor/empresa (C-2b): administrativa_amazon_api, no agora_crud.
	amazonURL = web.AppConfig.DefaultString("AmazonService", "")
	// OIDC userinfo: endpoint directo de WSO2, no va bajo /apioas.
	userinfoURL         = web.AppConfig.DefaultString("Wso2UserService", "")
	tercerosURL         = web.AppConfig.DefaultString("TercerosService", "")
	sgaMidURL           = web.AppConfig.DefaultString("SgaMidService", "")
	gestorDocumentalURL = web.AppConfig.DefaultString("GestorDocumentalService", "")
	academicaJbpmURL    = web.AppConfig.DefaultString("AcademicaJbpmService", "")
)

// ctxConToken arma el context.Context que espera utils_oas/v2/request con el
// Authorization del request entrante (request-scoped, seguro bajo concurrencia).
func ctxConToken(token string) context.Context {
	if token == "" {
		return context.Background()
	}
	return context.WithValue(context.Background(), authorizationContextKey, token)
}

// normalizarListaVacia convierte el idioma [{}] de los *_crud del SGA (lista vacía)
// en un array vacío real. Compacta el JSON antes de comparar porque en RunMode=dev
// Beego responde pretty-printed.
func normalizarListaVacia(body []byte) []byte {
	compact := new(bytes.Buffer)
	if err := json.Compact(compact, body); err == nil && compact.String() == "[{}]" {
		return []byte("[]")
	}
	return body
}

// GetCRUD realiza un GET al CRUD y decodifica la respuesta en dest.
func GetCRUD(token, path string, dest interface{}) error {
	var raw json.RawMessage
	status, err := request.GetWithContext(ctxConToken(token), crudURL+path, &raw)
	if err != nil {
		return fmt.Errorf("CRUD respondió %d: %w", status, err)
	}
	return json.Unmarshal(normalizarListaVacia(raw), dest)
}

// PostCRUD realiza un POST al CRUD con el payload dado y decodifica la respuesta en dest.
func PostCRUD(token, path string, payload interface{}, dest interface{}) error {
	status, err := request.PostWithContext(ctxConToken(token), crudURL+path, payload, dest)
	if err != nil {
		return fmt.Errorf("CRUD respondió %d: %w", status, err)
	}
	return nil
}

// PutCRUD realiza un PUT al CRUD con el payload dado.
func PutCRUD(token, path string, payload interface{}) error {
	var discard json.RawMessage
	status, err := request.PutWithContext(ctxConToken(token), crudURL+path, payload, &discard)
	if err != nil {
		return fmt.Errorf("CRUD respondió %d: %w", status, err)
	}
	return nil
}

// DeleteCRUD realiza un DELETE al CRUD (borrado lógico: el CRUD pone Activo=false).
func DeleteCRUD(token, path string) error {
	var discard json.RawMessage
	status, err := request.DeleteWithContext(ctxConToken(token), crudURL+path, &discard)
	if err != nil {
		return fmt.Errorf("CRUD respondió %d: %w", status, err)
	}
	return nil
}

// AuthURL devuelve la URL base del servicio de autenticación.
func AuthURL() string { return authURL }

// PostAuth realiza un POST al servicio autenticacion_mid (p. ej. /token/userRol) y
// decodifica la respuesta en dest. La respuesta NO usa el envelope { Success, Data }.
func PostAuth(token, path string, payload interface{}, dest interface{}) error {
	status, err := request.PostWithContext(ctxConToken(token), authURL+path, payload, dest)
	if err != nil {
		return fmt.Errorf("autenticacion_mid respondió %d: %w", status, err)
	}
	return nil
}

// GetAmazon realiza un GET al servicio administrativa_amazon_api (datos de proveedor)
// y decodifica la respuesta en dest. La respuesta es un array JSON crudo (sin envelope).
func GetAmazon(token, path string, dest interface{}) error {
	status, err := request.GetWithContext(ctxConToken(token), amazonURL+path, dest)
	if err != nil {
		return fmt.Errorf("administrativa_amazon_api respondió %d: %w", status, err)
	}
	return nil
}

// GetUserInfo consulta el endpoint OIDC userinfo: la identidad del dueño del token
// (sub, email, documento), fuente confiable del email autenticado para el JIT.
func GetUserInfo(token string, dest interface{}) error {
	status, err := request.GetWithContext(ctxConToken(token), userinfoURL, dest)
	if err != nil {
		return fmt.Errorf("userinfo respondió %d: %w", status, err)
	}
	return nil
}

// GetTerceros realiza un GET a terceros_crud y decodifica la respuesta en dest
// (array crudo con el idioma [{}] = lista vacía).
func GetTerceros(token, path string, dest interface{}) error {
	var raw json.RawMessage
	status, err := request.GetWithContext(ctxConToken(token), tercerosURL+path, &raw)
	if err != nil {
		return fmt.Errorf("terceros_crud respondió %d: %w", status, err)
	}
	return json.Unmarshal(normalizarListaVacia(raw), dest)
}

// GetSgaMid realiza un GET al sga_mid institucional y decodifica la respuesta en dest.
func GetSgaMid(token, path string, dest interface{}) error {
	status, err := request.GetWithContext(ctxConToken(token), sgaMidURL+path, dest)
	if err != nil {
		return fmt.Errorf("sga_mid respondió %d: %w", status, err)
	}
	return nil
}

// GetParametros realiza un GET al servicio institucional de parámetros y
// decodifica la respuesta estándar { Success, Status, Message, Data } en dest.
func GetParametros(token, path string, dest interface{}) error {
	status, err := request.GetWithContext(ctxConToken(token), parametrosURL+path, dest)
	if err != nil {
		return fmt.Errorf("servicio de parámetros respondió %d: %w", status, err)
	}
	return nil
}

// GetAcademicaJbpm realiza un GET al servicio académico del SGA (jBPM).
func GetAcademicaJbpm(token, path string, dest interface{}) error {
	status, err := request.GetWithContext(ctxConToken(token), academicaJbpmURL+path, dest)
	if err != nil {
		return fmt.Errorf("academica_jbpm respondió %d: %w", status, err)
	}
	return nil
}

// PostGestorDocumental realiza un POST al gestor documental institucional.
func PostGestorDocumental(token, path string, payload interface{}, dest interface{}) error {
	status, err := request.PostWithContext(ctxConToken(token), gestorDocumentalURL+path, payload, dest)
	if err != nil {
		return fmt.Errorf("gestor_documental_mid respondió %d: %w", status, err)
	}
	return nil
}

// GetGestorDocumental realiza un GET al gestor documental institucional.
func GetGestorDocumental(token, path string, dest interface{}) error {
	status, err := request.GetWithContext(ctxConToken(token), gestorDocumentalURL+path, dest)
	if err != nil {
		return fmt.Errorf("gestor_documental_mid respondió %d: %w", status, err)
	}
	return nil
}

// DeleteGestorDocumental realiza un DELETE al gestor documental institucional.
func DeleteGestorDocumental(token, path string) error {
	var discard json.RawMessage
	status, err := request.DeleteWithContext(ctxConToken(token), gestorDocumentalURL+path, &discard)
	if err != nil {
		return fmt.Errorf("gestor_documental_mid respondió %d: %w", status, err)
	}
	return nil
}
