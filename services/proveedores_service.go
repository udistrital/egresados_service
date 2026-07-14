package services

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/udistrital/egresados_service/helpers"
)

// ProveedorAgora es la proyección mínima de un registro de informacion_proveedor.
// Deliberadamente no declara los campos sensibles (datos bancarios, anexos): al no
// estar en el struct, encoding/json los descarta y nunca entran al MID. NumDocumento
// es interno (amarre JIT); al frontend solo se expone ToPublico (RNF-002b).
type ProveedorAgora struct {
	Id            int    `json:"Id"`
	Tipopersona   string `json:"Tipopersona"`  // NATURAL | JURIDICA
	NumDocumento  string `json:"NumDocumento"` // cédula (NATURAL) o NIT (JURIDICA)
	Correo        string `json:"Correo"`
	NomProveedor  string `json:"NomProveedor"`
	Descripcion   string `json:"Descripcion"`
	Direccion     string `json:"Direccion"`
	Web           string `json:"Web"`
	FechaRegistro string `json:"FechaRegistro"` // "2025-01-15 - 05:04:37 PM"
}

// ProveedorPublico es lo único que el MID devuelve al frontend sobre una empresa
// (RNF-002b / Ley 1581): sin documento, datos bancarios ni anexos.
type ProveedorPublico struct {
	AgoraIdExterno int    `json:"agora_id_externo"` // = ProveedorAgora.Id
	RazonSocial    string `json:"razon_social"`     // = NomProveedor
	TipoPersona    string `json:"tipo_persona"`
	Correo         string `json:"correo"`
}

// ToPublico proyecta el proveedor interno a su forma pública (whitelist explícita).
func (p *ProveedorAgora) ToPublico() ProveedorPublico {
	return ProveedorPublico{
		AgoraIdExterno: p.Id,
		RazonSocial:    p.NomProveedor,
		TipoPersona:    p.Tipopersona,
		Correo:         p.Correo,
	}
}

// BuscarProveedoresPorCorreo consulta los proveedores asociados a un correo en
// administrativa_amazon_api. Devuelve 0..N: un correo puede tener varios proveedores.
func BuscarProveedoresPorCorreo(token, correo string) ([]ProveedorAgora, error) {
	if strings.TrimSpace(correo) == "" {
		return nil, fmt.Errorf("correo es requerido para buscar proveedor")
	}
	// El ':' del DSL de query va literal; solo se escapa el valor.
	q := "correo:" + url.QueryEscape(correo)
	var proveedores []ProveedorAgora
	if err := helpers.GetAmazon(token, fmt.Sprintf("/informacion_proveedor?query=%s", q), &proveedores); err != nil {
		return nil, fmt.Errorf("no se pudieron consultar proveedores del correo %s: %v", correo, err)
	}
	return proveedores, nil
}

// BuscarProveedorPorId consulta un proveedor por su id de Ágora
// (empresa.agora_id_externo). Devuelve nil sin error si no existe; los callers
// deben degradar con gracia si falla.
func BuscarProveedorPorId(token, agoraId string) (*ProveedorAgora, error) {
	if strings.TrimSpace(agoraId) == "" {
		return nil, fmt.Errorf("agoraId es requerido para buscar proveedor")
	}
	var proveedores []ProveedorAgora
	q := "id:" + url.QueryEscape(agoraId)
	if err := helpers.GetAmazon(token, fmt.Sprintf("/informacion_proveedor?query=%s", q), &proveedores); err != nil {
		return nil, fmt.Errorf("no se pudo consultar el proveedor %s: %v", agoraId, err)
	}
	if len(proveedores) == 0 {
		return nil, nil
	}
	return &proveedores[0], nil
}

// ProveedoresPublicos mapea una lista de proveedores a su forma pública.
func ProveedoresPublicos(proveedores []ProveedorAgora) []ProveedorPublico {
	out := make([]ProveedorPublico, 0, len(proveedores))
	for i := range proveedores {
		out = append(out, proveedores[i].ToPublico())
	}
	return out
}
