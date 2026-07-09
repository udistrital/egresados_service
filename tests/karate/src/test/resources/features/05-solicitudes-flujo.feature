Feature: Ciclo de vida completo de una solicitud (RF-003/006/007/008/013)

  Flujo feliz end-to-end con la máquina de estados RN-005 (modelo "de quién es
  la pelota"): PENDIENTE → REQUIERE_INFO (empresa pregunta) → EN_REVISION
  (ping-pong: el egresado responde) → APROBADA (con comprobante). Además:
  radicado BNF-YYYY-NNNNNN (C-5), cupo atómico RN-002b, RN-007, historial C-4b,
  minimización de datos en la bandeja (RNF-002b) y resumen RF-013.

  Background:
    * url midUrl
    * def creado = call read('comun/publicar-beneficio.feature') { cupos: 3 }
    * def beneficioId = creado.beneficioId

  Scenario: Solicitar, conversar, aprobar
    # ── El egresado crea la solicitud ────────────────────────────────────────
    Given path 'solicitudes'
    And header Authorization = tokenEgresado
    And request { egresado_id: '#(egresadoSeedId)', beneficio_id: '#(beneficioId)', usuario_id: '#(usuarioEgresadoSeedId)', datos_complementarios: 'Me interesa para actualizarme profesionalmente' }
    When method post
    Then status 201
    And match response.Body.radicado == '#regex BNF-\\d{4}-\\d{6}'
    * def solicitudId = response.Body.id

    # RN-002b: el cupo se descontó atómicamente al crear
    Given path 'beneficios', beneficioId
    And header Authorization = tokenEgresado
    When method get
    Then status 200
    And match response.Body.cupos_disponibles == 2
    And match response.Body.total_solicitudes == 1

    # Mis solicitudes: estado vigente PENDIENTE derivado del historial (C-4b)
    Given path 'solicitudes/egresado', egresadoSeedId
    And header Authorization = tokenEgresado
    When method get
    Then status 200
    * def mia = karate.jsonPath(response, "$.Body[?(@.id == " + solicitudId + ")]")[0]
    And match mia.estado_solicitud == 'PENDIENTE'
    And match mia.datos_complementarios == 'Me interesa para actualizarme profesionalmente'

    # RN-007: no se puede tener DOS solicitudes en curso del mismo beneficio
    Given path 'solicitudes'
    And header Authorization = tokenEgresado
    And request { egresado_id: '#(egresadoSeedId)', beneficio_id: '#(beneficioId)', usuario_id: '#(usuarioEgresadoSeedId)' }
    When method post
    Then status 422
    And match response.Message contains 'en curso'

    # ── Bandeja de la empresa (RNF-002b: datos mínimos del egresado) ─────────
    Given path 'empresas', empresaSeedId, 'solicitudes'
    And header Authorization = tokenEmpresa
    When method get
    Then status 200
    * def item = karate.jsonPath(response, "$.Body[?(@.id == " + solicitudId + ")]")[0]
    And match item.estado_solicitud == 'PENDIENTE'
    And match item.egresado == { nombre: '#string', codigo_institucional: '#string' }
    And match item.beneficio.id == beneficioId

    # ── La empresa pide información (PENDIENTE → REQUIERE_INFO) ──────────────
    Given path 'solicitudes', solicitudId, 'responder'
    And header Authorization = tokenEmpresa
    And request { estado_nuevo: 'REQUIERE_INFO', justificacion: 'Adjunta el certificado laboral actualizado', usuario_id: '#(usuarioEmpresaSeedId)' }
    When method put
    Then status 200

    # La nota llega al egresado como MENSAJE del hilo (no como historial)
    Given path 'solicitudes', solicitudId, 'mensajes'
    And header Authorization = tokenEgresado
    When method get
    Then status 200
    And match response.Body[*].mensaje contains 'Adjunta el certificado laboral actualizado'

    # Un tercero autenticado NO puede leer el hilo (anti-IDOR bidireccional)
    Given path 'solicitudes', solicitudId, 'mensajes'
    And header Authorization = tokenIntruso
    When method get
    Then status 403

    # ── Ping-pong: el egresado responde → auto-transición a EN_REVISION ──────
    Given path 'solicitudes', solicitudId, 'mensajes'
    And header Authorization = tokenEgresado
    And request { usuario_id: '#(usuarioEgresadoSeedId)', mensaje: 'Listo, aquí está el certificado' }
    When method post
    Then status 201

    Given path 'solicitudes/egresado', egresadoSeedId
    And header Authorization = tokenEgresado
    When method get
    Then status 200
    * def mia = karate.jsonPath(response, "$.Body[?(@.id == " + solicitudId + ")]")[0]
    And match mia.estado_solicitud == 'EN_REVISION'

    # ── La empresa aprueba con comprobante PDF (sube al gestor documental) ───
    Given path 'solicitudes', solicitudId, 'responder'
    And header Authorization = tokenEmpresa
    And request { estado_nuevo: 'APROBADA', justificacion: 'Felicitaciones, beneficio otorgado', usuario_id: '#(usuarioEmpresaSeedId)', comprobante: { nombre_archivo: 'comprobante.pdf', file: '#(pdfBase64)' } }
    When method put
    Then status 200

    Given path 'solicitudes/egresado', egresadoSeedId
    And header Authorization = tokenEgresado
    When method get
    Then status 200
    * def mia = karate.jsonPath(response, "$.Body[?(@.id == " + solicitudId + ")]")[0]
    And match mia.estado_solicitud == 'APROBADA'

    # El egresado descarga el comprobante (proxy del gestor documental)
    Given path 'solicitudes', solicitudId, 'comprobante'
    And header Authorization = tokenEgresado
    When method get
    Then status 200
    And match response.Body == { tiene_comprobante: true, nombre_archivo: 'comprobante.pdf', file: '#(pdfBase64)' }

    # Historial C-4b visible para ambas partes: nacimiento (PENDIENTE) +
    # REQUIERE_INFO + EN_REVISION (ping-pong) + APROBADA = 4 registros
    Given path 'solicitudes', solicitudId, 'historial'
    And header Authorization = tokenEmpresa
    When method get
    Then status 200
    And match response.Body == '#[4]'

    # Resumen RF-013 del egresado: al menos esta aprobada
    Given path 'solicitudes/egresado', egresadoSeedId, 'resumen'
    And header Authorization = tokenEgresado
    When method get
    Then status 200
    And match response.Body == { activas: '#number', aprobadas: '#? _ >= 1', rechazadas: '#number', canceladas: '#number' }

    # RN-005: una solicitud APROBADA es terminal — re-aprobar es transición inválida
    Given path 'solicitudes', solicitudId, 'responder'
    And header Authorization = tokenEmpresa
    And request { estado_nuevo: 'APROBADA', usuario_id: '#(usuarioEmpresaSeedId)' }
    When method put
    Then status 422
    And match response.Message contains 'inválida'
