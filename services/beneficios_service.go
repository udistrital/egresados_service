package services

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/udistrital/sga_mid_beneficios_egresados/helpers"
)

// GetCatalogoBeneficios devuelve el catálogo paginado con filtros (RN-008, RN-FILTROS).
// Solo beneficios con estado=PUBLICADO, fecha_fin >= hoy, cupos_disponibles > 0.
func GetCatalogoBeneficios(page, limit, categoriaId, empresaId int, q string) (interface{}, error) {
	// El estado ya no es FK local: se filtra por el id del parámetro PUBLICADO (C-1)
	publicadoId, err := ResolverParametroId(TipoParamEstadoBeneficio, "PUBLICADO")
	if err != nil {
		return nil, err
	}

	// Construir query string para el CRUD
	offset := (page - 1) * limit
	filtros := fmt.Sprintf("EstadoBeneficioId:%d,Activo:true", publicadoId)
	if categoriaId > 0 {
		filtros += fmt.Sprintf(",CategoriaBeneficioId:%d", categoriaId)
	}
	if empresaId > 0 {
		filtros += fmt.Sprintf(",Empresa.Id:%d", empresaId)
	}
	if q != "" {
		filtros += ",Titulo__icontains:" + url.QueryEscape(q)
	}
	// fecha_fin >= hoy y cupos_disponibles > 0 (RN-008): operadores nativos del ORM
	filtros += ",FechaFin__gte:" + time.Now().Format("2006-01-02")
	filtros += ",CuposDisponibles__gt:0"
	query := fmt.Sprintf("/beneficio?query=%s&limit=%d&offset=%d", filtros, limit, offset)

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

	// Verificar que la empresa esté APROBADA (estado_empresa_id → parámetro)
	var empresa map[string]interface{}
	if err := helpers.GetCRUD(fmt.Sprintf("/empresa/%d", empresaId), &empresa); err != nil {
		return nil, fmt.Errorf("empresa no encontrada")
	}
	aprobadaId, err := ResolverParametroId(TipoParamEstadoEmpresa, "APROBADA")
	if err != nil {
		return nil, err
	}
	if toInt(empresa["estado_empresa_id"]) != aprobadaId {
		return nil, fmt.Errorf("la empresa debe estar APROBADA para publicar beneficios")
	}

	// Resolver id del estado PUBLICADO (servicio de parámetros, C-1)
	estadoId, err := ResolverParametroId(TipoParamEstadoBeneficio, "PUBLICADO")
	if err != nil {
		return nil, err
	}

	// empresa y usuario_creador siguen siendo relaciones del CRUD (formato objeto);
	// categoría y estado son ids de parámetro planos
	body["empresa"] = map[string]interface{}{"id": empresaId}
	delete(body, "empresa_id")
	body["estado_beneficio_id"] = estadoId
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

// EditarBeneficio edita un beneficio existente (solo BORRADOR o PUBLICADO sin solicitudes activas).
func EditarBeneficio(id int, body map[string]interface{}) error {
	// TODO: verificar que el beneficio sea BORRADOR o PUBLICADO sin solicitudes activas
	return helpers.PutCRUD(fmt.Sprintf("/beneficio/%d", id), body)
}
