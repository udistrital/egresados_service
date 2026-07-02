package services

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/udistrital/sga_mid_beneficios_egresados/helpers"
)

// ProvisionEgresadoResult es el resultado del JIT provisioning de un egresado.
type ProvisionEgresadoResult struct {
	UsuarioId           int    `json:"usuario_id"`
	EgresadoId          int    `json:"egresado_id"`
	CodigoInstitucional string `json:"codigo_institucional"`
	Nombre              string `json:"nombre,omitempty"`
}

// ProvisionarEgresado hace el JIT provisioning del egresado al primer login
// (contraparte de ProvisionarEmpresa). Flujo: userinfo(token) → userRol(email) →
// validar que es egresado (Estado == "E") → terceros_crud (nombre real, C-2a) →
// alta idempotente de usuario (tipo EGR) + egresado local.
//
// SEGURIDAD: email y documento se derivan del token vía OIDC userinfo — un usuario
// autenticado no puede provisionar el perfil de otra persona. El documento es la
// llave de identidad del egresado (los egresados SIEMPRE lo traen; verificado
// 2026-07-01 con clalapea).
//
// programa/facultad NO se almacenan (C-2a: on-demand del SGA); fecha_grado queda
// NULL (no hay fuente institucional, verificado 2026-06-10).
func ProvisionarEgresado(token string) (*ProvisionEgresadoResult, error) {
	info, err := GetUserInfoDeToken(token)
	if err != nil {
		return nil, err
	}

	identidad, err := GetUserRol(token, info.Email)
	if err != nil {
		return nil, err
	}
	if !identidad.EsEgresado() {
		return nil, fmt.Errorf("el usuario %s no es egresado (Estado %q); el registro de empresa es POST /v1/empresas/provision", info.Email, identidad.Estado)
	}

	documento := strings.TrimSpace(info.Documento)
	if documento == "" {
		documento = strings.TrimSpace(identidad.Documento)
	}
	if documento == "" {
		return nil, fmt.Errorf("el token de %s no expone documento; no se puede amarrar la identidad del egresado", info.Email)
	}

	// Nombre real desde terceros_crud (best effort: si falla, se degrada al email).
	nombre, terceroId := buscarTerceroPorDocumento(token, documento)
	if nombre == "" {
		nombre = info.Email
	}

	codigo := strings.TrimSpace(identidad.Codigo)
	if codigo == "" && terceroId > 0 {
		codigo = buscarCodigoInstitucional(token, terceroId)
	}
	if codigo == "" {
		return nil, fmt.Errorf("no se pudo determinar el código institucional de %s (userRol sin Codigo y consultar_persona sin códigos)", info.Email)
	}

	usuarioId, err := findOrCreateUsuarioEgresado(token, documento, info.Sub, info.Email, nombre)
	if err != nil {
		return nil, err
	}
	egresadoId, err := findOrCreateEgresado(token, usuarioId, codigo)
	if err != nil {
		return nil, err
	}

	return &ProvisionEgresadoResult{
		UsuarioId:           usuarioId,
		EgresadoId:          egresadoId,
		CodigoInstitucional: codigo,
		Nombre:              nombre,
	}, nil
}

// buscarTerceroPorDocumento resuelve nombre real y TerceroId en terceros_crud
// (misma consulta que sga_cliente: datos_identificacion?query=Activo:true,Numero:{doc}).
// Best effort: cualquier fallo devuelve ("", 0) y el caller degrada.
func buscarTerceroPorDocumento(token, documento string) (nombre string, terceroId int) {
	var datos []struct {
		TerceroId struct {
			Id              int    `json:"Id"`
			NombreCompleto  string `json:"NombreCompleto"`
			PrimerNombre    string `json:"PrimerNombre"`
			SegundoNombre   string `json:"SegundoNombre"`
			PrimerApellido  string `json:"PrimerApellido"`
			SegundoApellido string `json:"SegundoApellido"`
		} `json:"TerceroId"`
	}
	q := fmt.Sprintf("/datos_identificacion?query=Activo:true,Numero:%s&limit=1", url.QueryEscape(documento))
	if err := helpers.GetTerceros(token, q, &datos); err != nil || len(datos) == 0 {
		return "", 0
	}
	t := datos[0].TerceroId
	nombre = strings.TrimSpace(t.NombreCompleto)
	if nombre == "" {
		var partes []string
		for _, p := range []string{t.PrimerNombre, t.SegundoNombre, t.PrimerApellido, t.SegundoApellido} {
			if p = strings.TrimSpace(p); p != "" {
				partes = append(partes, p)
			}
		}
		nombre = strings.Join(partes, " ")
	}
	return nombre, t.Id
}

