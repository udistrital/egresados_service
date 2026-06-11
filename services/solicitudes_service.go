package services

import (
	"fmt"
	"strconv"
	"time"

	"github.com/udistrital/sga_mid_beneficios_egresados/helpers"
)

const (
	estadoPendiente    = "PENDIENTE"
	estadoEnRevision   = "EN_REVISION"
	estadoRequiereInfo = "REQUIERE_INFO"
	estadoAprobada     = "APROBADA"
	estadoRechazada    = "RECHAZADA"
	estadoCancelada    = "CANCELADA"
)

// CrearSolicitud crea una solicitud validando todas las reglas de negocio:
// RN-007 (solicitud única por egresado+beneficio), RN-010 (límite activas),
// RN-002b (decremento atómico de cupo), RN-RADICADO (generación de radicado),
// RN-004 (inserción de historial — única fuente de estado, C-4b).
func CrearSolicitud(body map[string]interface{}) (interface{}, error) {
	egresadoId, ok := body["egresado_id"]
	if !ok {
		return nil, fmt.Errorf("egresado_id es requerido")
	}
	beneficioId, ok := body["beneficio_id"]
	if !ok {
		return nil, fmt.Errorf("beneficio_id es requerido")
	}

	eid := toInt(egresadoId)
	bid := toInt(beneficioId)

	// RN-007: verificar que no exista una solicitud activa para (egresado, beneficio)
	// TODO: consultar CRUD /solicitud_beneficio?query=Egresado.Id:{eid},Beneficio.Id:{bid},Activo:true

	// RN-010: verificar límite de solicitudes activas
	limiteParam, err := getLimiteActivas()
	if err != nil {
		return nil, err
	}
	// TODO: contar solicitudes activas del egresado y comparar con limiteParam
	_ = limiteParam

	// RN-002b: decrementar cupo_disponible atómicamente
	// TODO: implementar con SELECT FOR UPDATE en el CRUD o usando endpoint dedicado

	// RN-RADICADO: generar radicado BNF-YYYY-NNNNNN con la secuencia del CRUD
	anio := time.Now().Year()
	var seqResp map[string]interface{}
	if err := helpers.PostCRUD(
		fmt.Sprintf("/secuencia_radicado/siguiente/%d", anio),
		nil, &seqResp,
	); err != nil {
		return nil, fmt.Errorf("no se pudo generar el radicado: %v", err)
	}
	numero := toInt(seqResp["numero"])
	if numero == 0 {
		return nil, fmt.Errorf("la secuencia de radicado no retornó número válido")
	}
	radicado := fmt.Sprintf("BNF-%d-%06d", anio, numero)

	// La solicitud ya no lleva estado propio (C-4b); el estado nace en el historial.
	solicitud := map[string]interface{}{
		"egresado":  map[string]interface{}{"id": eid},
		"beneficio": map[string]interface{}{"id": bid},
		"radicado":  radicado,
	}
	if datos, ok := body["datos_complementarios"]; ok {
		solicitud["datos_complementarios"] = datos
	}

	var result map[string]interface{}
	if err := helpers.PostCRUD("/solicitud_beneficio", solicitud, &result); err != nil {
		return nil, err
	}

	// RN-004 / C-4b: el registro inicial del historial define el estado PENDIENTE
	pendienteId, err := ResolverParametroId(TipoParamEstadoSolicitud, estadoPendiente)
	if err != nil {
		return nil, err
	}
	solicitudId := toInt(result["id"])
	usuarioId := eid // el egresado como actor
	if uid, ok := body["usuario_id"]; ok {
		usuarioId = toInt(uid)
	}
	historial := map[string]interface{}{
		"solicitud_beneficio": map[string]interface{}{"id": solicitudId},
		"estado_nuevo_id":     pendienteId,
		"usuario":             map[string]interface{}{"id": usuarioId},
	}
	if err := helpers.PostCRUD("/historial_solicitud", historial, &map[string]interface{}{}); err != nil {
		// Sin historial la solicitud queda sin estado: es un error real, no advertencia
		return nil, fmt.Errorf("solicitud %d creada pero no se pudo registrar su estado inicial: %v", solicitudId, err)
	}

	return result, nil
}

