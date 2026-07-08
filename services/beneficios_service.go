package services

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/udistrital/sga_mid_beneficios_egresados/helpers"
)

// GetCatalogoBeneficios devuelve el catálogo paginado con filtros (RN-008, RN-FILTROS).
// Solo beneficios con estado=PUBLICADO y fecha_fin >= hoy. Los AGOTADOS
// (cupos_disponibles = 0) SÍ se listan: la UI los muestra deshabilitados ("Sin
// cupos") y ofrece el toggle "sólo con cupos"; ocultarlos del todo daba la
// impresión de que el beneficio nunca existió.
func GetCatalogoBeneficios(token string, page, limit, categoriaId, empresaId int, q string) (interface{}, error) {
	// El estado ya no es FK local: se filtra por el id del parámetro PUBLICADO (C-1)
	publicadoId, err := ResolverParametroId(token, TipoParamEstadoBeneficio, "PUBLICADO")
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
	// fecha_fin >= hoy (RN-008): operador nativo del ORM
	filtros += ",FechaFin__gte:" + time.Now().Format("2006-01-02")
	query := fmt.Sprintf("/beneficio?query=%s&limit=%d&offset=%d", filtros, limit, offset)

	var result interface{}
	if err := helpers.GetCRUD(token, query, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetBeneficioDetalle devuelve el detalle de un beneficio por id, con el total
// histórico de solicitudes ("N egresados ya lo solicitaron", social proof del detalle)
// y los documentos que la empresa exige para postularse.
func GetBeneficioDetalle(token string, id int) (interface{}, error) {
	var result map[string]interface{}
	if err := helpers.GetCRUD(token, fmt.Sprintf("/beneficio/%d", id), &result); err != nil {
		return nil, err
	}
	// Best-effort: si el conteo falla, el detalle sale sin él.
	var solicitudes []map[string]interface{}
	q := fmt.Sprintf("/solicitud_beneficio?query=Beneficio.Id:%d,Activo:true&fields=Id&limit=0", id)
	if err := helpers.GetCRUD(token, q, &solicitudes); err == nil {
		result["total_solicitudes"] = len(solicitudes)
	}
	// Best-effort: si el gestor de documentos requeridos falla, el detalle sale sin ellos.
	if documentos, err := GetDocumentosRequeridos(token, id); err == nil {
		result["documentos_requeridos"] = documentos
	}
	return result, nil
}

// GetDocumentosRequeridos lista los documentos que la empresa exige para postularse
// a un beneficio (definidos al publicar, RF-005/documentos). Usado tanto por el
// detalle del catálogo (egresado, antes de postularse) como por el merge de
// GetDocumentosDeSolicitud (egresado/empresa, después de postularse).
func GetDocumentosRequeridos(token string, beneficioId int) ([]map[string]interface{}, error) {
	var documentos []map[string]interface{}
	q := fmt.Sprintf("/documento_requerido_beneficio?query=Beneficio.Id:%d,Activo:true&limit=0", beneficioId)
	if err := helpers.GetCRUD(token, q, &documentos); err != nil {
		return nil, err
	}
	return documentos, nil
}

// GetBeneficiosDeEmpresa lista TODOS los beneficios de una empresa — es la vista
// de gestión del DUEÑO, así que incluye borradores, agotados, vencidos y retirados
// (a diferencia del catálogo público). Cada ítem lleva el código de su estado y
// las métricas de solicitudes: recibidas (histórico) y pendientes de acción de la
// empresa (estado vigente PENDIENTE o EN_REVISION).
// N+1 de getEstadoActual (C-4b), mismo caveat que RN-007/010.
func GetBeneficiosDeEmpresa(token string, empresaId int) (interface{}, error) {
	var beneficios []map[string]interface{}
	q := fmt.Sprintf("/beneficio?query=Empresa.Id:%d,Activo:true&limit=0", empresaId)
	if err := helpers.GetCRUD(token, q, &beneficios); err != nil {
		return nil, err
	}
	for _, b := range beneficios {
		if codigo, err := ResolverParametroCodigo(token, TipoParamEstadoBeneficio, toInt(b["estado_beneficio_id"])); err == nil {
			b["estado_beneficio"] = codigo
		}
		var sols []map[string]interface{}
		qs := fmt.Sprintf("/solicitud_beneficio?query=Beneficio.Id:%d,Activo:true&fields=Id&limit=0", toInt(b["id"]))
		if err := helpers.GetCRUD(token, qs, &sols); err != nil {
			continue // best effort: la card sale sin métricas
		}
		b["total_solicitudes"] = len(sols)
		pendientes := 0
		for _, s := range sols {
			codigo, _, err := getEstadoActual(token, toInt(firstOf(s, "id", "Id")))
			if err == nil && (codigo == estadoPendiente || codigo == estadoEnRevision) {
				pendientes++
			}
		}
		b["solicitudes_pendientes"] = pendientes
	}
	return beneficios, nil
}

// PublicarBeneficio valida RN-008b y crea el beneficio en el CRUD.
// Solo permite publicar si la empresa está ACTIVA (no suspendida).
func PublicarBeneficio(token string, empresaId int, body map[string]interface{}) (interface{}, error) {
	// RN-008b: validar campos obligatorios
	required := []string{"titulo", "descripcion", "condiciones", "categoria_beneficio_id", "fecha_inicio", "fecha_fin", "cupos_total"}
	for _, field := range required {
		if v, ok := body[field]; !ok || v == nil || v == "" {
			return nil, fmt.Errorf("campo obligatorio faltante: %s", field)
		}
	}

	// Verificar que la empresa esté ACTIVA (estado_empresa_id → parámetro)
	var empresa map[string]interface{}
	if err := helpers.GetCRUD(token, fmt.Sprintf("/empresa/%d", empresaId), &empresa); err != nil {
		return nil, fmt.Errorf("empresa no encontrada")
	}
	activaId, err := ResolverParametroId(token, TipoParamEstadoEmpresa, "ACTIVA")
	if err != nil {
		return nil, err
	}
	if toInt(empresa["estado_empresa_id"]) != activaId {
		return nil, fmt.Errorf("la empresa debe estar ACTIVA (no suspendida) para publicar beneficios")
	}

	// Resolver id del estado PUBLICADO (servicio de parámetros, C-1)
	estadoId, err := ResolverParametroId(token, TipoParamEstadoBeneficio, "PUBLICADO")
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

	// El beneficio nace PUBLICADO → fecha_publicacion es ahora (el cliente no la manda)
	body["fecha_publicacion"] = time.Now().UTC().Format(time.RFC3339)

	// cupos_disponibles = cupos_total al publicar
	body["cupos_disponibles"] = body["cupos_total"]

	// documentos_requeridos (opcional): se crean aparte, no son columnas de beneficio.
	documentosRequeridos, _ := body["documentos_requeridos"].([]interface{})
	delete(body, "documentos_requeridos")

	var result map[string]interface{}
	if err := helpers.PostCRUD(token, "/beneficio", body, &result); err != nil {
		return nil, err
	}

	if len(documentosRequeridos) > 0 {
		beneficioId := toInt(result["id"])
		for _, d := range documentosRequeridos {
			doc, ok := d.(map[string]interface{})
			if !ok {
				continue
			}
			nombre, _ := doc["nombre"].(string)
			if strings.TrimSpace(nombre) == "" {
				continue // fila vacía del formulario: se ignora, no es un error
			}
			descripcion, _ := doc["descripcion"].(string)
			payload := map[string]interface{}{
				"beneficio":   map[string]interface{}{"id": beneficioId},
				"nombre":      nombre,
				"descripcion": descripcion,
			}
			var docResult interface{}
			if err := helpers.PostCRUD(token, "/documento_requerido_beneficio", payload, &docResult); err != nil {
				// El beneficio YA se creó; un documento requerido que falla no debe
				// revertirlo (no hay cupo ni radicado en juego aquí). Se reporta el
				// error para que la empresa sepa que ese documento no quedó registrado.
				return nil, fmt.Errorf("beneficio creado pero no se pudo registrar el documento requerido %q: %v", nombre, err)
			}
		}
	}

	return result, nil
}

// EditarBeneficio edita el CONTENIDO de un beneficio (RF-005). Regla: solo BORRADOR,
// o PUBLICADO sin solicitudes en curso — editar condiciones con postulaciones activas
// sería cambiar las reglas a mitad de camino. El estado NO se cambia por aquí
// (retirar tiene endpoint propio) y empresa/usuario_creador no cambian de dueño.
func EditarBeneficio(token string, id int, body map[string]interface{}) error {
	ben, err := getBeneficioBase(token, id)
	if err != nil {
		return err
	}
	estado, err := ResolverParametroCodigo(token, TipoParamEstadoBeneficio, toInt(ben["estado_beneficio_id"]))
	if err != nil {
		return err
	}
	switch estado {
	case "BORRADOR":
		// editable siempre
	case "PUBLICADO":
		activas, err := beneficioTieneSolicitudesActivas(token, id)
		if err != nil {
			return err
		}
		if activas {
			return fmt.Errorf("el beneficio tiene solicitudes en curso; respóndelas antes de editarlo o retíralo")
		}
	default:
		return fmt.Errorf("solo se puede editar un beneficio en BORRADOR o PUBLICADO (estado actual: %s)", estado)
	}

	// Whitelist de campos editables sobre el objeto completo (el PUT del CRUD
	// escribe todas las columnas; ver getBeneficioBase).
	cuposAntes := toInt(ben["cupos_total"])
	for _, campo := range []string{"titulo", "descripcion", "condiciones", "categoria_beneficio_id", "fecha_inicio", "fecha_fin", "cupos_total", "imagen_url"} {
		if v, ok := body[campo]; ok && v != nil && v != "" {
			ben[campo] = v
		}
	}
	// Normalizar fechas: "2026-06-01" → "2026-06-01T00:00:00Z" (mismo idioma que publicar)
	for _, campo := range []string{"fecha_inicio", "fecha_fin"} {
		if v, ok := ben[campo].(string); ok && !strings.Contains(v, "T") {
			ben[campo] = v + "T00:00:00Z"
		}
	}
	// Si cambia el total de cupos, los disponibles se mueven con el mismo delta
	// (lo ya consumido se respeta); nunca por debajo de 0.
	if delta := toInt(ben["cupos_total"]) - cuposAntes; delta != 0 {
		disponibles := toInt(ben["cupos_disponibles"]) + delta
		if disponibles < 0 {
			disponibles = 0
		}
		ben["cupos_disponibles"] = disponibles
	}
	return helpers.PutCRUD(token, fmt.Sprintf("/beneficio/%d", id), ben)
}

// RetirarBeneficio pasa el beneficio a RETIRADO (el "cerrar" de la empresa): sale del
// catálogo y no acepta nuevas solicitudes. Se permite desde cualquier estado no
// retirado. Las solicitudes EN CURSO no se tocan (la empresa las sigue respondiendo
// desde la bandeja) y los cupos no se devuelven: el beneficio deja de ser solicitable,
// así que no hay carrera por ellos.
func RetirarBeneficio(token string, id int) error {
	ben, err := getBeneficioBase(token, id)
	if err != nil {
		return err
	}
	retiradoId, err := ResolverParametroId(token, TipoParamEstadoBeneficio, "RETIRADO")
	if err != nil {
		return err
	}
	if toInt(ben["estado_beneficio_id"]) == retiradoId {
		return fmt.Errorf("el beneficio ya está retirado")
	}
	ben["estado_beneficio_id"] = retiradoId
	return helpers.PutCRUD(token, fmt.Sprintf("/beneficio/%d", id), ben)
}

// getBeneficioBase obtiene el beneficio del CRUD listo para un PUT completo:
// las relaciones se normalizan a {id} (el PUT del CRUD escribe todas las columnas).
func getBeneficioBase(token string, id int) (map[string]interface{}, error) {
	var ben map[string]interface{}
	if err := helpers.GetCRUD(token, fmt.Sprintf("/beneficio/%d", id), &ben); err != nil {
		return nil, fmt.Errorf("beneficio %d no encontrado", id)
	}
	for _, rel := range []string{"empresa", "usuario_creador"} {
		if m, ok := ben[rel].(map[string]interface{}); ok {
			ben[rel] = map[string]interface{}{"id": toInt(firstOf(m, "id", "Id"))}
		}
	}
	return ben, nil
}

// beneficioTieneSolicitudesActivas indica si el beneficio tiene solicitudes con estado
// vigente NO terminal (PENDIENTE/EN_REVISION/REQUIERE_INFO). N+1 de getEstadoActual,
// mismo caveat que RN-007/010.
func beneficioTieneSolicitudesActivas(token string, beneficioId int) (bool, error) {
	var solicitudes []map[string]interface{}
	q := fmt.Sprintf("/solicitud_beneficio?query=Beneficio.Id:%d,Activo:true&fields=Id&limit=0", beneficioId)
	if err := helpers.GetCRUD(token, q, &solicitudes); err != nil {
		return false, err
	}
	for _, s := range solicitudes {
		codigo, _, err := getEstadoActual(token, toInt(firstOf(s, "id", "Id")))
		if err != nil {
			continue // sin historial legible: no bloquea la edición
		}
		if codigo == estadoPendiente || codigo == estadoEnRevision || codigo == estadoRequiereInfo {
			return true, nil
		}
	}
	return false, nil
}
