package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/beego/beego/v2/server/web"
)

var (
	crudURL       = web.AppConfig.DefaultString("CrudService", "http://localhost:8080/v1")
	authURL       = web.AppConfig.DefaultString("AutenticacionService", "https://autenticacion.portaloas.udistrital.edu.co/apioas/autenticacion_mid/v1")
	parametrosURL = web.AppConfig.DefaultString("ParametrosService", "https://autenticacion.portaloas.udistrital.edu.co/apioas/parametros/v1")
	// Datos de proveedor/empresa (C-2b). OJO: es administrativa_amazon_api, NO agora_crud.
	amazonURL = web.AppConfig.DefaultString("AmazonService", "https://autenticacion.portaloas.udistrital.edu.co/apioas/administrativa_amazon_api/v1")
	// OIDC userinfo: identidad del dueño del token (sin pasar email). OJO: NO va bajo
	// /apioas, es endpoint directo de WSO2.
	userinfoURL = web.AppConfig.DefaultString("Wso2UserService", "https://autenticacion.portaloas.udistrital.edu.co/oauth2/userinfo")
	// Identidad institucional del egresado (C-2a): nombre real y TerceroId por documento.
	tercerosURL = web.AppConfig.DefaultString("TercerosService", "https://autenticacion.portaloas.udistrital.edu.co/apioas/terceros_crud/v1")
	// consultar_persona (C-2a) vive en sga_mid/v1, NO en derecho_pecunario_mid.
	sgaMidURL = web.AppConfig.DefaultString("SgaMidService", "https://autenticacion.portaloas.udistrital.edu.co/apioas/sga_mid/v1")
	// Gestor documental institucional (Nuxeo): subir/consultar/eliminar los PDFs de las
	// solicitudes. El cliente Angular nunca llama a este servicio directamente, solo el MID.
	gestorDocumentalURL = web.AppConfig.DefaultString("GestorDocumentalService", "https://autenticacion.portaloas.udistrital.edu.co/apioas/gestor_documental_mid/v1")
)

// doRequest realiza una petición HTTP propagando el token del usuario (Bearer del
// request entrante). El gateway institucional (parámetros, autenticacion_mid, Ágora)
// rechaza con 401 cualquier llamada sin este header; el token viaja explícitamente
// desde el controller (request-scoped) para ser seguro bajo concurrencia — nunca en
// una variable global de paquete. Si token == "" no se añade el header (útil para el
// CRUD local mientras no valide JWT).
func doRequest(method, token, urlStr string, payload interface{}) ([]byte, int, error) {
	var bodyReader io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, 0, err
		}
		bodyReader = strings.NewReader(string(b))
	}

	req, err := http.NewRequest(method, urlStr, bodyReader)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		// El header entrante ya viene con el prefijo "Bearer "; se reenvía tal cual.
		req.Header.Set("Authorization", token)
	}

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
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
	body, status, err := doRequest(http.MethodGet, token, fmt.Sprintf("%s%s", crudURL, path), nil)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("CRUD respondió %d: %s", status, string(body))
	}
	return json.Unmarshal(normalizarListaVacia(body), dest)
}

// PostCRUD realiza un POST al CRUD con el payload dado y decodifica la respuesta en dest.
func PostCRUD(token, path string, payload interface{}, dest interface{}) error {
	body, status, err := doRequest(http.MethodPost, token, fmt.Sprintf("%s%s", crudURL, path), payload)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("CRUD respondió %d: %s", status, string(body))
	}
	return json.Unmarshal(body, dest)
}

// PutCRUD realiza un PUT al CRUD con el payload dado.
func PutCRUD(token, path string, payload interface{}) error {
	body, status, err := doRequest(http.MethodPut, token, fmt.Sprintf("%s%s", crudURL, path), payload)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("CRUD respondió %d: %s", status, string(body))
	}
	return nil
}

// DeleteCRUD realiza un DELETE al CRUD (borrado lógico: el CRUD pone Activo=false).
func DeleteCRUD(token, path string) error {
	body, status, err := doRequest(http.MethodDelete, token, fmt.Sprintf("%s%s", crudURL, path), nil)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("CRUD respondió %d: %s", status, string(body))
	}
	return nil
}

// AuthURL devuelve la URL base del servicio de autenticación.
func AuthURL() string { return authURL }

