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

	// Empresa autenticada vía Ágora → queda APROBADA directamente.
	// estado_empresa_id y sector_economico_id son ids de parámetro planos (C-1).
	estadoId, err := ResolverParametroId(TipoParamEstadoEmpresa, "APROBADA")
	if err != nil {
		return nil, err
	}
	body["estado_empresa_id"] = estadoId

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
	query := fmt.Sprintf("/solicitud_beneficio?query=Beneficio.Empresa.Id:%d,Activo:true&limit=0", empresaId)
	if err := helpers.GetCRUD(query, &solicitudes); err != nil {
		return nil, err
	}

	// RNF-002b: minimizar datos del egresado — solo exponer campos mínimos
	var bandeja []map[string]interface{}
	for _, s := range solicitudes {
		item := map[string]interface{}{
			"id":              s["id"],
			"radicado":        s["radicado"],
			"fecha_solicitud": s["fecha_solicitud"],
		}
		// C-4b: el estado vigente se deriva del historial, no de la solicitud
		if codigo, estadoId, err := getEstadoActual(toInt(s["id"])); err == nil {
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
func SuspenderEmpresa(id int) error {
	estadoId, err := ResolverParametroId(TipoParamEstadoEmpresa, "SUSPENDIDA")
	if err != nil {
		return err
	}
	empresa, err := getEmpresaBase(id)
	if err != nil {
		return err
	}
	empresa["estado_empresa_id"] = estadoId
	return helpers.PutCRUD(fmt.Sprintf("/empresa/%d", id), empresa)
}

// getEmpresaBase obtiene la empresa del CRUD lista para ser actualizada.
func getEmpresaBase(id int) (map[string]interface{}, error) {
	var empresa map[string]interface{}
	if err := helpers.GetCRUD(fmt.Sprintf("/empresa/%d", id), &empresa); err != nil {
		return nil, fmt.Errorf("empresa %d no encontrada", id)
	}
	// Normalizar la relación restante a formato {id} para el PUT
	if ua, ok := empresa["usuario_aprobador"].(map[string]interface{}); ok {
		empresa["usuario_aprobador"] = map[string]interface{}{"id": toInt(ua["id"])}
	}
	return empresa, nil
}
