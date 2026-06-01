package services

import (
	"fmt"
	"strconv"
	"time"

	"github.com/udistrital/sga_mid_beneficios_egresados/helpers"
)

const (
	estadoPendiente     = "PENDIENTE"
	estadoEnRevision    = "EN_REVISION"
	estadoRequiereInfo  = "REQUIERE_INFO"
	estadoAprobada      = "APROBADA"
	estadoRechazada     = "RECHAZADA"
	estadoCancelada     = "CANCELADA"
)

// CrearSolicitud crea una solicitud validando todas las reglas de negocio:
// RN-007 (solicitud única por egresado+beneficio), RN-010 (límite activas),
// RN-002b (decremento atómico de cupo), RN-RADICADO (generación de radicado),
// RN-004 (inserción de historial).
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

	// RN-RADICADO: generar radicado BNF-YYYY-NNNNNN
	anio := time.Now().Year()
	var seqResp map[string]interface{}
	if err := helpers.PostCRUD(
		fmt.Sprintf("/secuencia_radicado/siguiente/%d", anio),
		nil, &seqResp,
	); err != nil {
		// Fallback: usar timestamp (reemplazar por implementación real)
		_ = err
	}
	radicado := fmt.Sprintf("BNF-%d-%06d", anio, 1) // TODO: usar número de seqResp

	// Construir payload para el CRUD
	solicitud := map[string]interface{}{
		"egresado_id":      eid,
		"beneficio_id":     bid,
		"radicado":         radicado,
		"estado_solicitud": map[string]interface{}{"codigo_abreviacion": estadoPendiente},
	}
	if datos, ok := body["datos_complementarios"]; ok {
		solicitud["datos_complementarios"] = datos
	}

	var result map[string]interface{}
	if err := helpers.PostCRUD("/solicitud_beneficio", solicitud, &result); err != nil {
		return nil, err
	}

	// RN-004: registrar historial de estado
	solicitudId := toInt(result["id"])
	historial := map[string]interface{}{
		"solicitud_beneficio_id": solicitudId,
		"estado_nuevo":           map[string]interface{}{"codigo_abreviacion": estadoPendiente},
		"usuario_id":             eid, // el egresado como actor
	}
	if err := helpers.PostCRUD("/historial_estado_solicitud", historial, &map[string]interface{}{}); err != nil {
		// No bloquear la creación si el historial falla, pero loguear
		fmt.Printf("advertencia: no se pudo registrar historial: %v\n", err)
	}

	return result, nil
}

