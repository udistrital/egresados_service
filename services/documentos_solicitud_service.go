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

// GetDocumentosDeSolicitud arma la vista combinada de documentos requeridos vs.
// subidos de una solicitud: por cada documento_requerido_beneficio del beneficio
// de la solicitud, indica si ya se subió (y con qué datos). Sirve igual para el
// egresado (qué le falta subir) y para la empresa (qué subió, para revisar/comentar).
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

// puedeGestionarDocumentos permite subir/reemplazar/eliminar documentos mientras
// la solicitud sigue en curso — mismo criterio de estado que CancelarSolicitud (RN-005).
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

// SubirDocumentoSolicitud sube (o reemplaza, si ya había uno) el PDF que cumple un
// documento requerido de una solicitud. IdTipoDocumento=167 (fijo) vía
// SubirDocumentoGestor. body: { documento_requerido_id, nombre_archivo, file (base64) }.
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
		// Reemplazo: se intenta borrar el archivo anterior en el gestor documental
		// (best-effort, igual criterio que devolverCupo — no bloquea el reemplazo si falla).
		enlaceAnterior := asString(firstOf(existente, "enlace_gestor_documental", "EnlaceGestorDocumental"))
		_ = EliminarDocumentoGestor(token, enlaceAnterior)

		docId := toInt(firstOf(existente, "id", "Id"))
		payload := map[string]interface{}{
			"nombre_archivo":           nombreArchivo,
			"enlace_gestor_documental": enlace,
		}
		if err := helpers.PutCRUD(token, fmt.Sprintf("/documento-solicitud/%d", docId), payload); err != nil {
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
// documento subido por el egresado. Campo único: no lleva historial de comentarios.
func ComentarDocumento(token string, documentoSolicitudId int, comentario string) error {
	if comentario == "" {
		return fmt.Errorf("el comentario no puede estar vacío")
	}
	payload := map[string]interface{}{
		"comentario_empresa": comentario,
		"fecha_comentario":   time.Now().Format(time.RFC3339),
	}
	return helpers.PutCRUD(token, fmt.Sprintf("/documento-solicitud/%d", documentoSolicitudId), payload)
}

// GetArchivoDocumento devuelve el nombre y el contenido en base64 de un documento
// subido, para que el cliente lo abra/descargue sin llamar directo al gestor documental.
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

// GetComprobanteSolicitud devuelve el comprobante (opcional) que la empresa adjuntó
// al aprobar la solicitud. tiene_comprobante=false (sin file/nombre_archivo) es el
// caso normal cuando la empresa no adjuntó nada — no es un error.
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
