package controllers_test

import (
	"net/http"
	"testing"
)

// GetCatalogo: catálogo paginado de beneficios (solo PUBLICADO, RN-008). No requiere
// token; el filtrado de negocio no impide responder 200 con catálogo vacío.
func TestGetCatalogoBeneficios(t *testing.T) {
	endpoint := baseURL + "/beneficios"

	if response, err := http.Get(endpoint); err == nil {
		if response.StatusCode != 200 {
			t.Error("Error en GetCatalogoBeneficios, se esperaba 200 y se obtuvo", response.StatusCode)
			t.Fail()
		} else {
			t.Log("GetCatalogoBeneficios finalizado correctamente (OK)")
		}
	} else {
		t.Error("Error GetCatalogoBeneficios:", err.Error())
		t.Fail()
	}
}
