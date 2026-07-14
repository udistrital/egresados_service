package services

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/udistrital/egresados_service/helpers"
)

// EmpresaProvisionada es cada empresa a la que el usuario queda vinculado tras el JIT.
type EmpresaProvisionada struct {
	EmpresaId        int `json:"empresa_id"`
	UsuarioEmpresaId int `json:"usuario_empresa_id"`
	// Nit va fuera de ProveedorPublico: esta respuesta es la identidad propia de la
	// empresa autenticada, no una vista pública de un tercero (RNF-002b).
	Nit       string           `json:"nit"`
	Proveedor ProveedorPublico `json:"proveedor"`
}

// ProvisionResult es el resultado del JIT provisioning de un usuario de empresa.
type ProvisionResult struct {
	UsuarioId int                   `json:"usuario_id"`
	Empresas  []EmpresaProvisionada `json:"empresas"`
}

// ProvisionarEmpresa hace el JIT provisioning de un usuario de empresa al primer
// login (C-2b/c): userinfo(token) → userRol(email) → informacion_proveedor por
// correo → amarre de identidad → alta idempotente de usuario/empresa/usuario-empresa.
// El email se deriva del token vía OIDC userinfo, nunca del body: un usuario
// autenticado no puede provisionar la empresa de otro correo.
func ProvisionarEmpresa(token string) (*ProvisionResult, error) {
	info, err := GetUserInfoDeToken(token)
	if err != nil {
		return nil, err
	}
	email := info.Email

	identidad, err := GetUserRol(token, email)
	if err != nil {
		return nil, err
	}
	if identidad.EsEgresado() {
		return nil, fmt.Errorf("el usuario %s es egresado, no una empresa", email)
	}

	proveedores, err := BuscarProveedoresPorCorreo(token, email)
	if err != nil {
		return nil, err
	}
	if len(proveedores) == 0 {
		return nil, fmt.Errorf("no hay ninguna empresa asociada al correo %s en el registro de proveedores", email)
	}

	usuarioId, err := findOrCreateUsuario(token, info.Sub, email)
	if err != nil {
		return nil, err
	}

	res := &ProvisionResult{UsuarioId: usuarioId}
	var rechazos []string
	for i := range proveedores {
		p := &proveedores[i]
		// Amarre de identidad solo si el token trae documento (los self-signup no lo
		// tienen); en ese caso la barrera es la coincidencia de correo de la query.
		if identidad.Documento != "" && strings.EqualFold(p.Tipopersona, "NATURAL") && p.NumDocumento != identidad.Documento {
			rechazos = append(rechazos, fmt.Sprintf("proveedor %d: el documento no coincide con el del usuario autenticado", p.Id))
			continue
		}
		empresaId, err := findOrCreateEmpresa(token, p)
		if err != nil {
			return nil, err
		}
		ueId, err := findOrCreateUsuarioEmpresa(token, usuarioId, empresaId, len(res.Empresas) == 0)
		if err != nil {
			return nil, err
		}
		res.Empresas = append(res.Empresas, EmpresaProvisionada{
			EmpresaId: empresaId, UsuarioEmpresaId: ueId, Nit: p.NumDocumento, Proveedor: p.ToPublico(),
		})
	}
	if len(res.Empresas) == 0 {
		return nil, fmt.Errorf("ninguna empresa del correo %s pudo validarse: %s", email, strings.Join(rechazos, "; "))
	}
	return res, nil
}

// findOrCreateUsuario busca el usuario de empresa por (sistema_origen, id_externo=sub)
// o lo crea como tipo EMP con documento NULL. Idempotente.
func findOrCreateUsuario(token, sub, email string) (int, error) {
	if strings.TrimSpace(sub) == "" {
		return 0, fmt.Errorf("el token no expone 'sub' (identificador WSO2) para el JIT")
	}
	var existentes []map[string]interface{}
	q := fmt.Sprintf("/usuario?query=SistemaOrigen:AGORA,IdExterno:%s,Activo:true&limit=1", url.QueryEscape(sub))
	if err := helpers.GetCRUD(token, q, &existentes); err != nil {
		return 0, err
	}
	if len(existentes) > 0 {
		u := existentes[0]
		if tu, _ := firstOf(u, "tipo_usuario", "TipoUsuario").(string); tu != "" && tu != "EMP" {
			return 0, fmt.Errorf("el usuario %s ya existe como tipo %s (no EMP); no se puede vincular como empresa", sub, tu)
		}
		return toInt(firstOf(u, "id", "Id")), nil
	}
	// documento se omite a propósito → NULL en BD (el self-signup no tiene cédula).
	nuevo := map[string]interface{}{
		"nombre":         email,
		"correo":         email,
		"tipo_usuario":   "EMP",
		"id_externo":     sub,
		"sistema_origen": "AGORA",
	}
	var creado map[string]interface{}
	if err := helpers.PostCRUD(token, "/usuario", nuevo, &creado); err != nil {
		return 0, err
	}
	return toInt(firstOf(creado, "id", "Id")), nil
}