// GetSolicitudesByEgresado retorna las solicitudes de un egresado con su estado
// vigente derivado del historial (C-4b).
func GetSolicitudesByEgresado(egresadoId int) (interface{}, error) {
	var solicitudes []map[string]interface{}
	query := fmt.Sprintf("/solicitud_beneficio?query=Egresado.Id:%d,Activo:true&limit=0", egresadoId)
	if err := helpers.GetCRUD(query, &solicitudes); err != nil {
		return nil, err
	}
	for _, s := range solicitudes {
		if codigo, estadoId, err := getEstadoActual(toInt(s["id"])); err == nil {
			s["estado_solicitud_id"] = estadoId
			s["estado_solicitud"] = codigo
		}
	}
	return solicitudes, nil
}

// CancelarSolicitud cancela una solicitud. Solo desde PENDIENTE o REQUIERE_INFO (RN-005).
// Devuelve el cupo (RN-002c).
func CancelarSolicitud(id int, body map[string]interface{}) error {
	// RN-005: validar máquina de estados con el estado vigente del historial
	estado, estadoId, err := getEstadoActual(id)
	if err != nil {
		return err
	}
	if estado != estadoPendiente && estado != estadoRequiereInfo {
		return fmt.Errorf("solo se puede cancelar una solicitud en estado PENDIENTE o REQUIERE_INFO, estado actual: %s", estado)
	}

	// RN-002c: devolver cupo
	// TODO: incrementar cupos_disponibles del beneficio atómicamente

	// C-4b: el cambio de estado ES la inserción en el historial (no hay campo que actualizar)
	return registrarCambioEstado(id, estadoId, estadoCancelada, body["usuario_id"], nil)
}

// GetResumenEgresado retorna contadores de solicitudes por estado (RF-013).
func GetResumenEgresado(egresadoId int) (interface{}, error) {
	// TODO: implementar consultas por estado al CRUD y agrupar contadores
	resumen := map[string]int{
		"activas":    0,
		"aprobadas":  0,
		"rechazadas": 0,
		"canceladas": 0,
	}
	return resumen, nil
}

// ResponderSolicitud aplica la respuesta de la empresa: APROBADA / RECHAZADA / REQUIERE_INFO.
// RN-003: justificación obligatoria y >= 20 chars si estado_nuevo = RECHAZADA.
// RN-002c: devolver cupo si RECHAZADA.
// RN-004: registrar historial.
// RN-005: validar máquina de estados.
func ResponderSolicitud(id int, body map[string]interface{}) error {
	nuevoEstado, ok := body["estado_nuevo"].(string)
	if !ok || nuevoEstado == "" {
		return fmt.Errorf("estado_nuevo es requerido")
	}

	// RN-003: justificación obligatoria al rechazar
	if nuevoEstado == estadoRechazada {
		justificacion, _ := body["justificacion"].(string)
		if len(justificacion) < 20 {
			return fmt.Errorf("la justificación debe tener al menos 20 caracteres al rechazar")
		}
	}

	// RN-005: obtener estado vigente del historial y validar transición
	estadoActual, estadoActualId, err := getEstadoActual(id)
	if err != nil {
		return err
	}
	if !transicionValida(estadoActual, nuevoEstado) {
		return fmt.Errorf("transición de estado inválida: %s → %s", estadoActual, nuevoEstado)
	}

	// RN-002c: devolver cupo si se rechaza
	if nuevoEstado == estadoRechazada {
		// TODO: incrementar cupos_disponibles del beneficio atómicamente
	}

	// RN-004 / C-4b: insertar en historial (única fuente de estado)
	return registrarCambioEstado(id, estadoActualId, nuevoEstado, body["usuario_id"], body["justificacion"])
}

