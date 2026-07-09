// Package middleware valida el token ENTRANTE del MID (la dirección opuesta a la
// propagación del Bearer de helpers/http_client.go).
//
// CONTEXTO (verificado 2026-07-07): utils_oas NO trae validación de JWT —
// security.SetSecurityHeaders() solo pone headers de respuesta (CSP/HSTS/etc.) y los
// servicios del SGA confían en que el gateway WSO2 valida el Bearer antes de enrutar.
// Este MID también se consume directo (localhost / micro-frontend), así que valida por
// su cuenta:
//   - Token JWT (3 segmentos): firma RS256 contra el JWKS de WSO2 + exp/nbf. Sin
//     llamadas remotas por request (el JWKS se cachea).
//   - Token opaco: se valida contra el endpoint OIDC userinfo (caché 5 min por token).
//
// Se desactiva con EGRESADOS_SERVICE_VALIDAR_JWT=false (ValidarJWT en conf/app.conf;
// solo para desarrollo sin conectividad; NUNCA en producción).
package middleware

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/beego/beego/v2/server/web"
	beectx "github.com/beego/beego/v2/server/web/context"
	"github.com/udistrital/egresados_service/helpers"
)

var (
	validarActivo = web.AppConfig.DefaultString("ValidarJWT", "true") != "false"
	// Sin default quemado: si EGRESADOS_SERVICE_JWKS_URL no está seteada, la
	// consulta al JWKS falla explícito en vez de pegarle en silencio al WSO2 real.
	jwksURL = web.AppConfig.DefaultString("Wso2JwksService", "")
)

// ValidarJWTEntrante es el filtro BeforeRouter de todas las rutas /v1/*. Si el token
// falta o es inválido responde 401 con el envelope OATI y corta el request.
func ValidarJWTEntrante(ctx *beectx.Context) {
	if !validarActivo {
		return
	}
	if ctx.Input.Method() == http.MethodOptions {
		return // preflight CORS: lo responde el filtro de CORS, no lleva Authorization
	}
	auth := ctx.Input.Header("Authorization")
	crudo := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	if !strings.HasPrefix(auth, "Bearer ") || crudo == "" {
		responder401(ctx, "se requiere un token Bearer en el header Authorization")
		return
	}

	var err error
	if strings.Count(crudo, ".") == 2 {
		err = validarComoJWT(crudo)
	} else {
		err = validarOpaco(auth)
	}
	if err != nil {
		responder401(ctx, fmt.Sprintf("token inválido: %v", err))
	}
}

func responder401(ctx *beectx.Context, msg string) {
	ctx.Output.SetStatus(401)
	// Al escribir la respuesta aquí, Beego corta la cadena (ReturnOnOutput por defecto).
	_ = ctx.Output.JSON(helpers.APIResponse{Status: "401", Success: false, Message: msg}, false, false)
}

// ── Validación local de JWT (RS256 + JWKS de WSO2) ────────────────────────────

func validarComoJWT(token string) error {
	partes := strings.Split(token, ".")

	headerJSON, err := base64.RawURLEncoding.DecodeString(partes[0])
	if err != nil {
		return fmt.Errorf("header no es base64url válido")
	}
	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return fmt.Errorf("header no es JSON válido")
	}
	// Solo RS256 (lo que publica el JWKS de WSO2). Rechazar cualquier otro alg cierra
	// los ataques de confusión de algoritmo (p. ej. none o HS256 firmado con la clave pública).
	if header.Alg != "RS256" {
		return fmt.Errorf("alg %q no soportado (se exige RS256)", header.Alg)
	}

	claimsJSON, err := base64.RawURLEncoding.DecodeString(partes[1])
	if err != nil {
		return fmt.Errorf("payload no es base64url válido")
	}
	var claims struct {
		Exp int64 `json:"exp"`
		Nbf int64 `json:"nbf"`
	}
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return fmt.Errorf("payload no es JSON válido")
	}
	ahora := time.Now().Unix()
	if claims.Exp > 0 && ahora > claims.Exp {
		return fmt.Errorf("el token expiró (exp)")
	}
	if claims.Nbf > 0 && ahora < claims.Nbf {
		return fmt.Errorf("el token aún no es válido (nbf)")
	}

	firma, err := base64.RawURLEncoding.DecodeString(partes[2])
	if err != nil {
		return fmt.Errorf("firma no es base64url válida")
	}
	clave, err := clavePorKid(header.Kid)
	if err != nil {
		return err
	}
	hash := sha256.Sum256([]byte(partes[0] + "." + partes[1]))
	if err := rsa.VerifyPKCS1v15(clave, crypto.SHA256, hash[:], firma); err != nil {
		return fmt.Errorf("la firma no corresponde a la clave del emisor")
	}
	return nil
}

