package services

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/udistrital/egresados_service/helpers"
)

const (
	estadoPendiente    = "PENDIENTE"
	estadoEnRevision   = "EN_REVISION"
	estadoRequiereInfo = "REQUIERE_INFO"
	estadoAprobada     = "APROBADA"
	estadoRechazada    = "RECHAZADA"
	estadoCancelada    = "CANCELADA"
)

// CrearSolicitud crea una solicitud validando RN-007 (única por egresado+beneficio),
// RN-010 (límite de activas) y RN-002b (cupo atómico), y registra el estado inicial
// en el historial (C-4b).
func CrearSolicitud(token string, body map[string]interface{}) (interface{}, error) {
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

	// Anti-IDOR: el egresado_id del body debe ser el del dueño del token.
	if err := VerificarEgresadoDelToken(token, eid); err != nil {
		return nil, err
	}

	activas, err := beneficiosConSolicitudActiva(token, eid)
	if err != nil {
		return nil, err
	}
	// RN-007: solicitud única en curso por (egresado, beneficio).
	for _, existenteBid := range activas {
		if existenteBid == bid {
			return nil, fmt.Errorf("ya tienes una solicitud en curso para este beneficio")
		}
	}
	// RN-010: límite de solicitudes activas por egresado.
	limite, err := getLimiteActivas(token)
	if err != nil {
		return nil, err
	}
	if len(activas) >= limite {
		return nil, fmt.Errorf("alcanzaste el límite de %d solicitudes activas", limite)
	}

	// RN-002b: reservar el cupo antes de crear; si algo falla después se devuelve (RN-002c).
	descontado, err := descontarCupo(token, bid)
	if err != nil {
		return nil, fmt.Errorf("no se pudo verificar el cupo del beneficio %d: %v", bid, err)
	}
	if !descontado {
		return nil, fmt.Errorf("el beneficio %d no tiene cupos disponibles", bid)
	}

	// La solicitud no lleva estado propio (C-4b); el radicado lo genera la BD al insertar.
	solicitud := map[string]interface{}{
		"egresado":  map[string]interface{}{"id": eid},
		"beneficio": map[string]interface{}{"id": bid},
	}
	// datos_complementarios es JSONB: el texto libre se normaliza a JSON válido.
	if datos := aJSONB(body["datos_complementarios"]); datos != nil {
		solicitud["datos_complementarios"] = datos
	}

	var result map[string]interface{}
	if err := helpers.PostCRUD(token, "/solicitud-beneficio", solicitud, &result); err != nil {
		devolverCupo(token, bid)
		return nil, err
	}

	// El registro inicial del historial define el estado PENDIENTE (RN-004, C-4b).
	pendienteId, err := ResolverParametroId(token, TipoParamEstadoSolicitud, estadoPendiente)
	if err != nil {
		devolverCupo(token, bid)
		return nil, err
	}
	solicitudId := toInt(result["id"])
	usuarioId := eid
	if uid, ok := body["usuario_id"]; ok {
		usuarioId = toInt(uid)
	}
	historial := map[string]interface{}{
		"solicitud_beneficio": map[string]interface{}{"id": solicitudId},
		"estado_nuevo_id":     pendienteId,
		"usuario":             map[string]interface{}{"id": usuarioId},
	}
	if err := helpers.PostCRUD(token, "/historial-solicitud", historial, &map[string]interface{}{}); err != nil {
		devolverCupo(token, bid)
		return nil, fmt.Errorf("solicitud %d creada pero no se pudo registrar su estado inicial: %v", solicitudId, err)
	}

	// Leer el radicado que generó la BD para devolverlo en la respuesta.
	var creada map[string]interface{}
	if err := helpers.GetCRUD(token, fmt.Sprintf("/solicitud-beneficio/%d", solicitudId), &creada); err == nil {
		if rad := asString(firstOf(creada, "radicado", "Radicado")); rad != "" {
			result["radicado"] = rad
		}
	}

	return result, nil
}

