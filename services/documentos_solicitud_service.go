package services

import (
	"fmt"
	"time"

	"github.com/udistrital/egresados_service/helpers"
)

// relId extrae el id de una relación anidada del CRUD (p. ej. s["documento_requerido"] = {"id": N, ...}).
func relId(m map[string]interface{}, key string) int {
	rel, ok := m[key].(map[string]interface{})
	if !ok {
		return 0
	}
	return toInt(firstOf(rel, "id", "Id"))
}

// GetDocumentosDeSolicitud arma la vista de documentos requeridos vs. subidos de
// una solicitud: por cada documento requerido indica si ya se subió y con qué datos.
func GetDocumentosDeSolicitud(token string, solicitudId int) ([]map[string]interface{}, error) {
	beneficioId, err := getBeneficioIdDeSolicitud(token, solicitudId)
	if err != nil {
		return nil, err
	}
	requeridos, err := GetDocumentosRequeridos(token, beneficioId)
	if err != nil {
		return nil, err
	}

	subidos, err := getDocumentosSubidos(token, solicitudId)
	if err != nil {
		return nil, err
	}
	porRequerido := make(map[int]map[string]interface{}, len(subidos))
	for _, s := range subidos {
		porRequerido[relId(s, "documento_requerido")] = s
	}

	items := make([]map[string]interface{}, 0, len(requeridos))
	for _, r := range requeridos {
		reqId := toInt(firstOf(r, "id", "Id"))
		item := map[string]interface{}{
			"documento_requerido_id": reqId,
			"nombre":                 firstOf(r, "nombre", "Nombre"),
			"descripcion":            firstOf(r, "descripcion", "Descripcion"),
			"subido":                 false,
		}
		if s, ok := porRequerido[reqId]; ok {
			item["subido"] = true
			item["documento_solicitud_id"] = toInt(firstOf(s, "id", "Id"))
			item["nombre_archivo"] = firstOf(s, "nombre_archivo", "NombreArchivo")
			item["comentario_empresa"] = firstOf(s, "comentario_empresa", "ComentarioEmpresa")
			item["fecha_comentario"] = firstOf(s, "fecha_comentario", "FechaComentario")
		}
		items = append(items, item)
	}
	return items, nil
}

// getDocumentosSubidos lista los documento_solicitud (activos) de una solicitud.
func getDocumentosSubidos(token string, solicitudId int) ([]map[string]interface{}, error) {
	var subidos []map[string]interface{}
	q := fmt.Sprintf("/documento-solicitud/solicitud/%d", solicitudId)
	if err := helpers.GetCRUD(token, q, &subidos); err != nil {
		return nil, err
	}
	return subidos, nil
}

// puedeGestionarDocumentos permite subir/reemplazar/eliminar documentos solo
// mientras la solicitud sigue en curso (RN-005).
func puedeGestionarDocumentos(token string, solicitudId int) error {
	estado, _, err := getEstadoActual(token, solicitudId)
	if err != nil {
		return err
	}
	if !esEstadoNoTerminal(estado) {
		return fmt.Errorf("no se pueden gestionar documentos: la solicitud ya no está en curso (estado actual: %s)", estado)
	}
	return nil
}

