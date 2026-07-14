package services

import (
	"fmt"
	"strings"

	"github.com/udistrital/egresados_service/helpers"
)

// UserRol es la identidad que devuelve autenticacion_mid/token/userRol: la fuente de
// verdad para saber quién se autenticó y si es egresado o empresa (campo Estado).
type UserRol struct {
	Role               []string `json:"role"`
	Documento          string   `json:"documento"`
	DocumentoCompuesto string   `json:"documento_compuesto"`
	Email              string   `json:"email"`
	FamilyName         string   `json:"FamilyName"`
	Codigo             string   `json:"Codigo"` // código estudiantil (solo egresados)
	Estado             string   `json:"Estado"` // "E" = egresado; distinto de E = empresa
}

// EsEgresado indica si la persona autenticada es egresado. Regla defensiva: egresado
// solo si Estado es exactamente "E"; cualquier otro valor (incluido vacío) es empresa.
func (u *UserRol) EsEgresado() bool {
	return strings.EqualFold(strings.TrimSpace(u.Estado), "E")
}

// EsEmpresa es el complemento de EsEgresado.
func (u *UserRol) EsEmpresa() bool { return !u.EsEgresado() }

// UserInfo es la identidad que devuelve el endpoint OIDC userinfo a partir del token:
// la fuente confiable del email autenticado para el JIT (deriva del token, no del body).
type UserInfo struct {
	Sub                string `json:"sub"`
	Email              string `json:"email"`
	Documento          string `json:"documento"`
	DocumentoCompuesto string `json:"documento_compuesto"`
}

// GetUserInfoDeToken resuelve la identidad del dueño del token vía OIDC userinfo.
func GetUserInfoDeToken(token string) (*UserInfo, error) {
	var u UserInfo
	if err := helpers.GetUserInfo(token, &u); err != nil {
		return nil, fmt.Errorf("no se pudo identificar al usuario del token: %v", err)
	}
	if strings.TrimSpace(u.Email) == "" {
		return nil, fmt.Errorf("el token no expone email (¿falta scope openid/email?)")
	}
	return &u, nil
}

// GetUserRol consulta la identidad del usuario a partir de su email.
func GetUserRol(token, email string) (*UserRol, error) {
	if strings.TrimSpace(email) == "" {
		return nil, fmt.Errorf("email es requerido para consultar userRol")
	}
	var u UserRol
	payload := map[string]interface{}{"user": email}
	if err := helpers.PostAuth(token, "/token/userRol", payload, &u); err != nil {
		// El 400 "Usuario no registrado" cae aquí: el usuario aún no existe en WSO2.
		return nil, fmt.Errorf("no se pudo resolver la identidad de %s: %v", email, err)
	}
	return &u, nil
}