// GetSolicitudesByEgresado retorna las solicitudes de un egresado con su estado
// vigente derivado del historial (C-4b).
func GetSolicitudesByEgresado(token string, egresadoId int) (interface{}, error) {
	var solicitudes []map[string]interface{}
	query := fmt.Sprintf("/solicitud-beneficio?query=Egresado.Id:%d,Activo:true&limit=0", egresadoId)
	if err := helpers.GetCRUD(token, query, &solicitudes); err != nil {
		return nil, err
	}
	for _, s := range solicitudes {
		if codigo, estadoId, err := getEstadoActual(token, toInt(s["id"])); err == nil {
			s["estado_solicitud_id"] = estadoId
			s["estado_solicitud"] = codigo
		}
		s["datos_complementarios"] = desdeJSONB(s["datos_complementarios"])
	}
	return solicitudes, nil
}

// aJSONB normaliza el valor del formulario para la columna JSONB: JSON válido pasa
// tal cual; texto libre se codifica como string JSON; vacío → nil (columna NULL).
func aJSONB(v interface{}) interface{} {
	s, esString := v.(string)
	if !esString {
		return v
	}
	if s == "" {
		return nil
	}
	if json.Valid([]byte(s)) {
		return s
	}
	b, _ := json.Marshal(s)
	return string(b)
}

// desdeJSONB deshace aJSONB: un string JSON escalar vuelve a texto plano.
func desdeJSONB(v interface{}) interface{} {
	raw, esString := v.(string)
	if !esString {
		return v
	}
	var texto string
	if err := json.Unmarshal([]byte(raw), &texto); err == nil {
		return texto
	}
	return v
}

// CancelarSolicitud cancela una solicitud en curso (RN-005) y devuelve el cupo (RN-002c).
func CancelarSolicitud(token string, id int, body map[string]interface{}) error {
	estado, estadoId, err := getEstadoActual(token, id)
	if err != nil {
		return err
	}
	if estado != estadoPendiente && estado != estadoRequiereInfo && estado != estadoEnRevision {
		return fmt.Errorf("solo se puede cancelar una solicitud en curso (PENDIENTE, REQUIERE_INFO o EN_REVISION), estado actual: %s", estado)
	}

	// RN-002c: obtener el beneficio antes del cambio para poder devolver el cupo.
	bid, err := getBeneficioIdDeSolicitud(token, id)
	if err != nil {
		return err
	}

	if err := registrarCambioEstado(token, id, estadoId, estadoCancelada, body["usuario_id"], nil, "", ""); err != nil {
		return err
	}
	devolverCupo(token, bid) // RN-002c
	return nil
}

// GetResumenEgresado retorna contadores de solicitudes por estado vigente (RF-013).
func GetResumenEgresado(token string, egresadoId int) (interface{}, error) {
	resumen := map[string]int{
		"activas":    0,
		"aprobadas":  0,
		"rechazadas": 0,
		"canceladas": 0,
	}
	var solicitudes []map[string]interface{}
	q := fmt.Sprintf("/solicitud-beneficio?query=Egresado.Id:%d,Activo:true&limit=0", egresadoId)
	if err := helpers.GetCRUD(token, q, &solicitudes); err != nil {
		return nil, err
	}
	for _, s := range solicitudes {
		codigo, _, err := getEstadoActual(token, toInt(s["id"]))
		if err != nil {
			continue
		}
		switch codigo {
		case estadoAprobada:
			resumen["aprobadas"]++
		case estadoRechazada:
			resumen["rechazadas"]++
		case estadoCancelada:
			resumen["canceladas"]++
		default:
			if esEstadoNoTerminal(codigo) {
				resumen["activas"]++
			}
		}
	}
	return resumen, nil
}

