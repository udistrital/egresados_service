package services

// Los catálogos del módulo viven en el servicio institucional de parámetros (C-1);
// el CRUD local ya no los expone.

// GetCategoriasBeneficio retorna las categorías de beneficio activas (tipo CATEGORIA_BENEFICIO).
func GetCategoriasBeneficio() (interface{}, error) {
	return GetParametrosPorTipo(TipoParamCategoriaBeneficio)
}

// GetSectoresEconomicos retorna los sectores económicos activos (tipo SECTOR_ECONOMICO).
func GetSectoresEconomicos() (interface{}, error) {
	return GetParametrosPorTipo(TipoParamSectorEconomico)
}
