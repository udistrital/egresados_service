package services

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/udistrital/sga_mid_beneficios_egresados/helpers"
)

// EmpresaProvisionada es cada empresa a la que el usuario queda vinculado tras el JIT.
type EmpresaProvisionada struct {
	EmpresaId        int              `json:"empresa_id"`
	UsuarioEmpresaId int              `json:"usuario_empresa_id"`
	Proveedor        ProveedorPublico `json:"proveedor"`
}

// ProvisionResult es el resultado del JIT provisioning de un usuario de empresa.
type ProvisionResult struct {
	UsuarioId int                   `json:"usuario_id"`
	Empresas  []EmpresaProvisionada `json:"empresas"`
}

// ProvisionarEmpresa hace el JIT provisioning de un usuario de empresa al primer login.
// Flujo (C-2b/c): userinfo(token) → userRol(email) → validar que es empresa →
// informacion_proveedor por correo → amarre de identidad → alta idempotente de
// usuario/empresa/usuario_empresa.
//
// SEGURIDAD: el email se deriva del token vía OIDC userinfo (GetUserInfoDeToken), NO
// del body — así un usuario autenticado no puede provisionar la empresa de otro correo.
// El amarre token.documento == NumDocumento (persona NATURAL) añade una segunda barrera;
// un proveedor JURIDICA se valida por coincidencia de correo + aprobación del ciclo de vida.
func ProvisionarEmpresa(token string) (*ProvisionResult, error) {
	info, err := GetUserInfoDeToken(token)
	if err != nil {
		return nil, err
	}
	email := info.Email

	identidad, err := GetUserRol(token, email)
	if err != nil {
		return nil, err // incluye el 400 "Usuario no registrado" (aún no está en WSO2)
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
		// Amarre de identidad: SOLO aplicable si tenemos el documento del usuario. Los
		// usuarios de empresa self-signup NO traen documento (verificado 2026-07-01:
		// userinfo y userRol lo devuelven vacío), así que la barrera real es la
		// coincidencia de correo (garantizada por la query) + aprobación del ciclo de
		// vida. Cuando SÍ haya documento (p. ej. futuro), se exige que coincida en
		// proveedores NATURAL.
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
			EmpresaId: empresaId, UsuarioEmpresaId: ueId, Proveedor: p.ToPublico(),
		})
	}
	if len(res.Empresas) == 0 {
		return nil, fmt.Errorf("ninguna empresa del correo %s pudo validarse: %s", email, strings.Join(rechazos, "; "))
	}
	return res, nil
}