// ResponderSolicitud aplica la respuesta de la empresa (APROBADA / RECHAZADA /
// REQUIERE_INFO) validando la máquina de estados (RN-005) y registrando el historial
// (RN-004). Al aprobar puede adjuntarse un comprobante PDF opcional.
func ResponderSolicitud(token string, id int, body map[string]interface{}) error {
	nuevoEstado, ok := body["estado_nuevo"].(string)
	if !ok || nuevoEstado == "" {
		return fmt.Errorf("estado_nuevo es requerido")
	}
	justificacion, _ := body["justificacion"].(string)

	comprobante, _ := body["comprobante"].(map[string]interface{})
	if comprobante != nil && nuevoEstado != estadoAprobada {
		return fmt.Errorf("el comprobante solo se puede adjuntar al aprobar la solicitud")
	}

	// RN-003: si la empresa rechaza sin justificación se registra un texto por defecto.
	if nuevoEstado == estadoRechazada && strings.TrimSpace(justificacion) == "" {
		justificacion = "Solicitud rechazada sin perjuicio: la empresa no otorgó el " +
			"beneficio en esta oportunidad. Esta decisión no afecta tus futuras " +
			"postulaciones a este u otros beneficios del módulo."
		body["justificacion"] = justificacion
	}

	estadoActual, estadoActualId, err := getEstadoActual(token, id)
	if err != nil {
		return err
	}

	// Pedir información estando ya en REQUIERE_INFO no es transición: la pregunta
	// adicional se publica como un mensaje más del hilo.
	if estadoActual == estadoRequiereInfo && nuevoEstado == estadoRequiereInfo {
		if strings.TrimSpace(justificacion) == "" {
			return fmt.Errorf("la solicitud ya está en REQUIERE_INFO; escribe el mensaje para el egresado")
		}
		_, err := EnviarMensaje(token, id, map[string]interface{}{
			"usuario_id": body["usuario_id"], "mensaje": justificacion,
		})
		return err
	}

	if !transicionValida(estadoActual, nuevoEstado) {
		return fmt.Errorf("transición de estado inválida: %s → %s", estadoActual, nuevoEstado)
	}

	// RN-002c: si se rechaza, obtener el beneficio antes para devolver el cupo.
	var bid int
	if nuevoEstado == estadoRechazada {
		if bid, err = getBeneficioIdDeSolicitud(token, id); err != nil {
			return err
		}
	}

	// El comprobante se sube antes de tocar el historial: si falla, la aprobación
	// se aborta sin dejar un estado a medias.
	var nombreComprobante, enlaceComprobante string
	if comprobante != nil {
		nombreComprobante, _ = comprobante["nombre_archivo"].(string)
		fileBase64, _ := comprobante["file"].(string)
		if nombreComprobante == "" || fileBase64 == "" {
			return fmt.Errorf("el comprobante requiere nombre_archivo y file")
		}
		enlace, err := SubirDocumentoGestor(token, nombreComprobante, "Comprobante de aprobación de beneficio", fileBase64)
		if err != nil {
			return fmt.Errorf("no se pudo adjuntar el comprobante: %v", err)
		}
		enlaceComprobante = enlace
	}

	if err := registrarCambioEstado(token, id, estadoActualId, nuevoEstado, body["usuario_id"], body["justificacion"], nombreComprobante, enlaceComprobante); err != nil {
		return err
	}
	if nuevoEstado == estadoRechazada {
		devolverCupo(token, bid) // RN-002c
	}

	// La nota de "pedir información" se publica como mensaje del hilo: la
	// justificación del historial no es visible para el egresado.
	if nuevoEstado == estadoRequiereInfo && strings.TrimSpace(justificacion) != "" {
		if _, err := EnviarMensaje(token, id, map[string]interface{}{
			"usuario_id": body["usuario_id"], "mensaje": justificacion,
		}); err != nil {
			return fmt.Errorf("la solicitud pasó a REQUIERE_INFO pero el mensaje no se pudo publicar: %v", err)
		}
	}

	// La justificación de aprobar/rechazar cierra el hilo. Inserta directo en el
	// CRUD: EnviarMensaje exige estado conversacional y aquí ya es terminal.
	if (nuevoEstado == estadoAprobada || nuevoEstado == estadoRechazada) && strings.TrimSpace(justificacion) != "" {
		payload := map[string]interface{}{
			"solicitud_beneficio": map[string]interface{}{"id": id},
			"usuario":             map[string]interface{}{"id": toInt(body["usuario_id"])},
			"mensaje":             justificacion,
		}
		var mensajeCreado interface{}
		if err := helpers.PostCRUD(token, "/mensaje-solicitud", payload, &mensajeCreado); err != nil {
			return fmt.Errorf("la solicitud pasó a %s pero el mensaje de cierre no se pudo publicar: %v", nuevoEstado, err)
		}
	}
	return nil
}

