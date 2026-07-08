package services

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/udistrital/egresados_service/helpers"
)

// ProveedorAgora es la proyección MÍNIMA de un registro de informacion_proveedor
// (administrativa_amazon_api). Deliberadamente NO declara los campos sensibles del
// servicio (NumCuentaBancaria, IdEntidadBancaria, TipoCuentaBancaria, Anexorut,
// Anexorup, datos del asesor): al no estar en el struct, encoding/json los DESCARTA
// al deserializar y nunca entran siquiera a la memoria del MID.
// NumDocumento se conserva porque es interno del MID (amarre JIT token.documento ==
// NumDocumento); NO se expone al frontend — para eso está ToPublico (RNF-002b).
// Descripcion/Direccion/Web/FechaRegistro son datos públicos de la empresa (alimentan
// el "acerca de la empresa" del detalle de beneficio).
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

// ProveedorPublico es lo ÚNICO que el MID puede devolver al frontend sobre una empresa
// (RNF-002b / Ley 1581). Sin documento, sin datos bancarios, sin anexos.
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
// administrativa_amazon_api. Devuelve 0..N (un correo puede tener varios proveedores,
// caso construdenco — de ahí que sea slice y no un único registro).
// token: Bearer del request entrante (exigido por el gateway).
func BuscarProveedoresPorCorreo(token, correo string) ([]ProveedorAgora, error) {
	if strings.TrimSpace(correo) == "" {
		return nil, fmt.Errorf("correo es requerido para buscar proveedor")
	}
	// El ':' del DSL de query va literal (forma verificada con 200 el 2026-07-01);
	// solo se escapa el valor (el correo) para no romper con caracteres especiales.
	q := "correo:" + url.QueryEscape(correo)
	var proveedores []ProveedorAgora
	if err := helpers.GetAmazon(token, fmt.Sprintf("/informacion_proveedor?query=%s", q), &proveedores); err != nil {
		return nil, fmt.Errorf("no se pudieron consultar proveedores del correo %s: %v", correo, err)
	}
	return proveedores, nil
}

// BuscarProveedorPorId consulta un proveedor por su id de Ágora (empresa.agora_id_externo).
// Devuelve nil (sin error) si no existe. La clave de query en minúscula sigue el patrón
// verificado con `correo:` (2026-07-01); la variante por id no se ha probado con token
// vivo — los callers deben degradar con gracia si falla.
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

// ProveedoresPublicos mapea una lista de proveedores a su forma pública (para exponer
// al frontend, p. ej. en el selector de empresa del caso 1:N).
func ProveedoresPublicos(proveedores []ProveedorAgora) []ProveedorPublico {
	out := make([]ProveedorPublico, 0, len(proveedores))
	for i := range proveedores {
		out = append(out, proveedores[i].ToPublico())
	}
	return out
}