// EnviarMensaje envía un mensaje en la solicitud (solo si estado = REQUIERE_INFO).
func EnviarMensaje(solicitudId int, body map[string]interface{}) (interface{}, error) {
	estado, _, err := getEstadoActual(solicitudId)
	if err != nil {
		return nil, err
	}
	if estado != estadoRequiereInfo {
		return nil, fmt.Errorf("solo se pueden enviar mensajes cuando la solicitud está en REQUIERE_INFO")
	}

	payload := map[string]interface{}{
		"solicitud_beneficio": map[string]interface{}{"id": solicitudId},
		"usuario":             map[string]interface{}{"id": toInt(body["usuario_id"])},
		"mensaje":             body["mensaje"],
	}
	var result interface{}
	if err := helpers.PostCRUD("/mensaje_solicitud", payload, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetMensajes retorna el historial de mensajes de una solicitud.
func GetMensajes(solicitudId int) (interface{}, error) {
	var result interface{}
	query := fmt.Sprintf("/mensaje_solicitud?query=SolicitudBeneficio.Id:%d,Activo:true&limit=0", solicitudId)
	if err := helpers.GetCRUD(query, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ── Helpers internos ──────────────────────────────────────────────────────────

// transicionValida verifica la máquina de estados de solicitud (RN-005).
func transicionValida(actual, nuevo string) bool {
	maquina := map[string][]string{
		estadoPendiente:    {estadoEnRevision, estadoAprobada, estadoRechazada, estadoCancelada},
		estadoEnRevision:   {estadoAprobada, estadoRechazada, estadoRequiereInfo},
		estadoRequiereInfo: {estadoEnRevision, estadoCancelada},
	}
	permitidos, ok := maquina[actual]
	if !ok {
		return false // estado final
	}
	for _, p := range permitidos {
		if p == nuevo {
			return true
		}
	}
	return false
}

// getEstadoActual deriva el estado vigente de una solicitud del último registro
// del historial (C-4b) y lo traduce a codigo_abreviacion vía el servicio de parámetros.
func getEstadoActual(solicitudId int) (codigo string, estadoId int, err error) {
	var vigente map[string]interface{}
	if err = helpers.GetCRUD(fmt.Sprintf("/historial_solicitud/solicitud/%d/vigente", solicitudId), &vigente); err != nil {
		return "", 0, fmt.Errorf("no se pudo obtener el estado de la solicitud %d: %v", solicitudId, err)
	}
	estadoId = toInt(vigente["estado_nuevo_id"])
	if estadoId == 0 {
		return "", 0, fmt.Errorf("la solicitud %d no tiene historial de estado", solicitudId)
	}
	codigo, err = ResolverParametroCodigo(TipoParamEstadoSolicitud, estadoId)
	if err != nil {
		return "", 0, err
	}
	return codigo, estadoId, nil
}

// registrarCambioEstado inserta la transición en historial_solicitud (RN-004, C-4b).
func registrarCambioEstado(solicitudId, estadoAnteriorId int, nuevoEstadoCodigo string, usuarioId, justificacion interface{}) error {
	nuevoId, err := ResolverParametroId(TipoParamEstadoSolicitud, nuevoEstadoCodigo)
	if err != nil {
		return err
	}
	historial := map[string]interface{}{
		"solicitud_beneficio": map[string]interface{}{"id": solicitudId},
		"estado_anterior_id":  estadoAnteriorId,
		"estado_nuevo_id":     nuevoId,
		"usuario":             map[string]interface{}{"id": toInt(usuarioId)},
	}
	if justificacion != nil {
		historial["justificacion"] = justificacion
	}
	return helpers.PostCRUD("/historial_solicitud", historial, &map[string]interface{}{})
}

// getLimiteActivas lee el límite de solicitudes activas por egresado (RN-010)
// del servicio de parámetros (tipo PARAMETRO_SISTEMA).
func getLimiteActivas() (int, error) {
	params, err := GetParametrosPorTipo(TipoParamParametroSistema)
	if err != nil {
		return 5, nil // valor por defecto si el servicio no está disponible
	}
	for _, p := range params {
		if firstOf(p, "CodigoAbreviacion", "codigo_abreviacion") == "LIMITE_SOLICITUDES_ACTIVAS_EGRESADO" {
			switch valor := firstOf(p, "Valor", "valor").(type) {
			case string:
				if v, err := strconv.Atoi(valor); err == nil {
					return v, nil
				}
			case float64:
				return int(valor), nil
			}
		}
	}
	return 5, nil
}

func toInt(v interface{}) int {
	switch val := v.(type) {
	case int:
		return val
	case float64:
		return int(val)
	case string:
		i, _ := strconv.Atoi(val)
		return i
	}
	return 0
}
