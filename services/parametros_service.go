package services

import (
	"fmt"
	"net/url"

	"github.com/beego/beego/v2/server/web"
	"github.com/udistrital/egresados_service/helpers"
)

// parametrosLocal activa un catálogo de parámetros en memoria para desarrollo local
// (ParametrosLocal=true en conf/app.conf), sin token ni servicio institucional.
var parametrosLocal = web.AppConfig.DefaultString("ParametrosLocal", "false") == "true"

// parametrosSeed espeja los parámetros reales del servicio institucional con sus
// mismos ids, así el modo local y el real son intercambiables.
var parametrosSeed = map[string][]map[string]interface{}{
	TipoParamEstadoEmpresa: {
		{"Id": 7199, "CodigoAbreviacion": "ACTIVA", "Nombre": "Activa"},
		{"Id": 7200, "CodigoAbreviacion": "SUSPENDIDA", "Nombre": "Suspendida"},
	},
	TipoParamEstadoBeneficio: {
		{"Id": 7201, "CodigoAbreviacion": "BORRADOR", "Nombre": "Borrador"},
		{"Id": 7202, "CodigoAbreviacion": "PUBLICADO", "Nombre": "Publicado"},
		{"Id": 7203, "CodigoAbreviacion": "AGOTADO", "Nombre": "Agotado"},
		{"Id": 7204, "CodigoAbreviacion": "VENCIDO", "Nombre": "Vencido"},
		{"Id": 7205, "CodigoAbreviacion": "RETIRADO", "Nombre": "Retirado"},
	},
	TipoParamEstadoSolicitud: {
		{"Id": 7206, "CodigoAbreviacion": "PENDIENTE", "Nombre": "Pendiente"},
		{"Id": 7207, "CodigoAbreviacion": "EN_REVISION", "Nombre": "En revisión"},
		{"Id": 7208, "CodigoAbreviacion": "REQUIERE_INFO", "Nombre": "Requiere información"},
		{"Id": 7209, "CodigoAbreviacion": "APROBADA", "Nombre": "Aprobada"},
		{"Id": 7210, "CodigoAbreviacion": "RECHAZADA", "Nombre": "Rechazada"},
		{"Id": 7211, "CodigoAbreviacion": "CANCELADA", "Nombre": "Cancelada"},
	},
	TipoParamCategoriaBeneficio: {
		{"Id": 7212, "CodigoAbreviacion": "EDUCACION", "Nombre": "Educación"},
		{"Id": 7213, "CodigoAbreviacion": "SALUD", "Nombre": "Salud"},
		{"Id": 7214, "CodigoAbreviacion": "RECREACION", "Nombre": "Recreación"},
		{"Id": 7215, "CodigoAbreviacion": "EMPLEO", "Nombre": "Empleo"},
		{"Id": 7216, "CodigoAbreviacion": "DESCUENTOS", "Nombre": "Descuentos"},
		{"Id": 7217, "CodigoAbreviacion": "OTRO", "Nombre": "Otro"},
	},
	// Subconjunto local; el servicio real tiene 10 sectores (7218-7227).
	TipoParamSectorEconomico: {
		{"Id": 7218, "CodigoAbreviacion": "TEC", "Nombre": "Tecnología e Innovación"},
		{"Id": 7222, "CodigoAbreviacion": "COM", "Nombre": "Comercio y Retail"},
		{"Id": 7227, "CodigoAbreviacion": "OTR", "Nombre": "Otro"},
	},
	// Códigos ≤20 chars (codigo_abreviacion institucional es varchar(20)). "Valor"
	// solo existe en esta semilla: la tabla institucional no tiene columna de valor,
	// contra el servicio real el MID usa sus defaults.
	TipoParamParametroSistema: {
		{"Id": 7228, "CodigoAbreviacion": "LIMITE_SOLIC_ACTIVAS", "Nombre": "Límite solicitudes activas", "Valor": "5"},
		{"Id": 7229, "CodigoAbreviacion": "PAGINACION_DEFAULT", "Nombre": "Paginación catálogo", "Valor": "20"},
		{"Id": 7230, "CodigoAbreviacion": "JUSTIF_RECHAZO_MIN", "Nombre": "Mínimo justificación rechazo", "Valor": "20"},
	},
}

// Códigos de tipo_parametro del módulo en el servicio institucional de parámetros (C-1).
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
// identificado por su codigo_abreviacion.
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
// dentro de un tipo_parametro.
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