// EnviarMensaje publica un mensaje mientras la solicitud está en conversación
// (REQUIERE_INFO o EN_REVISION). Si el egresado responde estando en REQUIERE_INFO,
// la solicitud pasa automáticamente a EN_REVISION.
func EnviarMensaje(token string, solicitudId int, body map[string]interface{}) (interface{}, error) {
	estado, estadoId, err := getEstadoActual(token, solicitudId)
	if err != nil {
		return nil, err
	}
	if estado != estadoRequiereInfo && estado != estadoEnRevision {
		return nil, fmt.Errorf("solo se pueden enviar mensajes mientras la solicitud está en conversación (REQUIERE_INFO o EN_REVISION); estado actual: %s", estado)
	}

	usuarioId := toInt(body["usuario_id"])
	payload := map[string]interface{}{
		"solicitud_beneficio": map[string]interface{}{"id": solicitudId},
		"usuario":             map[string]interface{}{"id": usuarioId},
		"mensaje":             body["mensaje"],
	}
	var result interface{}
	if err := helpers.PostCRUD(token, "/mensaje-solicitud", payload, &result); err != nil {
		return nil, err
	}

	if estado == estadoRequiereInfo && esDelEgresado(token, solicitudId, usuarioId) {
		if err := registrarCambioEstado(token, solicitudId, estadoId, estadoEnRevision, usuarioId, nil, "", ""); err != nil {
			return result, fmt.Errorf("mensaje enviado, pero no se pudo pasar la solicitud a EN_REVISION: %v", err)
		}
	}
	return result, nil
}

// esDelEgresado indica si el usuario local es el dueño (egresado) de la solicitud.
// Best effort: ante cualquier duda devuelve false (no se cambia el estado).
func esDelEgresado(token string, solicitudId, usuarioId int) bool {
	if usuarioId <= 0 {
		return false
	}
	var sol map[string]interface{}
	if err := helpers.GetCRUD(token, fmt.Sprintf("/solicitud-beneficio/%d", solicitudId), &sol); err != nil {
		return false
	}
	eg, _ := sol["egresado"].(map[string]interface{})
	if eg == nil {
		return false
	}
	eid := toInt(firstOf(eg, "id", "Id"))
	if eid <= 0 {
		return false
	}
	var egresado map[string]interface{}
	if err := helpers.GetCRUD(token, fmt.Sprintf("/egresado/%d", eid), &egresado); err != nil {
		return false
	}
	u, _ := egresado["usuario"].(map[string]interface{})
	return u != nil && toInt(firstOf(u, "id", "Id")) == usuarioId
}

