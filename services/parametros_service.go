package services

import (
	"fmt"
	"net/url"

	"github.com/udistrital/sga_mid_beneficios_egresados/helpers"
)

// Códigos de tipo_parametro del módulo en el servicio institucional de parámetros (C-1).
// Definidos en db/schema.sql (semilla de parametro.tipo_parametro).
const (
	TipoParamTipoUsuario        = "TIPO_USUARIO"
	TipoParamEstadoEmpresa      = "ESTADO_EMPRESA"
	TipoParamEstadoBeneficio    = "ESTADO_BENEFICIO"
	TipoParamEstadoSolicitud    = "ESTADO_SOLICITUD"
	TipoParamCategoriaBeneficio = "CATEGORIA_BENEFICIO"
	TipoParamSectorEconomico    = "SECTOR_ECONOMICO"
	TipoParamParametroSistema   = "PARAMETRO_SISTEMA"
)

// respuestaParametros formato estándar del servicio de parámetros:
// { "Success": true, "Status": "200", "Message": "...", "Data": [...] }
type respuestaParametros struct {
	Success bool                     `json:"Success"`
	Status  string                   `json:"Status"`
	Message string                   `json:"Message"`
	Data    []map[string]interface{} `json:"Data"`
}

// GetParametrosPorTipo retorna los parámetros activos de un tipo_parametro
// identificado por su codigo_abreviacion (p. ej. ESTADO_SOLICITUD).
func GetParametrosPorTipo(codigoTipo string) ([]map[string]interface{}, error) {
	query := url.QueryEscape(fmt.Sprintf("tipo_parametro_id.codigo_abreviacion:%s,activo:true", codigoTipo))
	var resp respuestaParametros
	if err := helpers.GetParametros(fmt.Sprintf("/parametro/?query=%s&limit=0", query), &resp); err != nil {
		return nil, fmt.Errorf("no se pudo consultar parámetros de tipo %s: %v", codigoTipo, err)
	}
	if !resp.Success {
		return nil, fmt.Errorf("servicio de parámetros respondió error para tipo %s: %s", codigoTipo, resp.Message)
	}
	return resp.Data, nil
}

// ResolverParametroId obtiene el id del parámetro con el codigo_abreviacion dado
// dentro de un tipo_parametro (p. ej. PENDIENTE dentro de ESTADO_SOLICITUD).
func ResolverParametroId(codigoTipo, codigo string) (int, error) {
	params, err := GetParametrosPorTipo(codigoTipo)
	if err != nil {
		return 0, err
	}
	for _, p := range params {
		if p["CodigoAbreviacion"] == codigo || p["codigo_abreviacion"] == codigo {
			return toInt(firstOf(p, "Id", "id")), nil
		}
	}
	return 0, fmt.Errorf("parámetro %s no encontrado en tipo %s", codigo, codigoTipo)
}

// ResolverParametroCodigo operación inversa: obtiene el codigo_abreviacion de un
// parámetro por su id dentro de un tipo_parametro.
func ResolverParametroCodigo(codigoTipo string, id int) (string, error) {
	params, err := GetParametrosPorTipo(codigoTipo)
	if err != nil {
		return "", err
	}
	for _, p := range params {
		if toInt(firstOf(p, "Id", "id")) == id {
			if codigo, ok := firstOf(p, "CodigoAbreviacion", "codigo_abreviacion").(string); ok {
				return codigo, nil
			}
		}
	}
	return "", fmt.Errorf("parámetro con id %d no encontrado en tipo %s", id, codigoTipo)
}

// firstOf retorna el primer valor presente entre varias llaves alternativas
// (el servicio institucional serializa en PascalCase; la semilla local en snake_case).
func firstOf(m map[string]interface{}, keys ...string) interface{} {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			return v
		}
	}
	return nil
}
