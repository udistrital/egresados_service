@ignore
Feature: Helper reutilizable - publicar un beneficio como la empresa del seed

  Se invoca con karate.call / call read(...) pasando argumentos opcionales:
    titulo (string), cupos (int), documentos (array de {nombre, descripcion})
  Devuelve beneficioId.

  Scenario:
    * def titulo = karate.get('titulo', 'Beneficio de prueba Karate ' + java.lang.System.currentTimeMillis())
    * def cupos = karate.get('cupos', 3)
    * def documentos = karate.get('documentos', [])
    Given url midUrl
    And path 'empresas', empresaSeedId, 'beneficios'
    And header Authorization = tokenEmpresa
    And request
      """
      {
        titulo: '#(titulo)',
        descripcion: 'Descripción generada por la suite Karate',
        condiciones: 'Condiciones generadas por la suite Karate',
        categoria_beneficio_id: 7212,
        fecha_inicio: '#(hoy)',
        fecha_fin: '#(finVigencia)',
        cupos_total: '#(cupos)',
        usuario_creador_id: '#(usuarioEmpresaSeedId)',
        documentos_requeridos: '#(documentos)'
      }
      """
    When method post
    Then status 201
    And match response.Body.id == '#? _ > 0'
    * def beneficioId = response.Body.id