// GetHistorialSolicitud retorna la bitácora de estados (más reciente primero) con
// los códigos resueltos. Proyección mínima (RNF-002b): actor solo como usuario{id};
// el comprobante tiene su propio endpoint.
func GetHistorialSolicitud(token string, solicitudId int) ([]map[string]interface{}, error) {
	var filas []map[string]interface{}
	if err := helpers.GetCRUD(token, fmt.Sprintf("/historial-solicitud/solicitud/%d", solicitudId), &filas); err != nil {
		return nil, err
	}
	historial := make([]map[string]interface{}, 0, len(filas))
	for _, h := range filas {
		item := map[string]interface{}{
			"id":              h["id"],
			"estado_nuevo_id": h["estado_nuevo_id"],
			"fecha_cambio":    h["fecha_cambio"],
		}
		if codigo, err := ResolverParametroCodigo(token, TipoParamEstadoSolicitud, toInt(h["estado_nuevo_id"])); err == nil {
			item["estado_nuevo"] = codigo
		}
		if aid := toInt(h["estado_anterior_id"]); aid > 0 {
			item["estado_anterior_id"] = aid
			if codigo, err := ResolverParametroCodigo(token, TipoParamEstadoSolicitud, aid); err == nil {
				item["estado_anterior"] = codigo
			}
		}
		if j := asString(h["justificacion"]); j != "" {
			item["justificacion"] = j
		}
		if u, ok := h["usuario"].(map[string]interface{}); ok {
			item["usuario"] = map[string]interface{}{"id": toInt(firstOf(u, "id", "Id"))}
		}
		historial = append(historial, item)
	}
	return historial, nil
}

