package services

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/udistrital/sga_mid_beneficios_egresados/helpers"
)

// IdTipoDocumentoBeneficiosEgresados es el IdTipoDocumento fijo (provisionado en
// documentos_crud por el gestor documental institucional) para todos los PDFs que
// se suban desde este módulo. Fijo por instrucción del negocio: NO se resuelve por
// ambiente ni por tipo de documento requerido.
const IdTipoDocumentoBeneficiosEgresados = 167

// respuestaGestorDocumental formato de POST document/uploadAnyFormat del gestor
// documental institucional (ver gestor_documental_mid.md en memoria): el uid/hash
// del documento en Nuxeo viaja en res.Enlace.
type respuestaGestorDocumental struct {
	Status string `json:"Status"`
	Res    struct {
		Enlace string `json:"Enlace"`
	} `json:"res"`
}

// esPdfBase64 valida (defensa en profundidad; el cliente ya valida) que el
// contenido decodificado empiece con la cabecera %PDF.
func esPdfBase64(fileBase64 string) bool {
	raw, err := base64.StdEncoding.DecodeString(fileBase64)
	if err != nil || len(raw) < 4 {
		return false
	}
	return strings.HasPrefix(string(raw[:4]), "%PDF")
}

// SubirDocumentoGestor sube un PDF al gestor documental institucional
// (IdTipoDocumento=167) y devuelve el uid/Enlace del documento en Nuxeo.
func SubirDocumentoGestor(token, nombre, descripcion, fileBase64 string, metadatos map[string]interface{}) (string, error) {
	if !esPdfBase64(fileBase64) {
		return "", fmt.Errorf("el archivo debe ser un PDF válido")
	}

	payload := []map[string]interface{}{
		{
			"IdTipoDocumento": IdTipoDocumentoBeneficiosEgresados,
			"nombre":          nombre,
			"metadatos":       metadatos,
			"descripcion":     descripcion,
			"file":            fileBase64,
		},
	}

	var resp respuestaGestorDocumental
	if err := helpers.PostGestorDocumental(token, "document/uploadAnyFormat", payload, &resp); err != nil {
		return "", fmt.Errorf("no se pudo subir el documento al gestor documental: %v", err)
	}
	if resp.Res.Enlace == "" {
		return "", fmt.Errorf("el gestor documental no devolvió el enlace del documento")
	}
	return resp.Res.Enlace, nil
}

// ObtenerDocumentoGestor consulta un documento en el gestor documental por su
// uid/Enlace y devuelve el archivo en base64.
func ObtenerDocumentoGestor(token, enlace string) (string, error) {
	var resp map[string]interface{}
	if err := helpers.GetGestorDocumental(token, fmt.Sprintf("document/%s", enlace), &resp); err != nil {
		return "", fmt.Errorf("no se pudo consultar el documento en el gestor documental: %v", err)
	}
	archivo, _ := resp["file"].(string)
	if archivo == "" {
		return "", fmt.Errorf("el gestor documental no devolvió el contenido del documento")
	}
	return archivo, nil
}

// EliminarDocumentoGestor elimina (borrado lógico en Nuxeo) un documento por su
// uid/Enlace. Best-effort: el caller decide si un fallo aquí es bloqueante.
func EliminarDocumentoGestor(token, enlace string) error {
	if enlace == "" {
		return nil
	}
	if err := helpers.DeleteGestorDocumental(token, fmt.Sprintf("document/%s", enlace)); err != nil {
		return fmt.Errorf("no se pudo eliminar el documento del gestor documental: %v", err)
	}
	return nil
}