// SubirDocumentoSolicitud sube (o reemplaza) el PDF que cumple un documento
// requerido de una solicitud. body: { documento_requerido_id, nombre_archivo, file }.
func SubirDocumentoSolicitud(token string, solicitudId int, body map[string]interface{}) (interface{}, error) {
	documentoRequeridoId := toInt(body["documento_requerido_id"])
	nombreArchivo, _ := body["nombre_archivo"].(string)
	fileBase64, _ := body["file"].(string)
	if documentoRequeridoId == 0 || nombreArchivo == "" || fileBase64 == "" {
		return nil, fmt.Errorf("documento_requerido_id, nombre_archivo y file son requeridos")
	}

	if err := puedeGestionarDocumentos(token, solicitudId); err != nil {
		return nil, err
	}

	subidos, err := getDocumentosSubidos(token, solicitudId)
	if err != nil {
		return nil, err
	}
	var existente map[string]interface{}
	for _, s := range subidos {
		if relId(s, "documento_requerido") == documentoRequeridoId {
			existente = s
			break
		}
	}

	enlace, err := SubirDocumentoGestor(token, nombreArchivo, "Documento de solicitud de beneficio para egresados", fileBase64)
	if err != nil {
		return nil, err
	}

	if existente != nil {
		// Reemplazo: borrar el archivo anterior en el gestor documental (best-effort).
		enlaceAnterior := asString(firstOf(existente, "enlace_gestor_documental", "EnlaceGestorDocumental"))
		_ = EliminarDocumentoGestor(token, enlaceAnterior)

		// El PUT del CRUD reemplaza la fila entera: se parte del row completo con las
		// relaciones normalizadas a {id}, nunca de un payload parcial.
		docId := toInt(firstOf(existente, "id", "Id"))
		doc := documentoParaPut(existente)
		doc["nombre_archivo"] = nombreArchivo
		doc["enlace_gestor_documental"] = enlace
		if err := helpers.PutCRUD(token, fmt.Sprintf("/documento-solicitud/%d", docId), doc); err != nil {
			return nil, err
		}
		return map[string]interface{}{"id": docId}, nil
	}

	payload := map[string]interface{}{
		"solicitud_beneficio":      map[string]interface{}{"id": solicitudId},
		"documento_requerido":      map[string]interface{}{"id": documentoRequeridoId},
		"nombre_archivo":           nombreArchivo,
		"enlace_gestor_documental": enlace,
	}
	var result interface{}
	if err := helpers.PostCRUD(token, "/documento-solicitud", payload, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// EliminarDocumentoSolicitud quita un documento subido por el egresado (borrado
// lógico) y su archivo en el gestor documental.
func EliminarDocumentoSolicitud(token string, solicitudId, documentoSolicitudId int) error {
	if err := puedeGestionarDocumentos(token, solicitudId); err != nil {
		return err
	}
	var doc map[string]interface{}
	if err := helpers.GetCRUD(token, fmt.Sprintf("/documento-solicitud/%d", documentoSolicitudId), &doc); err != nil {
		return fmt.Errorf("documento no encontrado: %v", err)
	}
	enlace := asString(firstOf(doc, "enlace_gestor_documental", "EnlaceGestorDocumental"))
	if err := EliminarDocumentoGestor(token, enlace); err != nil {
		return err
	}
	return helpers.DeleteCRUD(token, fmt.Sprintf("/documento-solicitud/%d", documentoSolicitudId))
}

// ComentarDocumento registra (o reemplaza) la observación de la empresa sobre un
// documento subido. Se lee el documento completo y se sobreescriben solo los campos
// del comentario: un payload parcial dejaría la FK solicitud_beneficio en NULL.
func ComentarDocumento(token string, documentoSolicitudId int, comentario string) error {
	if comentario == "" {
		return fmt.Errorf("el comentario no puede estar vacío")
	}
	var existente map[string]interface{}
	if err := helpers.GetCRUD(token, fmt.Sprintf("/documento-solicitud/%d", documentoSolicitudId), &existente); err != nil {
		return fmt.Errorf("documento no encontrado: %v", err)
	}
	doc := documentoParaPut(existente)
	doc["comentario_empresa"] = comentario
	doc["fecha_comentario"] = time.Now().Format(time.RFC3339)
	return helpers.PutCRUD(token, fmt.Sprintf("/documento-solicitud/%d", documentoSolicitudId), doc)
}

// documentoParaPut prepara un documento_solicitud leído del CRUD para un PUT de
// fila completa: normaliza las relaciones anidadas a la forma {id} que espera el
// unmarshal del CRUD.
func documentoParaPut(doc map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(doc))
	for k, v := range doc {
		out[k] = v
	}
	for _, rel := range []string{"solicitud_beneficio", "documento_requerido"} {
		if id := relId(out, rel); id > 0 {
			out[rel] = map[string]interface{}{"id": id}
		}
	}
	return out
}

// GetArchivoDocumento devuelve el nombre y el contenido en base64 de un documento
// subido.
func GetArchivoDocumento(token string, documentoSolicitudId int) (map[string]interface{}, error) {
	var doc map[string]interface{}
	if err := helpers.GetCRUD(token, fmt.Sprintf("/documento-solicitud/%d", documentoSolicitudId), &doc); err != nil {
		return nil, fmt.Errorf("documento no encontrado: %v", err)
	}
	enlace := asString(firstOf(doc, "enlace_gestor_documental", "EnlaceGestorDocumental"))
	archivo, err := ObtenerDocumentoGestor(token, enlace)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"nombre_archivo": firstOf(doc, "nombre_archivo", "NombreArchivo"),
		"file":           archivo,
	}, nil
}

// GetComprobanteSolicitud devuelve el comprobante opcional que la empresa adjuntó
// al aprobar; tiene_comprobante=false cuando no adjuntó nada (no es error).
func GetComprobanteSolicitud(token string, solicitudId int) (map[string]interface{}, error) {
	nombreArchivo, enlace, err := getComprobanteDeSolicitud(token, solicitudId)
	if err != nil {
		return nil, err
	}
	if enlace == "" {
		return map[string]interface{}{"tiene_comprobante": false}, nil
	}
	archivo, err := ObtenerDocumentoGestor(token, enlace)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"tiene_comprobante": true,
		"nombre_archivo":    nombreArchivo,
		"file":              archivo,
	}, nil
}
