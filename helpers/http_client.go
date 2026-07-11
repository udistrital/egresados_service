package helpers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/beego/beego/v2/server/web"
	"github.com/udistrital/utils_oas/v2/request"
)

// authorizationContextKey debe ser un string plano (no un tipo propio): el paquete
// utils_oas/v2/request busca el token con ctx.Value("Authorization") usando ese mismo
// tipo/valor por dentro, así que un tipo distinto no lo encontraría.
const authorizationContextKey = "Authorization"

// Ninguna URL trae default quemado: todas se alimentan por conf/app.conf (env vars).
// Si falta la variable de entorno, el endpoint queda vacío y la llamada HTTP falla
// explícito (esquema faltante) en vez de pegarle en silencio a un servicio real.
var (
	crudURL       = web.AppConfig.DefaultString("CrudService", "")
	authURL       = web.AppConfig.DefaultString("AutenticacionService", "")
	parametrosURL = web.AppConfig.DefaultString("ParametrosService", "")
	// Datos de proveedor/empresa (C-2b). OJO: es administrativa_amazon_api, NO agora_crud.
	amazonURL = web.AppConfig.DefaultString("AmazonService", "")
	// OIDC userinfo: identidad del dueño del token (sin pasar email). OJO: NO va bajo
	// /apioas, es endpoint directo de WSO2.
	userinfoURL = web.AppConfig.DefaultString("Wso2UserService", "")
	// Identidad institucional del egresado (C-2a): nombre real y TerceroId por documento.
	tercerosURL = web.AppConfig.DefaultString("TercerosService", "")
	// consultar_persona (C-2a) vive en sga_mid/v1, NO en derecho_pecunario_mid.
	sgaMidURL = web.AppConfig.DefaultString("SgaMidService", "")
	// Gestor documental institucional (Nuxeo): subir/consultar/eliminar los PDFs de las
	// solicitudes. El cliente Angular nunca llama a este servicio directamente, solo el MID.
	gestorDocumentalURL = web.AppConfig.DefaultString("GestorDocumentalService", "")
	// Datos académicos del SGA (jBPM): carrera del estudiante por código. El cliente
	// Angular nunca llama a este servicio directamente, solo el MID.
	academicaJbpmURL = web.AppConfig.DefaultString("AcademicaJbpmService", "")
)

// ctxConToken arma el context.Context que espera utils_oas/v2/request, con el
// Authorization del request entrante (Bearer del controller, request-scoped —
// nunca una variable global de paquete, para ser seguro bajo concurrencia). De
// paso, request.GetWithContext/PostWithContext/etc. instrumentan cada llamada con
// un subsegmento de AWS X-Ray si xray.Init() está activo (ver main.go).
func ctxConToken(token string) context.Context {
	if token == "" {
		return context.Background()
	}
	return context.WithValue(context.Background(), authorizationContextKey, token)
}

// normalizarListaVacia convierte el idioma [{}] de los *_crud del SGA (lista vacía)
// en un array vacío real, para no inyectar un elemento zero-value en dest. Compacta
// el JSON antes de comparar: en RunMode=dev Beego responde pretty-printed
// ("[\n  {}\n]"), que una comparación literal con "[{}]" no detecta.
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

// GetUserInfo consulta el endpoint OIDC userinfo y decodifica en dest la identidad del
// dueño del token (sub, email, documento). Es la fuente CONFIABLE del email autenticado
// (derivada del token, no del body) para el JIT provisioning.
func GetUserInfo(token string, dest interface{}) error {
	status, err := request.GetWithContext(ctxConToken(token), userinfoURL, dest)
	if err != nil {
		return fmt.Errorf("userinfo respondió %d: %w", status, err)
	}
	return nil
}

// GetTerceros realiza un GET a terceros_crud y decodifica la respuesta en dest.
// Responde array JSON crudo con el idioma [{}] = lista vacía (mismo contrato que
// nuestro CRUD, que lo copió de terceros_crud).
func GetTerceros(token, path string, dest interface{}) error {
	var raw json.RawMessage
	status, err := request.GetWithContext(ctxConToken(token), tercerosURL+path, &raw)
	if err != nil {
		return fmt.Errorf("terceros_crud respondió %d: %w", status, err)
	}
	return json.Unmarshal(normalizarListaVacia(raw), dest)
}

// GetSgaMid realiza un GET al sga_mid institucional (p. ej. consultar_persona, C-2a)
// y decodifica la respuesta en dest.
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

// GetAcademicaJbpm realiza un GET al servicio académico del SGA (jBPM, p. ej.
// datos_estudiante/{codigo} o carrera/{codigo}) y decodifica la respuesta en dest.
func GetAcademicaJbpm(token, path string, dest interface{}) error {
	status, err := request.GetWithContext(ctxConToken(token), academicaJbpmURL+path, dest)
	if err != nil {
		return fmt.Errorf("academica_jbpm respondió %d: %w", status, err)
	}
	return nil
}

// PostGestorDocumental realiza un POST al gestor documental institucional
// (subida de archivos) y decodifica la respuesta en dest.
func PostGestorDocumental(token, path string, payload interface{}, dest interface{}) error {
	status, err := request.PostWithContext(ctxConToken(token), gestorDocumentalURL+path, payload, dest)
	if err != nil {
		return fmt.Errorf("gestor_documental_mid respondió %d: %w", status, err)
	}
	return nil
}

// GetGestorDocumental realiza un GET al gestor documental institucional
// (consultar un documento por su uid/Enlace) y decodifica la respuesta en dest.
func GetGestorDocumental(token, path string, dest interface{}) error {
	status, err := request.GetWithContext(ctxConToken(token), gestorDocumentalURL+path, dest)
	if err != nil {
		return fmt.Errorf("gestor_documental_mid respondió %d: %w", status, err)
	}
	return nil
}

// DeleteGestorDocumental realiza un DELETE al gestor documental institucional
// (borrado lógico del documento en Nuxeo, ver gestor_documental_mid.md en memoria).
func DeleteGestorDocumental(token, path string) error {
	var discard json.RawMessage
	status, err := request.DeleteWithContext(ctxConToken(token), gestorDocumentalURL+path, &discard)
	if err != nil {
		return fmt.Errorf("gestor_documental_mid respondió %d: %w", status, err)
	}
	return nil
}
