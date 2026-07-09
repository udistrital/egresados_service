Feature: JIT provisioning de egresados y empresas (C-2a / C-2b/c)

  El alta local se deriva 100% del token (OIDC userinfo → userRol → fuentes
  institucionales): nadie puede provisionar la identidad de otro. Debe ser
  idempotente (relogin no duplica) y respetar la minimización de datos
  (RNF-002b): nunca re-exponer datos bancarios/documentos de Ágora.

  Background:
    * url midUrl

  Scenario: JIT de egresado ya sembrado es idempotente (encuentra, no duplica)
    Given path 'egresados/provision'
    And header Authorization = tokenEgresado
    When method post
    Then status 200
    And match response.Body == { usuario_id: '#(usuarioEgresadoSeedId)', egresado_id: '#(egresadoSeedId)', codigo_institucional: '#string', nombre: '#string' }

    # Segunda llamada: mismos ids, nada nuevo
    Given path 'egresados/provision'
    And header Authorization = tokenEgresado
    When method post
    Then status 200
    And match response.Body.usuario_id == usuarioEgresadoSeedId
    And match response.Body.egresado_id == egresadoSeedId

  Scenario: JIT de egresado NUEVO crea usuario y egresado con la identidad del token
    Given path 'egresados/provision'
    And header Authorization = tokenEgresadoNuevo
    When method post
    Then status 200
    And match response.Body.usuario_id == '#? _ > 0'
    And match response.Body.egresado_id == '#? _ > 0'
    # Código institucional desde userRol y nombre real desde terceros_crud (mock)
    And match response.Body.codigo_institucional == '20241020999'
    And match response.Body.nombre == 'NUEVA EGRESADA DEMO'
    * def usuarioCreado = response.Body.usuario_id
    * def egresadoCreado = response.Body.egresado_id

    # Idempotencia: el relogin devuelve exactamente los mismos ids
    Given path 'egresados/provision'
    And header Authorization = tokenEgresadoNuevo
    When method post
    Then status 200
    And match response.Body.usuario_id == usuarioCreado
    And match response.Body.egresado_id == egresadoCreado

  Scenario: JIT de egresado rechaza a un usuario de empresa (Estado != 'E')
    Given path 'egresados/provision'
    And header Authorization = tokenEmpresa
    When method post
    Then status 422
    And match response.Success == false

  Scenario: JIT de empresa con caso 1:N (un correo, DOS proveedores en Ágora)
    Given path 'empresas/provision'
    And header Authorization = tokenEmpresaNueva
    When method post
    Then status 200
    And match response.Body.usuario_id == '#? _ > 0'
    And match response.Body.empresas == '#[2]'
    # RNF-002b: el proveedor expuesto es EXACTAMENTE la whitelist pública —
    # sin NumDocumento, sin datos bancarios, sin anexos (el mock SÍ los envía;
    # el MID debe descartarlos).
    And match each response.Body.empresas[*].proveedor == { agora_id_externo: '#number', razon_social: '#string', tipo_persona: '#string', correo: '#string' }
    And match karate.jsonPath(response, '$.Body.empresas[*].proveedor.agora_id_externo') contains [90001, 90002]
    And match each response.Body.empresas == { empresa_id: '#number', usuario_empresa_id: '#number', nit: '#string', proveedor: '#object' }
    * def usuarioEmpresaNueva = response.Body.usuario_id
    * def empresasCreadas = karate.jsonPath(response, '$.Body.empresas[*].empresa_id')

    # Idempotencia: relogin no duplica empresas ni vínculos
    Given path 'empresas/provision'
    And header Authorization = tokenEmpresaNueva
    When method post
    Then status 200
    And match response.Body.usuario_id == usuarioEmpresaNueva
    And match karate.jsonPath(response, '$.Body.empresas[*].empresa_id') == empresasCreadas

    # El selector multiempresa (paso 5) devuelve los 2 vínculos, sin datos sensibles
    Given path 'usuarios', usuarioEmpresaNueva, 'empresas'
    And header Authorization = tokenEmpresaNueva
    When method get
    Then status 200
    And match response.Body == '#[2]'
    And match each response.Body == { empresa_id: '#number', usuario_empresa_id: '#number', agora_id_externo: '##string', razon_social: '#string', estado_empresa_id: '#number', es_principal: '#boolean', cargo: '##string' }

  Scenario: JIT de empresa rechaza a un egresado (Estado == 'E')
    Given path 'empresas/provision'
    And header Authorization = tokenEgresado
    When method post
    Then status 422
    And match response.Success == false

  Scenario: JIT de empresa sin proveedores en Ágora responde 422
    Given path 'empresas/provision'
    And header Authorization = tokenIntruso
    When method post
    Then status 422
    And match response.Message contains 'ninguna empresa'
