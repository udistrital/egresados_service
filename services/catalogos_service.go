package services

import "github.com/udistrital/sga_mid_beneficios_egresados/helpers"

// GetCategoriasBeneficio retorna todas las categorías de beneficio activas.
func GetCategoriasBeneficio() (interface{}, error) {
	var result interface{}
	if err := helpers.GetCRUD("/categoria_beneficio", &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetSectoresEconomicos retorna todos los sectores económicos activos.
func GetSectoresEconomicos() (interface{}, error) {
	var result interface{}
	if err := helpers.GetCRUD("/sector_economico", &result); err != nil {
		return nil, err
	}
	return result, nil
}
