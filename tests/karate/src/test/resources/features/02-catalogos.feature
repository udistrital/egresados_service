Feature: Catálogos de parámetros institucionales (C-1)

  Los catálogos viven en el servicio institucional de parámetros; el MID los
  expone como solo-lectura. En la corrida local se usa la semilla en memoria
  (BENEFICIOS_EGRESADOS_MID_PARAMETROS_LOCAL=true), que espeja los ids REALES
  del servicio institucional (7199+, creados el 2026-07-07).

  Background:
    * url midUrl

  Scenario: Categorías de beneficio activas
    Given path 'categorias-beneficio'
    And header Authorization = tokenEgresado
    When method get
    Then status 200
    And match response.Success == true
    And match response.Body == '#[_ > 0]'
    And match response.Body[*].CodigoAbreviacion contains ['EDUCACION', 'SALUD', 'DESCUENTOS']
    And match each response.Body == { Id: '#number', CodigoAbreviacion: '#string', Nombre: '#string' }

  Scenario: Sectores económicos activos
    Given path 'sectores-economicos'
    And header Authorization = tokenEgresado
    When method get
    Then status 200
    And match response.Success == true
    And match response.Body[*].CodigoAbreviacion contains 'TEC'
