Feature: Mock de los servicios institucionales (WSO2 / gateway OAS)

  Simula, con los CONTRATOS REALES verificados contra producción (ver CLAUDE.md
  del proyecto, cortes 2026-07-01/02), los servicios externos que el MID consume:

    - GET  /oauth2/userinfo                                  (OIDC, identidad del token)
    - POST /autenticacion_mid/v1/token/userRol               (identidad por email)
    - GET  /administrativa_amazon_api/v1/informacion_proveedor (proveedores Ágora)
    - GET  /terceros_crud/v1/datos_identificacion            (nombre real por documento)
    - GET  /sga_mid/v1/derechos_pecuniarios/consultar_persona (código institucional)
    - POST/GET/DELETE /gestor_documental_mid/v1/document/... (PDFs en Nuxeo)

  Así la suite corre 100% OFFLINE y repetible: no depende de tokens vivos (~1h)
  ni de conectividad con la red de la UD. Los tokens ficticios son opacos, por lo
  que el middleware del MID los valida contra este mismo mock (rama validarOpaco).

Background:
  # ── Identidades por token (lo que devolvería userinfo) ──────────────────────
  # token-egresado / token-empresa calzan con el seed del CRUD (id_externo SGA-2
  # y AGORA-1). Los usuarios de empresa NO traen documento (hallazgo 2026-07-01).
  * def identidades =
    """
    {
      'token-egresado':       { sub: 'SGA-2',          email: 'egresado@correo.udistrital.edu.co',       documento: '1016060113', documento_compuesto: 'CC 1016060113' },
      'token-egresado-nuevo': { sub: 'WSO2-EGR-NUEVO', email: 'nuevo.egresado@correo.udistrital.edu.co', documento: '1234509876', documento_compuesto: 'CC 1234509876' },
      'token-empresa':        { sub: 'AGORA-1',        email: 'rep@empresademo.com' },
      'token-empresa-nueva':  { sub: 'WSO2-EMP-NUEVA', email: 'gerencia@innovatech.example.com' },
      'token-intruso':        { sub: 'WSO2-INTRUSO',   email: 'intruso@example.com' }
    }
    """

  # ── userRol por email (Estado 'E' = egresado; vacío = empresa, contrato real) ─
  * def userRoles =
    """
    {
      'egresado@correo.udistrital.edu.co':       { role: ['Internal/everyone'], documento: '1016060113', documento_compuesto: 'CC 1016060113', email: 'egresado@correo.udistrital.edu.co', FamilyName: 'Egresado Demo', Codigo: '20201020113', Estado: 'E' },
      'nuevo.egresado@correo.udistrital.edu.co': { role: ['Internal/everyone'], documento: '1234509876', documento_compuesto: 'CC 1234509876', email: 'nuevo.egresado@correo.udistrital.edu.co', FamilyName: 'Nueva Egresada Demo', Codigo: '20241020999', Estado: 'E' },
      'rep@empresademo.com':                     { role: ['Internal/everyone','Internal/selfsignup'], documento: '', documento_compuesto: '', email: 'rep@empresademo.com', FamilyName: '', Codigo: '', Estado: '' },
      'gerencia@innovatech.example.com':         { role: ['Internal/everyone','Internal/selfsignup'], documento: '', documento_compuesto: '', email: 'gerencia@innovatech.example.com', FamilyName: '', Codigo: '', Estado: '' },
      'intruso@example.com':                     { role: ['Internal/everyone','Internal/selfsignup'], documento: '', documento_compuesto: '', email: 'intruso@example.com', FamilyName: '', Codigo: '', Estado: '' }
    }
    """

  # ── Proveedores Ágora por correo (respuesta = ARRAY crudo, caso 1:N real) ────
  # Se incluyen a propósito los campos SENSIBLES (cuenta bancaria, anexos) que el
  # servicio real expone: la suite verifica que el MID NUNCA los re-exponga
  # (proyección mínima de ProveedorAgora + whitelist ProveedorPublico, RNF-002b).
  * def proveedoresPorCorreo =
    """
    {
      'gerencia@innovatech.example.com': [
        { Id: 90001, Tipopersona: 'JURIDICA', NumDocumento: '901234567',  Correo: 'gerencia@innovatech.example.com', NomProveedor: 'INNOVATECH S.A.S.',        Direccion: 'Cra 7 # 40-53, Bogotá', Web: 'https://innovatech.example.com', Descripcion: 'Servicios de tecnología', Estado: { Id: 1 }, FechaRegistro: '2025-03-10 - 09:15:00 AM', TipoCuentaBancaria: 'AHORROS',   NumCuentaBancaria: '000123456789', IdEntidadBancaria: 7,  Anexorut: 'rut-90001.pdf' },
        { Id: 90002, Tipopersona: 'NATURAL',  NumDocumento: '1090909090', Correo: 'gerencia@innovatech.example.com', NomProveedor: 'GERENTE INNOVATECH',       Direccion: 'Cll 45 # 13-20, Bogotá', Web: '',                              Descripcion: 'Persona natural',        Estado: { Id: 1 }, FechaRegistro: '2025-04-22 - 03:40:11 PM', TipoCuentaBancaria: 'CORRIENTE', NumCuentaBancaria: '000987654321', IdEntidadBancaria: 12, Anexorup: 'rup-90002.pdf' }
      ],
      'rep@empresademo.com': [
        { Id: 80001, Tipopersona: 'JURIDICA', NumDocumento: '900111222-3', Correo: 'rep@empresademo.com', NomProveedor: 'Empresa Demo S.A.S.', Direccion: 'Av Caracas # 1-11', Web: 'https://empresademo.example.com', Descripcion: 'Empresa aliada de demostración', Estado: { Id: 1 }, FechaRegistro: '2025-01-15 - 05:04:37 PM', TipoCuentaBancaria: 'AHORROS', NumCuentaBancaria: '000555555555', IdEntidadBancaria: 2 }
      ]
    }
    """

  # ── Proveedores por id de Ágora (enriquecimiento del perfil de empresa) ──────
  # 'AG-PROV-1' es el agora_id_externo de la empresa 1 del seed.
  * def proveedoresPorId =
    """
    {
      'AG-PROV-1': [ { Id: 1, Tipopersona: 'JURIDICA', NumDocumento: '900111222-3', Correo: 'contacto@empresademo.com', NomProveedor: 'Empresa Demo S.A.S.', Direccion: 'Av Caracas # 1-11, Bogotá', Web: 'https://empresademo.example.com', Descripcion: 'Empresa aliada de demostración para pruebas', Estado: { Id: 1 }, FechaRegistro: '2025-01-15 - 05:04:37 PM' } ],
      '90001':     [ { Id: 90001, Tipopersona: 'JURIDICA', NumDocumento: '901234567', Correo: 'gerencia@innovatech.example.com', NomProveedor: 'INNOVATECH S.A.S.', Direccion: 'Cra 7 # 40-53, Bogotá', Web: 'https://innovatech.example.com', Descripcion: 'Servicios de tecnología', Estado: { Id: 1 }, FechaRegistro: '2025-03-10 - 09:15:00 AM' } ],
      '90002':     [ { Id: 90002, Tipopersona: 'NATURAL', NumDocumento: '1090909090', Correo: 'gerencia@innovatech.example.com', NomProveedor: 'GERENTE INNOVATECH', Direccion: 'Cll 45 # 13-20, Bogotá', Web: '', Descripcion: 'Persona natural', Estado: { Id: 1 }, FechaRegistro: '2025-04-22 - 03:40:11 PM' } ]
    }
    """

  # Archivos "subidos" al gestor documental durante la corrida (estado del mock)
  * def uploads = {}

  # Extrae el token crudo del header Authorization ('Bearer x' → 'x')
  * def tokenDe =
    """
    function(headers) {
      for (var k in headers) {
        if (k.toLowerCase() == 'authorization') {
          var v = headers[k][0];
          return v && v.indexOf('Bearer ') == 0 ? v.substring(7) : '';
        }
      }
      return '';
    }
    """