// findOrCreateEmpresa busca la empresa local por agora_id_externo o la crea en
// estado ACTIVA (Ágora ya la verificó; no hay flujo de aprobación en el login).
func findOrCreateEmpresa(token string, p *ProveedorAgora) (int, error) {
	agoraId := strconv.Itoa(p.Id)
	// Sin filtro Activo:true: uq_nit_empresa no condiciona sobre activo, así que una
	// fila soft-deleted debe encontrarse y reactivarse en vez de chocar al re-insertar.
	var existentes []map[string]interface{}
	q := fmt.Sprintf("/empresa?query=AgoraIdExterno:%s&limit=1", url.QueryEscape(agoraId))
	if err := helpers.GetCRUD(token, q, &existentes); err != nil {
		return 0, err
	}
	if len(existentes) == 0 {
		// Fallback por NIT (la restricción real en BD): la empresa puede existir con
		// otro agora_id_externo o sin ninguno.
		q = fmt.Sprintf("/empresa?query=Nit:%s&limit=1", url.QueryEscape(p.NumDocumento))
		if err := helpers.GetCRUD(token, q, &existentes); err != nil {
			return 0, err
		}
	}
	if len(existentes) > 0 {
		return actualizarEmpresaDeAgora(token, existentes[0], p, agoraId)
	}
	estadoId, err := ResolverParametroId(token, TipoParamEstadoEmpresa, "ACTIVA")
	if err != nil {
		return 0, err
	}
	nueva := map[string]interface{}{
		"nit":               p.NumDocumento,
		"razon_social":      p.NomProveedor,
		"agora_id_externo":  agoraId,
		"correo_contacto":   p.Correo,
		"estado_empresa_id": estadoId,
	}
	var creada map[string]interface{}
	if err := helpers.PostCRUD(token, "/empresa", nueva, &creada); err != nil {
		return 0, err
	}
	return toInt(firstOf(creada, "id", "Id")), nil
}

// actualizarEmpresaDeAgora sincroniza los datos de Ágora sobre la empresa local y la
// reactiva si estaba soft-deleted; solo hace PUT si algo cambió. Parte del row
// completo porque el PUT del CRUD reemplaza la fila entera. No toca
// estado_empresa_id: el ciclo de vida local lo decide el módulo, no el login.
func actualizarEmpresaDeAgora(token string, existente map[string]interface{}, p *ProveedorAgora, agoraId string) (int, error) {
	id := toInt(firstOf(existente, "id", "Id"))
	cambio := false
	if !asBool(firstOf(existente, "activo", "Activo")) {
		existente["activo"] = true
		cambio = true
	}
	if asString(firstOf(existente, "razon_social", "RazonSocial")) != p.NomProveedor {
		existente["razon_social"] = p.NomProveedor
		cambio = true
	}
	if asString(firstOf(existente, "correo_contacto", "CorreoContacto")) != p.Correo {
		existente["correo_contacto"] = p.Correo
		cambio = true
	}
	if asString(firstOf(existente, "agora_id_externo", "AgoraIdExterno")) != agoraId {
		existente["agora_id_externo"] = agoraId
		cambio = true
	}
	if !cambio {
		return id, nil
	}
	if err := helpers.PutCRUD(token, fmt.Sprintf("/empresa/%d", id), existente); err != nil {
		return 0, err
	}
	return id, nil
}

// findOrCreateUsuarioEmpresa vincula usuario↔empresa (idempotente). Devuelve el id local.
func findOrCreateUsuarioEmpresa(token string, usuarioId, empresaId int, principal bool) (int, error) {
	var existentes []map[string]interface{}
	q := fmt.Sprintf("/usuario-empresa?query=Usuario.Id:%d,Empresa.Id:%d,Activo:true&limit=1", usuarioId, empresaId)
	if err := helpers.GetCRUD(token, q, &existentes); err != nil {
		return 0, err
	}
	if len(existentes) > 0 {
		return toInt(firstOf(existentes[0], "id", "Id")), nil
	}
	nuevo := map[string]interface{}{
		"usuario":      map[string]interface{}{"id": usuarioId},
		"empresa":      map[string]interface{}{"id": empresaId},
		"tipo_usuario": "EMP",
		"es_principal": principal,
	}
	var creado map[string]interface{}
	if err := helpers.PostCRUD(token, "/usuario-empresa", nuevo, &creado); err != nil {
		return 0, err
	}
	return toInt(firstOf(creado, "id", "Id")), nil
}

