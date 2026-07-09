Feature: Gestión de beneficios por la empresa (RF-005) y catálogo público (RF-002/003)

  Publicar (RN-008b), editar, retirar, vista de gestión del dueño y perfil
  público de la empresa. La empresa 1 del seed está ACTIVA (gate de publicar).

  Background:
    * url midUrl

  Scenario: Publicar un beneficio y verlo en catálogo y detalle
    * def titulo = 'Descuento Karate ' + java.lang.System.currentTimeMillis()
    Given path 'empresas', empresaSeedId, 'beneficios'
    And header Authorization = tokenEmpresa
    And request
      """
      {
        titulo: '#(titulo)',
        descripcion: 'Descuento en cursos de extensión',
        condiciones: 'Presentar carné de egresado',
        categoria_beneficio_id: 7212,
        fecha_inicio: '#(hoy)',
        fecha_fin: '#(finVigencia)',
        cupos_total: 3,
        usuario_creador_id: '#(usuarioEmpresaSeedId)',
        documentos_requeridos: [ { nombre: 'Certificado laboral', descripcion: 'No mayor a 30 días' } ]
      }
      """
    When method post
    Then status 201
    * def beneficioId = response.Body.id

    # Detalle (RF-003): nace PUBLICADO, cupos_disponibles = cupos_total, con social proof
    Given path 'beneficios', beneficioId
    And header Authorization = tokenEgresado
    When method get
    Then status 200
    And match response.Body contains { titulo: '#(titulo)', cupos_total: 3, cupos_disponibles: 3, estado_beneficio_id: 7202, total_solicitudes: 0 }
    And match response.Body.documentos_requeridos == '#[1]'
    And match response.Body.documentos_requeridos[0].nombre == 'Certificado laboral'

    # Catálogo (RF-002): lo lista para el egresado
    Given path 'beneficios'
    And param limit = 100
    And header Authorization = tokenEgresado
    When method get
    Then status 200
    And match response.Body[*].id contains beneficioId

    # Filtro de búsqueda por título
    Given path 'beneficios'
    And param q = titulo
    And header Authorization = tokenEgresado
    When method get
    Then status 200
    And match response.Body == '#[1]'
    And match response.Body[0].id == beneficioId

  Scenario: RN-008b - publicar sin un campo obligatorio responde 422
    Given path 'empresas', empresaSeedId, 'beneficios'
    And header Authorization = tokenEmpresa
    And request { descripcion: 'sin título', condiciones: 'x', categoria_beneficio_id: 7212, fecha_inicio: '#(hoy)', fecha_fin: '#(finVigencia)', cupos_total: 1 }
    When method post
    Then status 422
    And match response.Message contains 'titulo'

  Scenario: Vista de gestión del dueño con estado y métricas
    * def creado = call read('comun/publicar-beneficio.feature')
    Given path 'empresas', empresaSeedId, 'beneficios'
    And header Authorization = tokenEmpresa
    When method get
    Then status 200
    And match response.Body[*].id contains creado.beneficioId
    * def item = karate.jsonPath(response, "$.Body[?(@.id == " + creado.beneficioId + ")]")[0]
    And match item contains { estado_beneficio: 'PUBLICADO', total_solicitudes: 0, solicitudes_pendientes: 0 }

  Scenario: Editar un beneficio publicado sin solicitudes en curso (RN-008b edición)
    * def creado = call read('comun/publicar-beneficio.feature')
    Given path 'beneficios', creado.beneficioId
    And header Authorization = tokenEmpresa
    And request { titulo: 'Título editado por Karate', cupos_total: 5 }
    When method put
    Then status 200

    Given path 'beneficios', creado.beneficioId
    And header Authorization = tokenEgresado
    When method get
    Then status 200
    # Al subir cupos_total de 3 a 5, cupos_disponibles se mueve con el delta
    And match response.Body contains { titulo: 'Título editado por Karate', cupos_total: 5, cupos_disponibles: 5 }

  Scenario: Retirar un beneficio lo saca del catálogo pero no de la vista del dueño
    * def creado = call read('comun/publicar-beneficio.feature')
    Given path 'beneficios', creado.beneficioId, 'retirar'
    And header Authorization = tokenEmpresa
    And request {}
    When method put
    Then status 200

    Given path 'beneficios'
    And param limit = 1000
    And header Authorization = tokenEgresado
    When method get
    Then status 200
    And match response.Body[*].id !contains creado.beneficioId

    Given path 'empresas', empresaSeedId, 'beneficios'
    And header Authorization = tokenEmpresa
    When method get
    Then status 200
    * def item = karate.jsonPath(response, "$.Body[?(@.id == " + creado.beneficioId + ")]")[0]
    And match item.estado_beneficio == 'RETIRADO'

  Scenario: Perfil público de la empresa (local + enriquecimiento Ágora on-demand)
    Given path 'empresas', empresaSeedId
    And header Authorization = tokenEgresado
    When method get
    Then status 200
    And match response.Body contains { empresa_id: '#(empresaSeedId)', razon_social: 'Empresa Demo S.A.S.', estado_empresa_id: 7199 }
    # Datos que solo existen en Ágora (mock): descripción y fecha de registro
    And match response.Body.descripcion == 'Empresa aliada de demostración para pruebas'
    And match response.Body.aliado_desde == '2025-01-15'
    # RNF-002b: el perfil público NUNCA expone NIT ni datos bancarios
    And match response.Body.nit == '#notpresent'

  Scenario: Detalle de un beneficio inexistente responde 404
    Given path 'beneficios', 999999
    And header Authorization = tokenEgresado
    When method get
    Then status 404
