Feature: Seguridad - autenticación (middleware JWT) y autorización anti-IDOR

  Valida el middleware de token entrante (middleware/jwt.go) y las verificaciones
  de autorización (services/autorizacion_service.go): un usuario autenticado NO
  puede operar recursos de otro (403), y sin token válido nada responde (401).

  Background:
    * url midUrl

  Scenario: Sin header Authorization responde 401 con envelope OATI
    Given path 'beneficios'
    When method get
    Then status 401
    And match response == { Status: '401', Success: false, Message: '#string' }

  Scenario: Token desconocido (rechazado por userinfo) responde 401
    Given path 'beneficios'
    And header Authorization = 'Bearer token-que-no-existe'
    When method get
    Then status 401
    And match response.Success == false

  Scenario: Con token válido el catálogo responde 200
    Given path 'beneficios'
    And header Authorization = tokenEgresado
    When method get
    Then status 200
    And match response.Success == true

  Scenario: Anti-IDOR - un autenticado SIN provisionar no puede leer solicitudes ajenas
    Given path 'solicitudes/egresado', egresadoSeedId
    And header Authorization = tokenIntruso
    When method get
    Then status 403
    And match response == { Status: '403', Success: false, Message: '#string' }

  Scenario: Anti-IDOR - un egresado no puede ver la bandeja de una empresa
    Given path 'empresas', empresaSeedId, 'solicitudes'
    And header Authorization = tokenEgresado
    When method get
    Then status 403

  Scenario: Anti-IDOR - un usuario de empresa no puede leer las solicitudes de un egresado
    Given path 'solicitudes/egresado', egresadoSeedId
    And header Authorization = tokenEmpresa
    When method get
    Then status 403

  Scenario: Anti-IDOR - el selector de empresas exige que el usuario_id sea el del token
    Given path 'usuarios', usuarioEmpresaSeedId, 'empresas'
    And header Authorization = tokenEgresado
    When method get
    Then status 403

  Scenario: Anti-IDOR - no se puede crear una solicitud a nombre de otro egresado
    Given path 'solicitudes'
    And header Authorization = tokenEmpresa
    And request { egresado_id: '#(egresadoSeedId)', beneficio_id: 1 }
    When method post
    Then status 403

  Scenario: Anti-IDOR - un egresado no puede publicar beneficios por una empresa
    Given path 'empresas', empresaSeedId, 'beneficios'
    And header Authorization = tokenEgresado
    And request { titulo: 'x', descripcion: 'x', condiciones: 'x', categoria_beneficio_id: 7212, fecha_inicio: '#(hoy)', fecha_fin: '#(finVigencia)', cupos_total: 1 }
    When method post
    Then status 403
