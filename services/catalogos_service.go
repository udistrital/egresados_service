package services

// Los catálogos del módulo viven en el servicio institucional de parámetros (C-1);
// el CRUD local ya no los expone.

// GetCategoriasBeneficio retorna las categorías de beneficio activas (tipo CATEGORIA_BENEFICIO).
func GetCategoriasBeneficio(token string) (interface{}, error) {
	return GetParametrosPorTipo(token, TipoParamCategoriaBeneficio)
}

// GetSectoresEconomicos retorna los sectores económicos activos (tipo SECTOR_ECONOMICO).
func GetSectoresEconomicos(token string) (interface{}, error) {
	return GetParametrosPorTipo(token, TipoParamSectorEconomico)
}