// buscarCodigoInstitucional es el fallback cuando userRol no trae Codigo: consulta
// sga_mid consultar_persona (C-2a) con el TerceroId. Tolera las dos variantes de
// respuesta (mismo parseo defensivo que perfil-api.service.ts del frontend).
func buscarCodigoInstitucional(token string, terceroId int) string {
	var res map[string]interface{}
	path := fmt.Sprintf("/derechos_pecuniarios/consultar_persona/%d", terceroId)
	if err := helpers.GetSgaMid(token, path, &res); err != nil {
		return ""
	}
	data, _ := res["Data"].(map[string]interface{})
	if data == nil {
		return ""
	}
	// Variante documentada: Data.Codigos[] — se prefiere el código con Activo=false
	// (condición de egresado, C-2a); si no hay inactivos, el primero con Dato.
	if codigos, ok := data["Codigos"].([]interface{}); ok {
		var primero string
		for _, c := range codigos {
			m, _ := c.(map[string]interface{})
			if m == nil {
				continue
			}
			dato := strings.TrimSpace(asString(m["Dato"]))
			if dato == "" {
				continue
			}
			if activo, esBool := m["Activo"].(bool); esBool && !activo {
				return dato
			}
			if primero == "" {
				primero = dato
			}
		}
		if primero != "" {
			return primero
		}
	}
	// Variante plana observada (2026-06-10): el código viene en NumeroIdentificacion.
	return strings.TrimSpace(asString(data["NumeroIdentificacion"]))
}

// findOrCreateUsuarioEgresado busca el usuario por documento (llave institucional del
// egresado, UNIQUE en BD) o lo crea como tipo EGR (C-7). Idempotente: relogin no duplica.
func findOrCreateUsuarioEgresado(token, documento, sub, email, nombre string) (int, error) {
	var existentes []map[string]interface{}
	q := fmt.Sprintf("/usuario?query=Documento:%s,Activo:true&limit=1", url.QueryEscape(documento))
	if err := helpers.GetCRUD(token, q, &existentes); err != nil {
		return 0, err
	}
	if len(existentes) > 0 {
		u := existentes[0]
		if tu, _ := firstOf(u, "tipo_usuario", "TipoUsuario").(string); tu != "" && tu != "EGR" {
			return 0, fmt.Errorf("el documento %s ya existe como usuario tipo %s (no EGR); no se puede vincular como egresado", documento, tu)
		}
		return toInt(firstOf(u, "id", "Id")), nil
	}
	nuevo := map[string]interface{}{
		"documento":      documento,
		"nombre":         nombre,
		"correo":         email,
		"tipo_usuario":   "EGR",
		"id_externo":     sub,
		"sistema_origen": "SGA",
	}
	var creado map[string]interface{}
	if err := helpers.PostCRUD(token, "/usuario", nuevo, &creado); err != nil {
		return 0, err
	}
	id := toInt(firstOf(creado, "id", "Id"))
	if id <= 0 {
		return 0, fmt.Errorf("el CRUD no devolvió id al crear el usuario (respuesta: %v)", creado)
	}
	return id, nil
}

// findOrCreateEgresado busca el perfil de egresado por usuario (UNIQUE usuario_id) o lo
// crea con el código institucional. Idempotente. Devuelve el id local (egresado_id).
func findOrCreateEgresado(token string, usuarioId int, codigo string) (int, error) {
	var existentes []map[string]interface{}
	q := fmt.Sprintf("/egresado?query=Usuario.Id:%d,Activo:true&limit=1", usuarioId)
	if err := helpers.GetCRUD(token, q, &existentes); err != nil {
		return 0, err
	}
	if len(existentes) > 0 {
		return toInt(firstOf(existentes[0], "id", "Id")), nil
	}
	nuevo := map[string]interface{}{
		"usuario":              map[string]interface{}{"id": usuarioId},
		"tipo_usuario":         "EGR",
		"codigo_institucional": codigo,
	}
	var creado map[string]interface{}
	if err := helpers.PostCRUD(token, "/egresado", nuevo, &creado); err != nil {
		return 0, err
	}
	id := toInt(firstOf(creado, "id", "Id"))
	if id <= 0 {
		return 0, fmt.Errorf("el CRUD no devolvió id al crear el egresado (respuesta: %v)", creado)
	}
	return id, nil
}
