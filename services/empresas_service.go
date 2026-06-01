package services

import (
	"fmt"

	"github.com/udistrital/sga_mid_beneficios_egresados/helpers"
)

// RegistrarEmpresa hace JIT provisioning de una empresa autenticada vía Ágora/WSO2.
// La empresa queda directamente en estado APROBADA ya que Ágora es quien la verifica.
func RegistrarEmpresa(body map[string]interface{}) (interface{}, error) {
	// Validar campos obligatorios
	required := []string{"nit", "razon_social"}
	for _, field := range required {
		if v, ok := body[field]; !ok || v == nil || v == "" {
			return nil, fmt.Errorf("campo obligatorio faltante: %s", field)
		}
	}

	// Empresa autenticada vía Ágora → queda APROBADA directamente
	estadoId, err := resolverIdEstadoEmpresa("APROBADA")
	if err != nil {
		return nil, err
	}
	body["estado_empresa"] = map[string]interface{}{"id": estadoId}

	// Transformar sector_economico_id al formato objeto que espera el CRUD
	if sid, ok := body["sector_economico_id"]; ok && sid != nil {
		body["sector_economico"] = map[string]interface{}{"id": toInt(sid)}
		delete(body, "sector_economico_id")
	}

	var result interface{}
	if err := helpers.PostCRUD("/empresa", body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetBandejaEmpresa retorna las solicitudes recibidas por la empresa.
// Solo expone campos mínimos del egresado (RNF-002b / Ley 1581).
func GetBandejaEmpresa(empresaId int) (interface{}, error) {
	var solicitudes []map[string]interface{}
	query := fmt.Sprintf("/solicitud_beneficio?query=Beneficio.Empresa.Id:%d,Activo:true", empresaId)
	if err := helpers.GetCRUD(query, &solicitudes); err != nil {
		return nil, err
	}

	// RNF-002b: minimizar datos del egresado — solo exponer campos mínimos
	var bandeja []map[string]interface{}
	for _, s := range solicitudes {
		item := map[string]interface{}{
			"id":               s["id"],
			"radicado":         s["radicado"],
			"estado_solicitud": s["estado_solicitud"],
			"fecha_solicitud":  s["fecha_solicitud"],
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
func SuspenderEmpresa(id int) error {
	estadoId, err := resolverIdEstadoEmpresa("SUSPENDIDA")
	if err != nil {
		return err
	}
	empresa, err := getEmpresaBase(id)
	if err != nil {
		return err
	}
	empresa["estado_empresa"] = map[string]interface{}{"id": estadoId}
	return helpers.PutCRUD(fmt.Sprintf("/empresa/%d", id), empresa)
}

// resolverIdEstadoEmpresa obtiene el id del estado por su codigo_abreviacion.
func resolverIdEstadoEmpresa(codigo string) (int, error) {
	var estados []map[string]interface{}
	if err := helpers.GetCRUD("/estado_empresa", &estados); err != nil {
		return 0, fmt.Errorf("no se pudo obtener estados de empresa")
	}
	for _, e := range estados {
		if e["codigo_abreviacion"] == codigo {
			return toInt(e["id"]), nil
		}
	}
	return 0, fmt.Errorf("estado %s no encontrado", codigo)
}

// getEmpresaBase obtiene la empresa del CRUD lista para ser actualizada.
func getEmpresaBase(id int) (map[string]interface{}, error) {
	var empresa map[string]interface{}
	if err := helpers.GetCRUD(fmt.Sprintf("/empresa/%d", id), &empresa); err != nil {
		return nil, fmt.Errorf("empresa %d no encontrada", id)
	}
	// Normalizar relaciones a formato {id} para el PUT
	if ee, ok := empresa["estado_empresa"].(map[string]interface{}); ok {
		empresa["estado_empresa"] = map[string]interface{}{"id": toInt(ee["id"])}
	}
	if se, ok := empresa["sector_economico"].(map[string]interface{}); ok {
		empresa["sector_economico"] = map[string]interface{}{"id": toInt(se["id"])}
	}
	return empresa, nil
}
