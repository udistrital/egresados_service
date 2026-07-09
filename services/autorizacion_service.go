package services

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/udistrital/egresados_service/helpers"
)

// ErrAccesoDenegado marca los errores de autorización (anti-IDOR): el dueño del
// token no tiene vínculo con el recurso que intenta operar. Los controllers lo
// detectan con errors.Is y lo traducen a HTTP 403.
var ErrAccesoDenegado = errors.New("acceso denegado")

// getUsuariosLocalesDeToken resuelve los usuarios LOCALES del dueño del token:
// userinfo(token) → sub WSO2 → usuario por id_externo. La identidad sale del token
// (misma regla de seguridad del JIT), nunca de parámetros del request. Puede haber
// más de una fila (un egresado SGA que además es usuario de empresa AGORA comparte
// el sub), por eso devuelve todos los ids.
func getUsuariosLocalesDeToken(token string) ([]int, error) {
	info, err := GetUserInfoDeToken(token)
	if err != nil {
		return nil, err
	}
	sub := strings.TrimSpace(info.Sub)
	if sub == "" {
		return nil, fmt.Errorf("%w: el token no expone 'sub' (identificador WSO2)", ErrAccesoDenegado)
	}
	var usuarios []map[string]interface{}
	q := fmt.Sprintf("/usuario?query=IdExterno:%s,Activo:true&fields=Id&limit=0", url.QueryEscape(sub))
	if err := helpers.GetCRUD(token, q, &usuarios); err != nil {
		return nil, err
	}
	ids := make([]int, 0, len(usuarios))
	for _, u := range usuarios {
		if id := toInt(firstOf(u, "id", "Id")); id > 0 {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("%w: el usuario autenticado no está provisionado en el módulo", ErrAccesoDenegado)
	}
	return ids, nil
}

// vinculadoAEmpresa indica si alguno de los usuarios locales tiene vínculo activo
// usuario_empresa con la empresa.
func vinculadoAEmpresa(token string, usuarioIds []int, empresaId int) (bool, error) {
	for _, uid := range usuarioIds {
		var vinculos []map[string]interface{}
		q := fmt.Sprintf("/usuario_empresa?query=Usuario.Id:%d,Empresa.Id:%d,Activo:true&fields=Id&limit=1", uid, empresaId)
		if err := helpers.GetCRUD(token, q, &vinculos); err != nil {
			return false, err
		}
		if len(vinculos) > 0 {
			return true, nil
		}
	}
	return false, nil
}

// getEgresadosDeUsuarios lista los egresado.id de los usuarios locales (la tabla
// egresado se keya por Usuario.Id UNIQUE, así que sale a lo sumo uno por usuario).
func getEgresadosDeUsuarios(token string, usuarioIds []int) ([]int, error) {
	var ids []int
	for _, uid := range usuarioIds {
		var egresados []map[string]interface{}
		q := fmt.Sprintf("/egresado?query=Usuario.Id:%d,Activo:true&fields=Id&limit=1", uid)
		if err := helpers.GetCRUD(token, q, &egresados); err != nil {
			return nil, err
		}
		for _, e := range egresados {
			if id := toInt(firstOf(e, "id", "Id")); id > 0 {
				ids = append(ids, id)
			}
		}
	}
	return ids, nil
}

// empresaDeBeneficio resuelve la empresa dueña de un beneficio.
func empresaDeBeneficio(token string, beneficioId int) (int, error) {
	var ben map[string]interface{}
	if err := helpers.GetCRUD(token, fmt.Sprintf("/beneficio/%d", beneficioId), &ben); err != nil {
		return 0, fmt.Errorf("beneficio %d no encontrado", beneficioId)
	}
	emp, _ := ben["empresa"].(map[string]interface{})
	empresaId := toInt(firstOf(emp, "id", "Id"))
	if empresaId <= 0 {
		return 0, fmt.Errorf("no se pudo determinar la empresa del beneficio %d", beneficioId)
	}
	return empresaId, nil
}

// datosDeSolicitud resuelve (egresadoId, beneficioId) de una solicitud.
func datosDeSolicitud(token string, solicitudId int) (int, int, error) {
	var s map[string]interface{}
	if err := helpers.GetCRUD(token, fmt.Sprintf("/solicitud_beneficio/%d", solicitudId), &s); err != nil {
		return 0, 0, fmt.Errorf("solicitud %d no encontrada", solicitudId)
	}
	eg, _ := s["egresado"].(map[string]interface{})
	ben, _ := s["beneficio"].(map[string]interface{})
	return toInt(firstOf(eg, "id", "Id")), toInt(firstOf(ben, "id", "Id")), nil
}

// solicitudDeDocumento resuelve la solicitud a la que pertenece un documento_solicitud.
func solicitudDeDocumento(token string, documentoSolicitudId int) (int, error) {
	var doc map[string]interface{}
	if err := helpers.GetCRUD(token, fmt.Sprintf("/documento_solicitud/%d", documentoSolicitudId), &doc); err != nil {
		return 0, fmt.Errorf("documento %d no encontrado", documentoSolicitudId)
	}
	sol, _ := doc["solicitud_beneficio"].(map[string]interface{})
	solicitudId := toInt(firstOf(sol, "id", "Id"))
	if solicitudId <= 0 {
		return 0, fmt.Errorf("no se pudo determinar la solicitud del documento %d", documentoSolicitudId)
	}
	return solicitudId, nil
}

// ── Pata de EMPRESA ───────────────────────────────────────────────────────────

// VerificarAccesoEmpresa exige que el dueño del token tenga un vínculo activo
// usuario_empresa con la empresa. Protege los endpoints /v1/empresas/:id/... de
// dueño (bandeja, mis beneficios, publicar).
func VerificarAccesoEmpresa(token string, empresaId int) error {
	usuarioIds, err := getUsuariosLocalesDeToken(token)
	if err != nil {
		return err
	}
	ok, err := vinculadoAEmpresa(token, usuarioIds, empresaId)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: el usuario autenticado no está vinculado a la empresa %d", ErrAccesoDenegado, empresaId)
	}
	return nil
}

