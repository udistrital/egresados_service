Feature: RN-010 - límite de solicitudes activas por egresado

  Un egresado no puede tener más de N solicitudes EN CURSO (default 5,
  parámetro LIMITE_SOLIC_ACTIVAS). Las terminales (aprobada/rechazada/
  cancelada) no cuentan. Se usa el egresado NUEVO (provisionado por JIT en
  esta misma corrida) para que el conteo arranque en cero.

  Background:
    * url midUrl

  Scenario: La sexta solicitud activa se rechaza con 422
    # Identidad limpia: el JIT garantiza el egresado (idempotente si el feature
    # 03 ya lo creó)
    Given path 'egresados/provision'
    And header Authorization = tokenEgresadoNuevo
    When method post
    Then status 200
    * def egresadoId = response.Body.egresado_id
    * def usuarioId = response.Body.usuario_id

    # Este egresado no debe tener solicitudes activas previas en la corrida
    Given path 'solicitudes/egresado', egresadoId, 'resumen'
    And header Authorization = tokenEgresadoNuevo
    When method get
    Then status 200
    And match response.Body.activas == 0

    # 6 beneficios y 5 solicitudes (el límite exacto)
    * def publicar = function(i){ return karate.call('classpath:features/comun/publicar-beneficio.feature', { titulo: 'RN-010 beneficio ' + i + ' - ' + java.lang.System.currentTimeMillis(), cupos: 1 }).beneficioId }
    * def beneficios = karate.repeat(6, publicar)
    * def solicitar = function(i){ return karate.call('classpath:features/comun/crear-solicitud.feature', { beneficioId: beneficios[i], token: tokenEgresadoNuevo, egresadoId: egresadoId, usuarioId: usuarioId }).solicitudId }
    * def solicitudes = karate.repeat(5, solicitar)

    Given path 'solicitudes/egresado', egresadoId, 'resumen'
    And header Authorization = tokenEgresadoNuevo
    When method get
    Then status 200
    And match response.Body.activas == 5

    # La sexta (beneficio distinto, así que RN-007 no aplica) choca con RN-010
    Given path 'solicitudes'
    And header Authorization = tokenEgresadoNuevo
    And request { egresado_id: '#(egresadoId)', beneficio_id: '#(beneficios[5])', usuario_id: '#(usuarioId)' }
    When method post
    Then status 422
    And match response.Message contains 'límite'

    # RN-002b: el cupo del sexto beneficio NO quedó descontado (la validación
    # RN-010 corre ANTES de reservar cupo)
    Given path 'beneficios', beneficios[5]
    And header Authorization = tokenEgresadoNuevo
    When method get
    Then status 200
    And match response.Body.cupos_disponibles == 1

    # Una terminal deja de contar: al cancelar una, la sexta ya es posible
    Given path 'solicitudes', solicitudes[0], 'cancelar'
    And header Authorization = tokenEgresadoNuevo
    And request { usuario_id: '#(usuarioId)' }
    When method put
    Then status 200

    * call read('comun/crear-solicitud.feature') { beneficioId: '#(beneficios[5])', token: '#(tokenEgresadoNuevo)', egresadoId: '#(egresadoId)', usuarioId: '#(usuarioId)' }

    Given path 'solicitudes/egresado', egresadoId, 'resumen'
    And header Authorization = tokenEgresadoNuevo
    When method get
    Then status 200
    And match response.Body.activas == 5
    And match response.Body.canceladas == 1