// EmpresaDeUsuario es cada empresa a la que un usuario tiene acceso (selector
// multiempresa del frontend). Sin datos sensibles (RNF-002b).
type EmpresaDeUsuario struct {
	EmpresaId        int    `json:"empresa_id"`
	UsuarioEmpresaId int    `json:"usuario_empresa_id"`
	AgoraIdExterno   string `json:"agora_id_externo,omitempty"`
	RazonSocial      string `json:"razon_social"`
	EstadoEmpresaId  int    `json:"estado_empresa_id"`
	EsPrincipal      bool   `json:"es_principal"`
	Cargo            string `json:"cargo,omitempty"`
}

// GetEmpresasDeUsuario lista las empresas vinculadas a un usuario.
func GetEmpresasDeUsuario(token string, usuarioId int) ([]EmpresaDeUsuario, error) {
	var vinculos []map[string]interface{}
	q := fmt.Sprintf("/usuario-empresa?query=Usuario.Id:%d,Activo:true&limit=0", usuarioId)
	if err := helpers.GetCRUD(token, q, &vinculos); err != nil {
		return nil, err
	}
	out := make([]EmpresaDeUsuario, 0, len(vinculos))
	for _, v := range vinculos {
		emp, _ := v["empresa"].(map[string]interface{})
		if emp == nil {
			continue
		}
		out = append(out, EmpresaDeUsuario{
			EmpresaId:        toInt(firstOf(emp, "id", "Id")),
			UsuarioEmpresaId: toInt(firstOf(v, "id", "Id")),
			AgoraIdExterno:   asString(firstOf(emp, "agora_id_externo", "AgoraIdExterno")),
			RazonSocial:      asString(firstOf(emp, "razon_social", "RazonSocial")),
			EstadoEmpresaId:  toInt(firstOf(emp, "estado_empresa_id", "EstadoEmpresaId")),
			EsPrincipal:      asBool(firstOf(v, "es_principal", "EsPrincipal")),
			Cargo:            asString(firstOf(v, "cargo", "Cargo")),
		})
	}
	return out, nil
}

// PerfilEmpresa es el perfil público de una empresa. Whitelist RNF-002b: datos de
// contacto públicos + métricas; nunca NIT/documento ni datos bancarios.
type PerfilEmpresa struct {
	EmpresaId       int    `json:"empresa_id"`
	RazonSocial     string `json:"razon_social"`
	EstadoEmpresaId int    `json:"estado_empresa_id"`
	CorreoContacto  string `json:"correo_contacto,omitempty"`
	SitioWeb        string `json:"sitio_web,omitempty"`
	Telefono        string `json:"telefono,omitempty"`
	Direccion       string `json:"direccion,omitempty"`
	Descripcion     string `json:"descripcion,omitempty"`
	// AliadoDesde: fecha de registro como proveedor UD en Ágora.
	AliadoDesde          string `json:"aliado_desde,omitempty"`
	BeneficiosPublicados int    `json:"beneficios_publicados"`
	BeneficiosEntregados int    `json:"beneficios_entregados"` // solicitudes APROBADAS
}

// GetPerfilEmpresa arma el perfil público: base local + datos de Ágora on-demand
// (C-2b) + métricas. La consulta a Ágora es best-effort.
func GetPerfilEmpresa(token string, empresaId int) (*PerfilEmpresa, error) {
	var empresa map[string]interface{}
	if err := helpers.GetCRUD(token, fmt.Sprintf("/empresa/%d", empresaId), &empresa); err != nil {
		return nil, fmt.Errorf("empresa %d no encontrada", empresaId)
	}
	perfil := &PerfilEmpresa{
		EmpresaId:       empresaId,
		RazonSocial:     asString(firstOf(empresa, "razon_social", "RazonSocial")),
		EstadoEmpresaId: toInt(firstOf(empresa, "estado_empresa_id", "EstadoEmpresaId")),
		CorreoContacto:  asString(firstOf(empresa, "correo_contacto", "CorreoContacto")),
		SitioWeb:        asString(firstOf(empresa, "sitio_web", "SitioWeb")),
		Telefono:        asString(firstOf(empresa, "telefono_contacto", "TelefonoContacto")),
		Direccion:       asString(firstOf(empresa, "direccion", "Direccion")),
	}

	if agoraId := asString(firstOf(empresa, "agora_id_externo", "AgoraIdExterno")); agoraId != "" {
		if p, err := BuscarProveedorPorId(token, agoraId); err == nil && p != nil {
			perfil.Descripcion = p.Descripcion
			if p.Web != "" {
				perfil.SitioWeb = p.Web
			}
			if p.Direccion != "" {
				perfil.Direccion = p.Direccion
			}
			// "2025-01-15 - 05:04:37 PM" → "2025-01-15"
			if partes := strings.SplitN(p.FechaRegistro, " ", 2); partes[0] != "" {
				perfil.AliadoDesde = partes[0]
			}
		}
	}

	if publicadoId, err := ResolverParametroId(token, TipoParamEstadoBeneficio, "PUBLICADO"); err == nil {
		var beneficios []map[string]interface{}
		q := fmt.Sprintf("/beneficio?query=Empresa.Id:%d,EstadoBeneficioId:%d,Activo:true&fields=Id&limit=0", empresaId, publicadoId)
		if err := helpers.GetCRUD(token, q, &beneficios); err == nil {
			perfil.BeneficiosPublicados = len(beneficios)
		}
	}

	// Beneficios entregados = solicitudes con estado vigente APROBADA.
	var solicitudes []map[string]interface{}
	q := fmt.Sprintf("/solicitud-beneficio?query=Beneficio.Empresa.Id:%d,Activo:true&fields=Id&limit=0", empresaId)
	if err := helpers.GetCRUD(token, q, &solicitudes); err == nil {
		for _, s := range solicitudes {
			if codigo, _, err := getEstadoActual(token, toInt(firstOf(s, "id", "Id"))); err == nil && codigo == estadoAprobada {
				perfil.BeneficiosEntregados++
			}
		}
	}
	return perfil, nil
}

