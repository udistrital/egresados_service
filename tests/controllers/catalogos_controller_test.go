package controllers_test

import (
	"net/http"
	"testing"
)

// Familia B: pruebas de integración contra el servidor real (bee run / docker run).
// baseURL asume el MID corriendo en el puerto por defecto de dev (8081).
// Estos catálogos no requieren token cuando EGRESADOS_SERVICE_VALIDAR_JWT=false
// (o ParametrosLocal=true resuelve el catálogo local, sin llamar a parámetros).
const baseURL = "http://localhost:8081/v1"

func TestGetCategoriasBeneficio(t *testing.T) {
	endpoint := baseURL + "/categorias-beneficio"

	if response, err := http.Get(endpoint); err == nil {
		if response.StatusCode != 200 {
			t.Error("Error en GetCategoriasBeneficio, se esperaba 200 y se obtuvo", response.StatusCode)
			t.Fail()
		} else {
			t.Log("GetCategoriasBeneficio finalizado correctamente (OK)")
		}
	} else {
		t.Error("Error GetCategoriasBeneficio:", err.Error())
		t.Fail()
	}
}

func TestGetSectoresEconomicos(t *testing.T) {
	endpoint := baseURL + "/sectores-economicos"

	if response, err := http.Get(endpoint); err == nil {
		if response.StatusCode != 200 {
			t.Error("Error en GetSectoresEconomicos, se esperaba 200 y se obtuvo", response.StatusCode)
			t.Fail()
		} else {
			t.Log("GetSectoresEconomicos finalizado correctamente (OK)")
		}
	} else {
		t.Error("Error GetSectoresEconomicos:", err.Error())
		t.Fail()
	}
}