// PostAuth realiza un POST al servicio autenticacion_mid (p. ej. /token/userRol) y
// decodifica la respuesta en dest. La respuesta NO usa el envelope { Success, Data }.
func PostAuth(token, path string, payload interface{}, dest interface{}) error {
	body, status, err := doRequest(http.MethodPost, token, fmt.Sprintf("%s%s", authURL, path), payload)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("autenticacion_mid respondió %d: %s", status, string(body))
	}
	return json.Unmarshal(body, dest)
}

// GetAmazon realiza un GET al servicio administrativa_amazon_api (datos de proveedor)
// y decodifica la respuesta en dest. La respuesta es un array JSON crudo (sin envelope).
func GetAmazon(token, path string, dest interface{}) error {
	body, status, err := doRequest(http.MethodGet, token, fmt.Sprintf("%s%s", amazonURL, path), nil)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("administrativa_amazon_api respondió %d: %s", status, string(body))
	}
	return json.Unmarshal(body, dest)
}

// GetUserInfo consulta el endpoint OIDC userinfo y decodifica en dest la identidad del
// dueño del token (sub, email, documento). Es la fuente CONFIABLE del email autenticado
// (derivada del token, no del body) para el JIT provisioning.
func GetUserInfo(token string, dest interface{}) error {
	body, status, err := doRequest(http.MethodGet, token, userinfoURL, nil)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("userinfo respondió %d: %s", status, string(body))
	}
	return json.Unmarshal(body, dest)
}

// GetTerceros realiza un GET a terceros_crud y decodifica la respuesta en dest.
// Responde array JSON crudo con el idioma [{}] = lista vacía (mismo contrato que
// nuestro CRUD, que lo copió de terceros_crud).
func GetTerceros(token, path string, dest interface{}) error {
	body, status, err := doRequest(http.MethodGet, token, fmt.Sprintf("%s%s", tercerosURL, path), nil)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("terceros_crud respondió %d: %s", status, string(body))
	}
	return json.Unmarshal(normalizarListaVacia(body), dest)
}

// GetSgaMid realiza un GET al sga_mid institucional (p. ej. consultar_persona, C-2a)
// y decodifica la respuesta en dest.
func GetSgaMid(token, path string, dest interface{}) error {
	body, status, err := doRequest(http.MethodGet, token, fmt.Sprintf("%s%s", sgaMidURL, path), nil)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("sga_mid respondió %d: %s", status, string(body))
	}
	return json.Unmarshal(body, dest)
}

// GetParametros realiza un GET al servicio institucional de parámetros y
// decodifica la respuesta estándar { Success, Status, Message, Data } en dest.
func GetParametros(token, path string, dest interface{}) error {
	body, status, err := doRequest(http.MethodGet, token, fmt.Sprintf("%s%s", parametrosURL, path), nil)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("servicio de parámetros respondió %d: %s", status, string(body))
	}
	return json.Unmarshal(body, dest)
}

// PostGestorDocumental realiza un POST al gestor documental institucional
// (subida de archivos) y decodifica la respuesta en dest.
func PostGestorDocumental(token, path string, payload interface{}, dest interface{}) error {
	body, status, err := doRequest(http.MethodPost, token, fmt.Sprintf("%s%s", gestorDocumentalURL, path), payload)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("gestor_documental_mid respondió %d: %s", status, string(body))
	}
	return json.Unmarshal(body, dest)
}

// GetGestorDocumental realiza un GET al gestor documental institucional
// (consultar un documento por su uid/Enlace) y decodifica la respuesta en dest.
func GetGestorDocumental(token, path string, dest interface{}) error {
	body, status, err := doRequest(http.MethodGet, token, fmt.Sprintf("%s%s", gestorDocumentalURL, path), nil)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("gestor_documental_mid respondió %d: %s", status, string(body))
	}
	return json.Unmarshal(body, dest)
}

// DeleteGestorDocumental realiza un DELETE al gestor documental institucional
// (borrado lógico del documento en Nuxeo, ver gestor_documental_mid.md en memoria).
func DeleteGestorDocumental(token, path string) error {
	body, status, err := doRequest(http.MethodDelete, token, fmt.Sprintf("%s%s", gestorDocumentalURL, path), nil)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("gestor_documental_mid respondió %d: %s", status, string(body))
	}
	return nil
}