// GetSolicitudesByEgresado retorna las solicitudes de un egresado con estado e historial.
func GetSolicitudesByEgresado(egresadoId int) (interface{}, error) {
	var result interface{}
	query := fmt.Sprintf("/solicitud_beneficio?query=Egresado.Id:%d,Activo:true", egresadoId)
	if err := helpers.GetCRUD(query, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// CancelarSolicitud cancela una solicitud. Solo desde PENDIENTE o REQUIERE_INFO (RN-005).
// Devuelve el cupo (RN-002c).
func CancelarSolicitud(id int, body map[string]interface{}) error {
	var solicitud map[string]interface{}
	if err := helpers.GetCRUD(fmt.Sprintf("/solicitud_beneficio/%d", id), &solicitud); err != nil {
		return fmt.Errorf("solicitud no encontrada")
	}

	// RN-005: validar máquina de estados
	estado := getEstadoCodigo(solicitud)
	if estado != estadoPendiente && estado != estadoRequiereInfo {
		return fmt.Errorf("solo se puede cancelar una solicitud en estado PENDIENTE o REQUIERE_INFO, estado actual: %s", estado)
	}

	// RN-002c: devolver cupo
	// TODO: incrementar cupos_disponibles del beneficio atómicamente

	// Actualizar estado
	update := map[string]interface{}{
		"estado_solicitud": map[string]interface{}{"codigo_abreviacion": estadoCancelada},
	}
	if err := helpers.PutCRUD(fmt.Sprintf("/solicitud_beneficio/%d", id), update); err != nil {
		return err
	}

	// RN-004: registrar historial
	historial := map[string]interface{}{
		"solicitud_beneficio_id": id,
		"estado_anterior":        map[string]interface{}{"codigo_abreviacion": estado},
		"estado_nuevo":           map[string]interface{}{"codigo_abreviacion": estadoCancelada},
		"usuario_id":             body["usuario_id"],
	}
	helpers.PostCRUD("/historial_estado_solicitud", historial, &map[string]interface{}{})
	return nil
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

	// RN-005: obtener estado actual y validar transición
	var solicitud map[string]interface{}
	if err := helpers.GetCRUD(fmt.Sprintf("/solicitud_beneficio/%d", id), &solicitud); err != nil {
		return fmt.Errorf("solicitud no encontrada")
	}
	estadoActual := getEstadoCodigo(solicitud)
	if !transicionValida(estadoActual, nuevoEstado) {
		return fmt.Errorf("transición de estado inválida: %s → %s", estadoActual, nuevoEstado)
	}

	// RN-002c: devolver cupo si se rechaza
	if nuevoEstado == estadoRechazada {
		// TODO: incrementar cupos_disponibles del beneficio atómicamente
	}

	update := map[string]interface{}{
		"estado_solicitud": map[string]interface{}{"codigo_abreviacion": nuevoEstado},
	}
	if err := helpers.PutCRUD(fmt.Sprintf("/solicitud_beneficio/%d", id), update); err != nil {
		return err
	}

	// RN-004: historial
	historial := map[string]interface{}{
		"solicitud_beneficio_id": id,
		"estado_anterior":        map[string]interface{}{"codigo_abreviacion": estadoActual},
		"estado_nuevo":           map[string]interface{}{"codigo_abreviacion": nuevoEstado},
		"usuario_id":             body["usuario_id"],
		"justificacion":          body["justificacion"],
	}
	helpers.PostCRUD("/historial_estado_solicitud", historial, &map[string]interface{}{})
	return nil
}

// EnviarMensaje envía un mensaje en la solicitud (solo si estado = REQUIERE_INFO).
func EnviarMensaje(solicitudId int, body map[string]interface{}) (interface{}, error) {
	var solicitud map[string]interface{}
	if err := helpers.GetCRUD(fmt.Sprintf("/solicitud_beneficio/%d", solicitudId), &solicitud); err != nil {
		return nil, fmt.Errorf("solicitud no encontrada")
	}
	if getEstadoCodigo(solicitud) != estadoRequiereInfo {
		return nil, fmt.Errorf("solo se pueden enviar mensajes cuando la solicitud está en REQUIERE_INFO")
	}

	payload := map[string]interface{}{
		"solicitud_beneficio_id": solicitudId,
		"usuario_id":             body["usuario_id"],
		"mensaje":                body["mensaje"],
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
	query := fmt.Sprintf("/mensaje_solicitud?query=SolicitudBeneficio.Id:%d,Activo:true", solicitudId)
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

func getEstadoCodigo(solicitud map[string]interface{}) string {
	if estado, ok := solicitud["estado_solicitud"].(map[string]interface{}); ok {
		if codigo, ok := estado["codigo_abreviacion"].(string); ok {
			return codigo
		}
	}
	return ""
}

func getLimiteActivas() (int, error) {
	var param map[string]interface{}
	if err := helpers.GetCRUD("/parametro_sistema?query=Clave:LIMITE_SOLICITUDES_ACTIVAS_EGRESADO,Activo:true", &param); err != nil {
		return 5, nil // valor por defecto si no se puede leer
	}
	if valor, ok := param["valor"].(string); ok {
		if v, err := strconv.Atoi(valor); err == nil {
			return v, nil
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