// GetMensajes retorna el historial de mensajes de una solicitud.
func GetMensajes(token string, solicitudId int) (interface{}, error) {
	var result interface{}
	query := fmt.Sprintf("/mensaje-solicitud?query=SolicitudBeneficio.Id:%d,Activo:true&sortby=FechaEnvio&order=asc&limit=0", solicitudId)
	if err := helpers.GetCRUD(token, query, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ── Helpers internos ──────────────────────────────────────────────────────────

// transicionValida verifica la máquina de estados de solicitud (RN-005).
func transicionValida(actual, nuevo string) bool {
	maquina := map[string][]string{
		estadoPendiente:    {estadoEnRevision, estadoAprobada, estadoRechazada, estadoRequiereInfo, estadoCancelada},
		estadoEnRevision:   {estadoAprobada, estadoRechazada, estadoRequiereInfo, estadoCancelada},
		estadoRequiereInfo: {estadoEnRevision, estadoAprobada, estadoRechazada, estadoCancelada},
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
func getEstadoActual(token string, solicitudId int) (codigo string, estadoId int, err error) {
	var vigente map[string]interface{}
	if err = helpers.GetCRUD(token, fmt.Sprintf("/historial-solicitud/solicitud/%d/vigente", solicitudId), &vigente); err != nil {
		return "", 0, fmt.Errorf("no se pudo obtener el estado de la solicitud %d: %v", solicitudId, err)
	}
	estadoId = toInt(vigente["estado_nuevo_id"])
	if estadoId == 0 {
		return "", 0, fmt.Errorf("la solicitud %d no tiene historial de estado", solicitudId)
	}
	codigo, err = ResolverParametroCodigo(token, TipoParamEstadoSolicitud, estadoId)
	if err != nil {
		return "", 0, err
	}
	return codigo, estadoId, nil
}

// registrarCambioEstado inserta la transición en historial_solicitud (RN-004, C-4b).
func registrarCambioEstado(token string, solicitudId, estadoAnteriorId int, nuevoEstadoCodigo string, usuarioId, justificacion interface{}, nombreComprobante, enlaceComprobante string) error {
	nuevoId, err := ResolverParametroId(token, TipoParamEstadoSolicitud, nuevoEstadoCodigo)
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
	if enlaceComprobante != "" {
		historial["nombre_archivo_comprobante"] = nombreComprobante
		historial["enlace_comprobante"] = enlaceComprobante
	}
	return helpers.PostCRUD(token, "/historial-solicitud", historial, &map[string]interface{}{})
}

// getComprobanteDeSolicitud lee el comprobante opcional adjuntado al aprobar,
// desde el registro vigente del historial. Vacíos si no hay comprobante.
func getComprobanteDeSolicitud(token string, solicitudId int) (nombreArchivo, enlace string, err error) {
	var vigente map[string]interface{}
	if err = helpers.GetCRUD(token, fmt.Sprintf("/historial-solicitud/solicitud/%d/vigente", solicitudId), &vigente); err != nil {
		return "", "", fmt.Errorf("no se pudo obtener el estado de la solicitud %d: %v", solicitudId, err)
	}
	nombreArchivo = asString(firstOf(vigente, "nombre_archivo_comprobante", "NombreArchivoComprobante"))
	enlace = asString(firstOf(vigente, "enlace_comprobante", "EnlaceComprobante"))
	return nombreArchivo, enlace, nil
}

// esEstadoNoTerminal indica si una solicitud sigue EN CURSO (cuenta para RN-007/RN-010).
// Los terminales (APROBADA, RECHAZADA, CANCELADA) no bloquean ni cuentan.
func esEstadoNoTerminal(codigo string) bool {
	switch codigo {
	case estadoPendiente, estadoEnRevision, estadoRequiereInfo:
		return true
	}
	return false
}

// beneficiosConSolicitudActiva retorna los beneficio_id de las solicitudes en curso
// del egresado; alimenta RN-007 y RN-010 con una sola consulta.
func beneficiosConSolicitudActiva(token string, egresadoId int) ([]int, error) {
	var solicitudes []map[string]interface{}
	q := fmt.Sprintf("/solicitud-beneficio?query=Egresado.Id:%d,Activo:true&limit=0", egresadoId)
	if err := helpers.GetCRUD(token, q, &solicitudes); err != nil {
		return nil, err
	}
	var beneficioIds []int
	for _, s := range solicitudes {
		codigo, _, err := getEstadoActual(token, toInt(s["id"]))
		if err != nil {
			continue
		}
		if esEstadoNoTerminal(codigo) {
			bid := 0
			if ben, ok := s["beneficio"].(map[string]interface{}); ok {
				bid = toInt(firstOf(ben, "id", "Id"))
			}
			beneficioIds = append(beneficioIds, bid)
		}
	}
	return beneficioIds, nil
}

// descontarCupo reserva un cupo del beneficio de forma atómica (RN-002b).
// Devuelve false si no había cupos disponibles.
func descontarCupo(token string, beneficioId int) (bool, error) {
	var r map[string]interface{}
	if err := helpers.PostCRUD(token, fmt.Sprintf("/beneficio/%d/cupo/descontar", beneficioId), nil, &r); err != nil {
		return false, err
	}
	return asBool(firstOf(r, "descontado", "Descontado")), nil
}

// devolverCupo devuelve un cupo al beneficio (RN-002c). Best-effort: un fallo
// no aborta la operación principal, que ya ocurrió.
func devolverCupo(token string, beneficioId int) {
	var r map[string]interface{}
	_ = helpers.PostCRUD(token, fmt.Sprintf("/beneficio/%d/cupo/devolver", beneficioId), nil, &r)
}

// getBeneficioIdDeSolicitud obtiene el id del beneficio de una solicitud (para RN-002c).
func getBeneficioIdDeSolicitud(token string, solicitudId int) (int, error) {
	var s map[string]interface{}
	if err := helpers.GetCRUD(token, fmt.Sprintf("/solicitud-beneficio/%d", solicitudId), &s); err != nil {
		return 0, err
	}
	if ben, ok := s["beneficio"].(map[string]interface{}); ok {
		return toInt(firstOf(ben, "id", "Id")), nil
	}
	return 0, fmt.Errorf("no se pudo determinar el beneficio de la solicitud %d", solicitudId)
}

// getLimiteActivas lee el límite de solicitudes activas por egresado (RN-010)
// del servicio de parámetros (tipo PARAMETRO_SISTEMA).
func getLimiteActivas(token string) (int, error) {
	params, err := GetParametrosPorTipo(token, TipoParamParametroSistema)
	if err != nil {
		return 5, nil // default si el servicio no está disponible
	}
	for _, p := range params {
		if firstOf(p, "CodigoAbreviacion", "codigo_abreviacion") == "LIMITE_SOLIC_ACTIVAS" {
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

func asString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func asBool(v interface{}) bool {
	switch b := v.(type) {
	case bool:
		return b
	case string:
		return b == "true" || b == "1"
	case float64:
		return b != 0
	}
	return false
}
