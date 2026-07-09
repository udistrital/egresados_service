Feature: Cancelación (RN-002c) y rechazo (RN-003) de solicitudes

  Al cancelar o rechazar, el cupo vuelve al pool (RN-002c). El rechazo sin
  justificación registra el texto institucional "sin perjuicio" y lo publica
  como mensaje de cierre del hilo (el egresado no ve el historial).

  Background:
    * url midUrl
    * def creado = call read('comun/publicar-beneficio.feature') { cupos: 2 }
    * def beneficioId = creado.beneficioId

    * def crearSolicitud =
      """
      function() {
        var res = karate.call('classpath:features/comun/crear-solicitud.feature', { beneficioId: beneficioId });
        return res.solicitudId;
      }
      """

  Scenario: Cancelar devuelve el cupo y es terminal
    * def solicitudId = crearSolicitud()

    # Cupo reservado al crear: 2 → 1
    Given path 'beneficios', beneficioId
    And header Authorization = tokenEgresado
    When method get
    Then status 200
    And match response.Body.cupos_disponibles == 1

    # Un intruso NO puede cancelar la solicitud de otro (anti-IDOR)
    Given path 'solicitudes', solicitudId, 'cancelar'
    And header Authorization = tokenIntruso
    And request { usuario_id: 999 }
    When method put
    Then status 403

    # El dueño sí puede cancelar
    Given path 'solicitudes', solicitudId, 'cancelar'
    And header Authorization = tokenEgresado
    And request { usuario_id: '#(usuarioEgresadoSeedId)' }
    When method put
    Then status 200

    # RN-002c: el cupo volvió al pool (1 → 2)
    Given path 'beneficios', beneficioId
    And header Authorization = tokenEgresado
    When method get
    Then status 200
    And match response.Body.cupos_disponibles == 2

    Given path 'solicitudes/egresado', egresadoSeedId
    And header Authorization = tokenEgresado
    When method get
    Then status 200
    * def mia = karate.jsonPath(response, "$.Body[?(@.id == " + solicitudId + ")]")[0]
    And match mia.estado_solicitud == 'CANCELADA'

    # RN-005: cancelar una solicitud ya terminal responde 422
    Given path 'solicitudes', solicitudId, 'cancelar'
    And header Authorization = tokenEgresado
    And request { usuario_id: '#(usuarioEgresadoSeedId)' }
    When method put
    Then status 422

  Scenario: El rechazo devuelve el cupo y publica el mensaje de cierre
    * def solicitudId = crearSolicitud()

    # La empresa rechaza SIN justificación → texto institucional "sin perjuicio"
    Given path 'solicitudes', solicitudId, 'responder'
    And header Authorization = tokenEmpresa
    And request { estado_nuevo: 'RECHAZADA', usuario_id: '#(usuarioEmpresaSeedId)' }
    When method put
    Then status 200

    # RN-002c: cupo devuelto también al rechazar
    Given path 'beneficios', beneficioId
    And header Authorization = tokenEgresado
    When method get
    Then status 200
    And match response.Body.cupos_disponibles == 2

    Given path 'solicitudes/egresado', egresadoSeedId
    And header Authorization = tokenEgresado
    When method get
    Then status 200
    * def mia = karate.jsonPath(response, "$.Body[?(@.id == " + solicitudId + ")]")[0]
    And match mia.estado_solicitud == 'RECHAZADA'

    # El mensaje de cierre institucional quedó publicado en el hilo
    Given path 'solicitudes', solicitudId, 'mensajes'
    And header Authorization = tokenEgresado
    When method get
    Then status 200
    * def mensajes = karate.jsonPath(response, '$.Body[*].mensaje')
    And match mensajes == '#[1]'
    And match mensajes[0] contains 'sin perjuicio'
