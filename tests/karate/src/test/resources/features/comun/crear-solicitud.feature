@ignore
Feature: Helper reutilizable - crear una solicitud como el egresado del seed

  Argumentos: beneficioId (obligatorio), token y egresadoId/usuarioId opcionales
  (por defecto el egresado del seed). Devuelve solicitudId y radicado.

  Scenario:
    * def token = karate.get('token', tokenEgresado)
    * def egresadoId = karate.get('egresadoId', egresadoSeedId)
    * def usuarioId = karate.get('usuarioId', usuarioEgresadoSeedId)
    Given url midUrl
    And path 'solicitudes'
    And header Authorization = token
    And request { egresado_id: '#(egresadoId)', beneficio_id: '#(beneficioId)', usuario_id: '#(usuarioId)', datos_complementarios: 'Solicitud generada por la suite Karate' }
    When method post
    Then status 201
    And match response.Body.radicado == '#regex BNF-\\d{4}-\\d{6}'
    * def solicitudId = response.Body.id
    * def radicado = response.Body.radicado