// jwk es una clave del documento JWKS de WSO2 (solo los campos RSA que se usan).
type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

var (
	jwksMu     sync.Mutex
	jwksClaves map[string]*rsa.PublicKey
	jwksCarga  time.Time
)

// clavePorKid devuelve la clave pública del JWKS para el kid. El documento se cachea;
// ante un kid desconocido se recarga (rotación de claves de WSO2), a lo sumo una vez
// por minuto para que un atacante no fuerce recargas con kids inventados.
func clavePorKid(kid string) (*rsa.PublicKey, error) {
	jwksMu.Lock()
	defer jwksMu.Unlock()

	if clave, ok := jwksClaves[kid]; ok {
		return clave, nil
	}
	if jwksClaves != nil && time.Since(jwksCarga) < time.Minute {
		return nil, fmt.Errorf("kid %q no está en el JWKS del emisor", kid)
	}

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Get(jwksURL)
	if err != nil {
		return nil, fmt.Errorf("no se pudo consultar el JWKS: %v", err)
	}
	defer resp.Body.Close()
	var doc struct {
		Keys []jwk `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, fmt.Errorf("JWKS ilegible: %v", err)
	}

	claves := make(map[string]*rsa.PublicKey, len(doc.Keys))
	for _, k := range doc.Keys {
		if k.Kty != "RSA" {
			continue
		}
		clave, err := jwkARSA(k)
		if err != nil {
			continue
		}
		claves[k.Kid] = clave
	}
	jwksClaves = claves
	jwksCarga = time.Now()

	if clave, ok := jwksClaves[kid]; ok {
		return clave, nil
	}
	return nil, fmt.Errorf("kid %q no está en el JWKS del emisor", kid)
}

func jwkARSA(k jwk) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, err
	}
	e := new(big.Int).SetBytes(eBytes)
	if !e.IsInt64() || e.Int64() <= 0 {
		return nil, fmt.Errorf("exponente inválido")
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: int(e.Int64())}, nil
}

// ── Fallback para tokens opacos (no JWT): validación remota vía userinfo ──────

const opacoTTL = 5 * time.Minute

var (
	opacoMu    sync.Mutex
	opacoCache = map[string]time.Time{} // header Authorization completo → vencimiento
)

// validarOpaco valida un token no-JWT contra el endpoint OIDC userinfo. Un token
// aceptado se cachea 5 minutos (el userinfo remoto cuesta ~1 RTT por request si no).
// Los rechazos NO se cachean: un token recién emitido no debe quedar vetado.
func validarOpaco(authHeader string) error {
	opacoMu.Lock()
	if vence, ok := opacoCache[authHeader]; ok && time.Now().Before(vence) {
		opacoMu.Unlock()
		return nil
	}
	opacoMu.Unlock()

	var identidad map[string]interface{}
	if err := helpers.GetUserInfo(authHeader, &identidad); err != nil {
		return fmt.Errorf("el emisor rechazó el token: %v", err)
	}

	opacoMu.Lock()
	for k, vence := range opacoCache { // poda oportunista de vencidos
		if time.Now().After(vence) {
			delete(opacoCache, k)
		}
	}
	opacoCache[authHeader] = time.Now().Add(opacoTTL)
	opacoMu.Unlock()
	return nil
}