# ─────────────────────────────────────────────────────────────────────────────
Scenario: pathMatches('/oauth2/userinfo')
  * def ident = identidades[tokenDe(requestHeaders)]
  * def responseStatus = ident ? 200 : 401
  * def response = ident ? ident : { error: 'invalid_token', error_description: 'token no reconocido por el mock' }

Scenario: pathMatches('/autenticacion_mid/v1/token/userRol') && methodIs('post')
  * def rol = userRoles[request.user]
  * def responseStatus = rol ? 200 : 400
  * def response = rol ? rol : 'Usuario no registrado'

Scenario: pathMatches('/administrativa_amazon_api/v1/informacion_proveedor')
  * def q = requestParams['query'][0]
  * def response = q.indexOf('correo:') == 0 ? (proveedoresPorCorreo[q.substring(7)] || []) : q.indexOf('id:') == 0 ? (proveedoresPorId[q.substring(3)] || []) : []
  * def responseStatus = 200

Scenario: pathMatches('/terceros_crud/v1/datos_identificacion')
  * def q = requestParams['query'][0]
  # Idioma [{}] = lista vacía (contrato de los *_crud del SGA)
  * def response = q.indexOf('1016060113') >= 0 ? [ { TerceroId: { Id: 1111, NombreCompleto: 'Egresado Demo' } } ] : q.indexOf('1234509876') >= 0 ? [ { TerceroId: { Id: 5555, NombreCompleto: 'NUEVA EGRESADA DEMO' } } ] : [ {} ]
  * def responseStatus = 200

Scenario: pathMatches('/sga_mid/v1/derechos_pecuniarios/consultar_persona/{terceroId}')
  * def response = { Success: true, Status: '200', Data: { Codigos: [ { Proyecto: '20700 - Ingeniería de Sistemas', Dato: '20201020113', IdProyecto: 16, Activo: false } ] } }
  * def responseStatus = 200

# ── Gestor documental (contrato verificado 2026-07-06: respuesta = OBJETO) ────
# OJO: dentro de expresiones JS (ternarios) NO sirven las expresiones embebidas
# '#(var)' de Karate — se referencian las variables JS directamente.
Scenario: pathMatches('/gestor_documental_mid/v1/document/upload') && methodIs('post')
  * def item = request[0]
  * def valido = item != null && item.file != null && item.IdTipoDocumento == 167
  * def uid = 'uid-mock-' + Java.type('java.util.UUID').randomUUID()
  * eval if (valido) uploads[uid] = item.file
  * def responseStatus = valido ? 200 : 422
  * def response = valido ? { Status: '200', res: { Enlace: uid } } : { Status: '422', Error: 'payload inválido para el mock (se espera array con file e IdTipoDocumento=167)' }

Scenario: pathMatches('/gestor_documental_mid/v1/document/{uid}') && methodIs('get')
  * def file = uploads[pathParams.uid]
  * def responseStatus = file ? 200 : 404
  * def response = file ? { file: file } : { error: 'documento no existe en el mock' }

Scenario: pathMatches('/gestor_documental_mid/v1/document/{uid}') && methodIs('delete')
  * eval delete uploads[pathParams.uid]
  * def response = { Status: '200' }

# ── Catch-all: cualquier ruta no simulada falla explícito (no en silencio) ────
Scenario:
  * def responseStatus = 404
  * def response = { error: 'mock institucional: ruta no simulada', metodo: '#(requestMethod)', uri: '#(requestUri)' }
