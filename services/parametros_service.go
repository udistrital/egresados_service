package services

import (
	"fmt"
	"net/url"
	"os"

	"github.com/udistrital/sga_mid_beneficios_egresados/helpers"
)

// parametrosLocal activa un catálogo de parámetros EN MEMORIA para desarrollo local,
// sin token ni servicio institucional. Se enciende con
// BENEFICIOS_EGRESADOS_MID_PARAMETROS_LOCAL=true. Los códigos son los de la semilla de
// db/schema.sql; los ids son locales y estables y DEBEN coincidir con los que se
// insertan en la BD de desarrollo (empresa.estado_empresa_id, beneficio.*_id, etc.).
var parametrosLocal = os.Getenv("BENEFICIOS_EGRESADOS_MID_PARAMETROS_LOCAL") == "true"

var parametrosSeed = map[string][]map[string]interface{}{
	TipoParamEstadoEmpresa: {
		{"Id": 10, "CodigoAbreviacion": "APROBADA", "Nombre": "Aprobada"},
		{"Id": 11, "CodigoAbreviacion": "PENDIENTE", "Nombre": "Pendiente"},
		{"Id": 12, "CodigoAbreviacion": "SUSPENDIDA", "Nombre": "Suspendida"},
	},
	TipoParamEstadoBeneficio: {
		{"Id": 20, "CodigoAbreviacion": "BORRADOR", "Nombre": "Borrador"},
		{"Id": 21, "CodigoAbreviacion": "PUBLICADO", "Nombre": "Publicado"},
		{"Id": 22, "CodigoAbreviacion": "AGOTADO", "Nombre": "Agotado"},
		{"Id": 23, "CodigoAbreviacion": "VENCIDO", "Nombre": "Vencido"},
		{"Id": 24, "CodigoAbreviacion": "RETIRADO", "Nombre": "Retirado"},
	},
	TipoParamEstadoSolicitud: {
		{"Id": 30, "CodigoAbreviacion": "PENDIENTE", "Nombre": "Pendiente"},
		{"Id": 31, "CodigoAbreviacion": "EN_REVISION", "Nombre": "En revisión"},
		{"Id": 32, "CodigoAbreviacion": "REQUIERE_INFO", "Nombre": "Requiere información"},
		{"Id": 33, "CodigoAbreviacion": "APROBADA", "Nombre": "Aprobada"},
		{"Id": 34, "CodigoAbreviacion": "RECHAZADA", "Nombre": "Rechazada"},
		{"Id": 35, "CodigoAbreviacion": "CANCELADA", "Nombre": "Cancelada"},
	},
	TipoParamCategoriaBeneficio: {
		{"Id": 40, "CodigoAbreviacion": "EDUCACION", "Nombre": "Educación"},
		{"Id": 41, "CodigoAbreviacion": "SALUD", "Nombre": "Salud"},
		{"Id": 42, "CodigoAbreviacion": "RECREACION", "Nombre": "Recreación"},
		{"Id": 43, "CodigoAbreviacion": "EMPLEO", "Nombre": "Empleo"},
		{"Id": 44, "CodigoAbreviacion": "DESCUENTOS", "Nombre": "Descuentos"},
		{"Id": 45, "CodigoAbreviacion": "OTRO", "Nombre": "Otro"},
	},
	TipoParamSectorEconomico: {
		{"Id": 50, "CodigoAbreviacion": "TEC", "Nombre": "Tecnología e Innovación"},
		{"Id": 51, "CodigoAbreviacion": "COM", "Nombre": "Comercio y Retail"},
		{"Id": 52, "CodigoAbreviacion": "OTR", "Nombre": "Otro"},
	},
	TipoParamParametroSistema: {
		{"Id": 60, "CodigoAbreviacion": "LIMITE_SOLIC_ACTIVAS", "Nombre": "Límite solicitudes activas", "Valor": "5"},
		{"Id": 61, "CodigoAbreviacion": "PAGINACION_DEFAULT", "Nombre": "Paginación catálogo", "Valor": "20"},
		{"Id": 62, "CodigoAbreviacion": "JUSTIF_RECHAZO_MIN", "Nombre": "Mínimo justificación rechazo", "Valor": "20"},
	},
}

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
// token: Bearer del request entrante, exigido por el gateway institucional.
func GetParametrosPorTipo(token, codigoTipo string) ([]map[string]interface{}, error) {
	if parametrosLocal {
		if seed, ok := parametrosSeed[codigoTipo]; ok {
			return seed, nil
		}
		return nil, fmt.Errorf("tipo de parámetro local desconocido: %s", codigoTipo)
	}
	query := url.QueryEscape(fmt.Sprintf("tipo_parametro_id.codigo_abreviacion:%s,activo:true", codigoTipo))
	var resp respuestaParametros
	if err := helpers.GetParametros(token, fmt.Sprintf("/parametro/?query=%s&limit=0", query), &resp); err != nil {
		return nil, fmt.Errorf("no se pudo consultar parámetros de tipo %s: %v", codigoTipo, err)
	}
	if !resp.Success {
		return nil, fmt.Errorf("servicio de parámetros respondió error para tipo %s: %s", codigoTipo, resp.Message)
	}
	return resp.Data, nil
}

// ResolverParametroId obtiene el id del parámetro con el codigo_abreviacion dado
// dentro de un tipo_parametro (p. ej. PENDIENTE dentro de ESTADO_SOLICITUD).
func ResolverParametroId(token, codigoTipo, codigo string) (int, error) {
	params, err := GetParametrosPorTipo(token, codigoTipo)
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
func ResolverParametroCodigo(token, codigoTipo string, id int) (string, error) {
	params, err := GetParametrosPorTipo(token, codigoTipo)
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