// VerificarUsuarioDelToken exige que el :usuario_id del path sea uno de los usuarios
// locales del dueño del token (GET /v1/usuarios/:usuario_id/empresas).
func VerificarUsuarioDelToken(token string, usuarioId int) error {
	ids, err := getUsuariosLocalesDeToken(token)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if id == usuarioId {
			return nil
		}
	}
	return fmt.Errorf("%w: el usuario %d no corresponde al usuario autenticado", ErrAccesoDenegado, usuarioId)
}

// VerificarAccesoBeneficio exige que el beneficio pertenezca a una empresa del dueño
// del token (PUT /v1/beneficios/:id).
func VerificarAccesoBeneficio(token string, beneficioId int) error {
	empresaId, err := empresaDeBeneficio(token, beneficioId)
	if err != nil {
		return err
	}
	return VerificarAccesoEmpresa(token, empresaId)
}

// VerificarAccesoSolicitudEmpresa exige que la solicitud pertenezca a un beneficio de
// una empresa del dueño del token (acciones de empresa sobre solicitudes: responder).
func VerificarAccesoSolicitudEmpresa(token string, solicitudId int) error {
	beneficioId, err := getBeneficioIdDeSolicitud(token, solicitudId)
	if err != nil {
		return err
	}
	return VerificarAccesoBeneficio(token, beneficioId)
}

// VerificarAccesoDocumentoEmpresa exige que el documento_solicitud pertenezca a una
// solicitud de la empresa del dueño del token (PUT /v1/documentos/:doc_id/comentario).
func VerificarAccesoDocumentoEmpresa(token string, documentoSolicitudId int) error {
	solicitudId, err := solicitudDeDocumento(token, documentoSolicitudId)
	if err != nil {
		return err
	}
	return VerificarAccesoSolicitudEmpresa(token, solicitudId)
}

// ── Pata de EGRESADO ──────────────────────────────────────────────────────────

// VerificarEgresadoDelToken exige que el egresado_id (del path o del body) sea el
// egresado del dueño del token (mis-solicitudes, resumen, crear solicitud).
func VerificarEgresadoDelToken(token string, egresadoId int) error {
	usuarioIds, err := getUsuariosLocalesDeToken(token)
	if err != nil {
		return err
	}
	egresadoIds, err := getEgresadosDeUsuarios(token, usuarioIds)
	if err != nil {
		return err
	}
	for _, id := range egresadoIds {
		if id == egresadoId {
			return nil
		}
	}
	return fmt.Errorf("%w: el egresado %d no corresponde al usuario autenticado", ErrAccesoDenegado, egresadoId)
}

// VerificarAccesoSolicitudEgresado exige que la solicitud sea del egresado dueño del
// token (cancelar, subir/eliminar documentos).
func VerificarAccesoSolicitudEgresado(token string, solicitudId int) error {
	egresadoId, _, err := datosDeSolicitud(token, solicitudId)
	if err != nil {
		return err
	}
	if egresadoId <= 0 {
		return fmt.Errorf("no se pudo determinar el egresado de la solicitud %d", solicitudId)
	}
	return VerificarEgresadoDelToken(token, egresadoId)
}

// ── Endpoints bidireccionales ─────────────────────────────────────────────────

// VerificarParticipanteSolicitud exige que el dueño del token sea PARTE de la
// solicitud: el egresado que la creó O un usuario de la empresa del beneficio
// (mensajes, documentos, comprobante). Un tercero autenticado no puede leer el hilo
// ni los PDFs (RNF-002b / Ley 1581).
func VerificarParticipanteSolicitud(token string, solicitudId int) error {
	usuarioIds, err := getUsuariosLocalesDeToken(token)
	if err != nil {
		return err
	}
	egresadoId, beneficioId, err := datosDeSolicitud(token, solicitudId)
	if err != nil {
		return err
	}

	if egresadoId > 0 {
		egresadoIds, err := getEgresadosDeUsuarios(token, usuarioIds)
		if err != nil {
			return err
		}
		for _, id := range egresadoIds {
			if id == egresadoId {
				return nil
			}
		}
	}
	if beneficioId > 0 {
		empresaId, err := empresaDeBeneficio(token, beneficioId)
		if err != nil {
			return err
		}
		ok, err := vinculadoAEmpresa(token, usuarioIds, empresaId)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
	}
	return fmt.Errorf("%w: el usuario autenticado no participa en la solicitud %d", ErrAccesoDenegado, solicitudId)
}

// VerificarParticipanteDocumento es VerificarParticipanteSolicitud partiendo del
// documento_solicitud (GET /v1/documentos/:doc_id/archivo).
func VerificarParticipanteDocumento(token string, documentoSolicitudId int) error {
	solicitudId, err := solicitudDeDocumento(token, documentoSolicitudId)
	if err != nil {
		return err
	}
	return VerificarParticipanteSolicitud(token, solicitudId)
}
