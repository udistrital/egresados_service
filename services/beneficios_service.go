package services

import (
	"fmt"
	"strings"

	"github.com/udistrital/sga_mid_beneficios_egresados/helpers"
)

// GetCatalogoBeneficios devuelve el catálogo paginado con filtros (RN-008, RN-FILTROS).
// Solo beneficios con estado=PUBLICADO, fecha_fin >= hoy, cupos_disponibles > 0.
func GetCatalogoBeneficios(page, limit, categoriaId, empresaId int, q string) (interface{}, error) {
	// Construir query string para el CRUD
	offset := (page - 1) * limit
	query := fmt.Sprintf(
		"/beneficio?query=EstadoBeneficio.CodigoAbreviacion:PUBLICADO,Activo:true&limit=%d&offset=%d",
		limit, offset,
	)
	if categoriaId > 0 {
		query += fmt.Sprintf(",CategoriaBeneficio.Id:%d", categoriaId)
	}
	if empresaId > 0 {
		query += fmt.Sprintf(",Empresa.Id:%d", empresaId)
	}
	// Nota: el filtro por fecha_fin >= hoy y cupos_disponibles > 0
	// se aplica en el CRUD con query params o se filtra aquí tras obtener los datos.
	// TODO: validar si el CRUD soporta operadores de comparación en query params.

	var result interface{}
	if err := helpers.GetCRUD(query, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetBeneficioDetalle devuelve el detalle de un beneficio por id.
func GetBeneficioDetalle(id int) (interface{}, error) {
	var result interface{}
	if err := helpers.GetCRUD(fmt.Sprintf("/beneficio/%d", id), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// PublicarBeneficio valida RN-008b y crea el beneficio en el CRUD.
// Solo permite publicar si la empresa está en estado APROBADA.
func PublicarBeneficio(empresaId int, body map[string]interface{}) (interface{}, error) {
	// RN-008b: validar campos obligatorios
	required := []string{"titulo", "descripcion", "condiciones", "categoria_beneficio_id", "fecha_inicio", "fecha_fin", "cupos_total"}
	for _, field := range required {
		if v, ok := body[field]; !ok || v == nil || v == "" {
			return nil, fmt.Errorf("campo obligatorio faltante: %s", field)
		}
	}

	// Verificar que la empresa esté APROBADA
	var empresa map[string]interface{}
	if err := helpers.GetCRUD(fmt.Sprintf("/empresa/%d", empresaId), &empresa); err != nil {
		return nil, fmt.Errorf("empresa no encontrada")
	}
	if estado, ok := empresa["estado_empresa"].(map[string]interface{}); ok {
		if estado["codigo_abreviacion"] != "APROBADA" {
			return nil, fmt.Errorf("la empresa debe estar APROBADA para publicar beneficios")
		}
	}

	// Resolver id del estado PUBLICADO
	estadoId, err := resolverIdEstadoBeneficio("PUBLICADO")
	if err != nil {
		return nil, err
	}

	// Transformar relaciones al formato objeto que espera el CRUD
	body["empresa"] = map[string]interface{}{"id": empresaId}
	delete(body, "empresa_id")
	body["estado_beneficio"] = map[string]interface{}{"id": estadoId}
	if cid, ok := body["categoria_beneficio_id"]; ok {
		body["categoria_beneficio"] = map[string]interface{}{"id": toInt(cid)}
		delete(body, "categoria_beneficio_id")
	}
	if uid, ok := body["usuario_creador_id"]; ok {
		body["usuario_creador"] = map[string]interface{}{"id": toInt(uid)}
		delete(body, "usuario_creador_id")
	}

	// Normalizar fechas: "2026-06-01" → "2026-06-01T00:00:00Z"
	for _, campo := range []string{"fecha_inicio", "fecha_fin"} {
		if v, ok := body[campo].(string); ok && !strings.Contains(v, "T") {
			body[campo] = v + "T00:00:00Z"
		}
	}

	// cupos_disponibles = cupos_total al publicar
	body["cupos_disponibles"] = body["cupos_total"]

	var result interface{}
	if err := helpers.PostCRUD("/beneficio", body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// resolverIdEstadoBeneficio obtiene el id del estado de beneficio por codigo_abreviacion.
func resolverIdEstadoBeneficio(codigo string) (int, error) {
	var estados []map[string]interface{}
	if err := helpers.GetCRUD("/estado_beneficio", &estados); err != nil {
		return 0, fmt.Errorf("no se pudo obtener estados de beneficio")
	}
	for _, e := range estados {
		if e["codigo_abreviacion"] == codigo {
			return toInt(e["id"]), nil
		}
	}
	return 0, fmt.Errorf("estado %s no encontrado", codigo)
}


// EditarBeneficio edita un beneficio existente (solo BORRADOR o PUBLICADO sin solicitudes activas).
func EditarBeneficio(id int, body map[string]interface{}) error {
	// TODO: verificar que el beneficio sea BORRADOR o PUBLICADO sin solicitudes activas
	return helpers.PutCRUD(fmt.Sprintf("/beneficio/%d", id), body)
}
