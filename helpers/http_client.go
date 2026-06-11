package helpers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

var (
	crudURL       = getEnv("BENEFICIOS_EGRESADOS_MID_CRUD_URL", "http://localhost:8080/v1")
	authURL       = getEnv("BENEFICIOS_EGRESADOS_MID_AUTENTICACION_URL", "https://autenticacion.portaloas.udistrital.edu.co/apioas/autenticacion_mid/v1")
	parametrosURL = getEnv("BENEFICIOS_EGRESADOS_MID_PARAMETROS_URL", "https://autenticacion.portaloas.udistrital.edu.co/apioas/parametros/v1")
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// GetCRUD realiza un GET al CRUD y decodifica la respuesta en dest.
func GetCRUD(path string, dest interface{}) error {
	resp, err := http.Get(fmt.Sprintf("%s%s", crudURL, path))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("CRUD respondió %d: %s", resp.StatusCode, string(body))
	}
	// Los GetAll del CRUD responden [{}] cuando la lista está vacía (idioma de los
	// *_crud del SGA); normalizar para no inyectar un elemento zero-value en dest.
	if strings.TrimSpace(string(body)) == "[{}]" {
		body = []byte("[]")
	}
	return json.Unmarshal(body, dest)
}

// PostCRUD realiza un POST al CRUD con el payload dado y decodifica la respuesta en dest.
func PostCRUD(path string, payload interface{}, dest interface{}) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := http.Post(
		fmt.Sprintf("%s%s", crudURL, path),
		"application/json",
		strings.NewReader(string(b)),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, dest)
}

// PutCRUD realiza un PUT al CRUD con el payload dado.
func PutCRUD(path string, payload interface{}) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPut,
		fmt.Sprintf("%s%s", crudURL, path),
		strings.NewReader(string(b)),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("CRUD respondió %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// AuthURL devuelve la URL base del servicio de autenticación.
func AuthURL() string { return authURL }

// GetParametros realiza un GET al servicio institucional de parámetros y
// decodifica la respuesta estándar { Success, Status, Message, Data } en dest.
func GetParametros(path string, dest interface{}) error {
	resp, err := http.Get(fmt.Sprintf("%s%s", parametrosURL, path))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, dest)
}