// findOrCreateUsuario busca el usuario de empresa por (sistema_origen, id_externo=sub)
// —los usuarios self-signup no tienen documento— o lo crea como tipo EMP (C-7) con
// documento NULL. Idempotente. Devuelve el id local.
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
	// documento se OMITE a propósito → NULL en BD (la empresa self-signup no tiene cédula).
	nuevo := map[string]interface{}{
		"nombre":         email, // userRol de empresa no trae nombre de persona
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

// findOrCreateEmpresa busca la empresa local por agora_id_externo (idempotente) o la
// crea en estado APROBADA (Ágora es quien la verifica). Devuelve el id local.
func findOrCreateEmpresa(token string, p *ProveedorAgora) (int, error) {
	agoraId := strconv.Itoa(p.Id)
	var existentes []map[string]interface{}
	q := fmt.Sprintf("/empresa?query=AgoraIdExterno:%s,Activo:true&limit=1", url.QueryEscape(agoraId))
	if err := helpers.GetCRUD(token, q, &existentes); err != nil {
		return 0, err
	}
	if len(existentes) > 0 {
		return toInt(firstOf(existentes[0], "id", "Id")), nil
	}
	estadoId, err := ResolverParametroId(token, TipoParamEstadoEmpresa, "APROBADA")
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

// findOrCreateUsuarioEmpresa vincula usuario↔empresa (idempotente). Devuelve el id local.
func findOrCreateUsuarioEmpresa(token string, usuarioId, empresaId int, principal bool) (int, error) {
	var existentes []map[string]interface{}
	q := fmt.Sprintf("/usuario_empresa?query=Usuario.Id:%d,Empresa.Id:%d,Activo:true&limit=1", usuarioId, empresaId)
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
	if err := helpers.PostCRUD(token, "/usuario_empresa", nuevo, &creado); err != nil {
		return 0, err
	}
	return toInt(firstOf(creado, "id", "Id")), nil
}

// EmpresaDeUsuario es cada empresa a la que un usuario tiene acceso (para el selector
// de empresa del frontend, caso 1:N). Sin datos sensibles (RNF-002b).
type EmpresaDeUsuario struct {
	EmpresaId        int    `json:"empresa_id"`
	UsuarioEmpresaId int    `json:"usuario_empresa_id"`
	AgoraIdExterno   string `json:"agora_id_externo,omitempty"`
	RazonSocial      string `json:"razon_social"`
	EstadoEmpresaId  int    `json:"estado_empresa_id"`
	EsPrincipal      bool   `json:"es_principal"`
	Cargo            string `json:"cargo,omitempty"`
}

// GetEmpresasDeUsuario lista las empresas vinculadas a un usuario (paso 5, selector
// multiempresa). Lee usuario_empresa (con la empresa anidada vía RelatedSel) y proyecta.
func GetEmpresasDeUsuario(token string, usuarioId int) ([]EmpresaDeUsuario, error) {
	var vinculos []map[string]interface{}
	q := fmt.Sprintf("/usuario_empresa?query=Usuario.Id:%d,Activo:true&limit=0", usuarioId)
	if err := helpers.GetCRUD(token, q, &vinculos); err != nil {
		return nil, err
	}
	out := make([]EmpresaDeUsuario, 0, len(vinculos))
	for _, v := range vinculos {
		emp, _ := v["empresa"].(map[string]interface{})
		if emp == nil {
			continue // vínculo sin empresa cargada: se omite en vez de exponer basura
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

// PerfilEmpresa es el perfil público de una empresa para el frontend (sección
// "acerca de la empresa" del detalle de beneficio). Whitelist RNF-002b: datos de
// contacto públicos + métricas de actividad; nunca NIT/documento ni datos bancarios.
type PerfilEmpresa struct {
	EmpresaId       int    `json:"empresa_id"`
	RazonSocial     string `json:"razon_social"`
	EstadoEmpresaId int    `json:"estado_empresa_id"`
	CorreoContacto  string `json:"correo_contacto,omitempty"`
	SitioWeb        string `json:"sitio_web,omitempty"`
	Telefono        string `json:"telefono,omitempty"`
	Direccion       string `json:"direccion,omitempty"`
	Descripcion     string `json:"descripcion,omitempty"`
	// AliadoDesde: fecha de registro como proveedor UD en Ágora (solo la fecha).
	AliadoDesde string `json:"aliado_desde,omitempty"`
	// Métricas de actividad en el módulo
	BeneficiosPublicados int `json:"beneficios_publicados"`
	BeneficiosEntregados int `json:"beneficios_entregados"` // solicitudes APROBADAS
}

// GetPerfilEmpresa arma el perfil público: base local + datos públicos de Ágora
// on-demand (C-2b: descripción/web/dirección no se almacenan) + métricas. La consulta
// a Ágora es best-effort: si falla, el perfil sale con lo local.
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

	// Enriquecimiento Ágora (best-effort)
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

	// Métrica: beneficios PUBLICADOS vigentes de la empresa
	if publicadoId, err := ResolverParametroId(token, TipoParamEstadoBeneficio, "PUBLICADO"); err == nil {
		var beneficios []map[string]interface{}
		q := fmt.Sprintf("/beneficio?query=Empresa.Id:%d,EstadoBeneficioId:%d,Activo:true&fields=Id&limit=0", empresaId, publicadoId)
		if err := helpers.GetCRUD(token, q, &beneficios); err == nil {
			perfil.BeneficiosPublicados = len(beneficios)
		}
	}

	// Métrica: beneficios entregados = solicitudes con estado vigente APROBADA.
	// N+1 de getEstadoActual (C-4b), mismo caveat que RN-007/010; optimizable con la
	// vista v_solicitud_estado_vigente si el volumen crece.
	var solicitudes []map[string]interface{}
	q := fmt.Sprintf("/solicitud_beneficio?query=Beneficio.Empresa.Id:%d,Activo:true&fields=Id&limit=0", empresaId)
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
	query := fmt.Sprintf("/solicitud_beneficio?query=Beneficio.Empresa.Id:%d,Activo:true&limit=0", empresaId)
	if err := helpers.GetCRUD(token, query, &solicitudes); err != nil {
		return nil, err
	}

	// RNF-002b: minimizar datos del egresado — solo exponer campos mínimos
	var bandeja []map[string]interface{}
	for _, s := range solicitudes {
		item := map[string]interface{}{
			"id":              s["id"],
			"radicado":        s["radicado"],
			"fecha_solicitud": s["fecha_solicitud"],
			// Lo que el egresado escribió al solicitar (texto plano, ver desdeJSONB)
			"datos_complementarios": desdeJSONB(s["datos_complementarios"]),
		}
		// C-4b: el estado vigente se deriva del historial, no de la solicitud
		if codigo, estadoId, err := getEstadoActual(token, toInt(s["id"])); err == nil {
			item["estado_solicitud_id"] = estadoId
			item["estado_solicitud"] = codigo
		}
		// Del egresado solo exponer nombre y código institucional, nunca teléfono ni programa completo
		if egresado, ok := s["egresado"].(map[string]interface{}); ok {
			if usuario, ok := egresado["usuario"].(map[string]interface{}); ok {
				item["egresado"] = map[string]interface{}{
					"nombre":               usuario["nombre"],
					"codigo_institucional": egresado["codigo_institucional"],
				}
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
	// Normalizar la relación restante a formato {id} para el PUT
	if ua, ok := empresa["usuario_aprobador"].(map[string]interface{}); ok {
		empresa["usuario_aprobador"] = map[string]interface{}{"id": toInt(ua["id"])}
	}
	return empresa, nil
}
