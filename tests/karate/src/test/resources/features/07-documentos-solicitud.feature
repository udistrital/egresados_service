Feature: Documentos de la solicitud (gestor documental, IdTipoDocumento=167)

  El egresado sube PDFs que cumplen los documentos requeridos del beneficio;
  la empresa los revisa y comenta. El MID es proxy del gestor documental
  (mockeado): el cliente nunca habla con Nuxeo directamente. Solo se aceptan
  PDFs (validación del magic number %PDF).

  Background:
    * url midUrl
    * def documentos = [ { nombre: 'Certificado laboral', descripcion: 'No mayor a 30 días' }, { nombre: 'Copia del diploma', descripcion: '' } ]
    * def creado = call read('comun/publicar-beneficio.feature') { cupos: 2, documentos: '#(documentos)' }
    * def beneficioId = creado.beneficioId
    * def sol = call read('comun/crear-solicitud.feature') { beneficioId: '#(beneficioId)' }
    * def solicitudId = sol.solicitudId

  Scenario: Subir, revisar, comentar y eliminar documentos
    # Documentos requeridos del beneficio (vista previa del egresado, RF-003)
    Given path 'beneficios', beneficioId, 'documentos-requeridos'
    And header Authorization = tokenEgresado
    When method get
    Then status 200
    And match response.Body == '#[2]'
    * def docRequeridoId = response.Body[0].id

    # Vista combinada: nada subido aún
    Given path 'solicitudes', solicitudId, 'documentos'
    And header Authorization = tokenEgresado
    When method get
    Then status 200
    And match response.Body == '#[2]'
    And match each response.Body contains { subido: false }

    # Un archivo que NO es PDF se rechaza (magic number)
    Given path 'solicitudes', solicitudId, 'documentos'
    And header Authorization = tokenEgresado
    And request { documento_requerido_id: '#(docRequeridoId)', nombre_archivo: 'certificado.txt', file: '#(noEsPdfBase64)' }
    When method post
    Then status 422
    And match response.Message contains 'PDF'

    # Subida válida
    Given path 'solicitudes', solicitudId, 'documentos'
    And header Authorization = tokenEgresado
    And request { documento_requerido_id: '#(docRequeridoId)', nombre_archivo: 'certificado.pdf', file: '#(pdfBase64)' }
    When method post
    Then status 201

    # La vista combinada refleja la subida
    Given path 'solicitudes', solicitudId, 'documentos'
    And header Authorization = tokenEgresado
    When method get
    Then status 200
    * def item = karate.jsonPath(response, "$.Body[?(@.documento_requerido_id == " + docRequeridoId + ")]")[0]
    And match item contains { subido: true, nombre_archivo: 'certificado.pdf' }
    * def documentoSolicitudId = item.documento_solicitud_id

    # La empresa (participante) descarga el archivo vía el proxy del MID
    Given path 'documentos', documentoSolicitudId, 'archivo'
    And header Authorization = tokenEmpresa
    When method get
    Then status 200
    And match response.Body == { nombre_archivo: 'certificado.pdf', file: '#(pdfBase64)' }

    # Un intruso NO puede descargarlo (anti-IDOR bidireccional)
    Given path 'documentos', documentoSolicitudId, 'archivo'
    And header Authorization = tokenIntruso
    When method get
    Then status 403

    # La empresa comenta el documento; el egresado NO puede usar ese endpoint
    Given path 'documentos', documentoSolicitudId, 'comentario'
    And header Authorization = tokenEgresado
    And request { comentario: 'intento indebido' }
    When method put
    Then status 403

    Given path 'documentos', documentoSolicitudId, 'comentario'
    And header Authorization = tokenEmpresa
    And request { comentario: 'El certificado está ilegible, súbelo de nuevo' }
    When method put
    Then status 200

    Given path 'solicitudes', solicitudId, 'documentos'
    And header Authorization = tokenEmpresa
    When method get
    Then status 200
    * def item = karate.jsonPath(response, "$.Body[?(@.documento_requerido_id == " + docRequeridoId + ")]")[0]
    And match item.comentario_empresa == 'El certificado está ilegible, súbelo de nuevo'

    # El egresado elimina el documento (borrado lógico + gestor documental)
    Given path 'solicitudes', solicitudId, 'documentos', documentoSolicitudId
    And header Authorization = tokenEgresado
    When method delete
    Then status 200

    Given path 'solicitudes', solicitudId, 'documentos'
    And header Authorization = tokenEgresado
    When method get
    Then status 200
    * def item = karate.jsonPath(response, "$.Body[?(@.documento_requerido_id == " + docRequeridoId + ")]")[0]
    And match item.subido == false