// GetBandejaEmpresa retorna las solicitudes recibidas por la empresa.
// Solo expone campos mínimos del egresado (RNF-002b / Ley 1581).
func GetBandejaEmpresa(token string, empresaId int) (interface{}, error) {
	var solicitudes []map[string]interface{}
	query := fmt.Sprintf("/solicitud-beneficio?query=Beneficio.Empresa.Id:%d,Activo:true&limit=0", empresaId)
	if err := helpers.GetCRUD(token, query, &solicitudes); err != nil {
		return nil, err
	}

	// Caché por request (código institucional → carrera).
	carrerasPorCodigo := map[string]string{}
	var bandeja []map[string]interface{}
	for _, s := range solicitudes {
		item := map[string]interface{}{
			"id":                    s["id"],
			"radicado":              s["radicado"],
			"fecha_solicitud":       s["fecha_solicitud"],
			"datos_complementarios": desdeJSONB(s["datos_complementarios"]),
		}
		if codigo, estadoId, err := getEstadoActual(token, toInt(s["id"])); err == nil {
			item["estado_solicitud_id"] = estadoId
			item["estado_solicitud"] = codigo
		}
		// RNF-002b: del egresado solo nombre, código institucional y carrera.
		if egresado, ok := s["egresado"].(map[string]interface{}); ok {
			if usuario, ok := egresado["usuario"].(map[string]interface{}); ok {
				egresadoOut := map[string]interface{}{
					"nombre":               usuario["nombre"],
					"codigo_institucional": egresado["codigo_institucional"],
				}
				if codigo := asString(egresado["codigo_institucional"]); codigo != "" {
					carrera, cacheada := carrerasPorCodigo[codigo]
					if !cacheada {
						resuelta, err := ResolverCarrera(token, codigo)
						if err != nil {
							resuelta = ""
						}
						carrerasPorCodigo[codigo] = resuelta
						carrera = resuelta
					}
					if carrera != "" {
						egresadoOut["programa_academico"] = carrera
					}
				}
				item["egresado"] = egresadoOut
			}
		}
		if beneficio, ok := s["beneficio"].(map[string]interface{}); ok {
			item["beneficio"] = map[string]interface{}{
				"id":     beneficio["id"],
				"titulo": beneficio["titulo"],
			}
		}
		bandeja = append(bandeja, item)
	}
	return bandeja, nil
}

// SuspenderEmpresa cambia el estado de la empresa a SUSPENDIDA.
func SuspenderEmpresa(token string, id int) error {
	estadoId, err := ResolverParametroId(token, TipoParamEstadoEmpresa, "SUSPENDIDA")
	if err != nil {
		return err
	}
	empresa, err := getEmpresaBase(token, id)
	if err != nil {
		return err
	}
	empresa["estado_empresa_id"] = estadoId
	return helpers.PutCRUD(token, fmt.Sprintf("/empresa/%d", id), empresa)
}

// getEmpresaBase obtiene la empresa del CRUD lista para ser actualizada.
func getEmpresaBase(token string, id int) (map[string]interface{}, error) {
	var empresa map[string]interface{}
	if err := helpers.GetCRUD(token, fmt.Sprintf("/empresa/%d", id), &empresa); err != nil {
		return nil, fmt.Errorf("empresa %d no encontrada", id)
	}
	// Normalizar la relación a formato {id} para el PUT.
	if ua, ok := empresa["usuario_aprobador"].(map[string]interface{}); ok {
		empresa["usuario_aprobador"] = map[string]interface{}{"id": toInt(ua["id"])}
	}
	return empresa, nil
}
