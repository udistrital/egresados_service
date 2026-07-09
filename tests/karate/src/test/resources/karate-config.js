function fn() {
  // URLs de los servicios bajo prueba (sobreescribibles con -Dmid.url=... etc.)
  var config = {
    midUrl: karate.properties['mid.url'] || 'http://localhost:8081/v1',
    crudUrl: karate.properties['crud.url'] || 'http://localhost:8080/v1',

    // Tokens FICTICIOS que el mock institucional (puerto 8090) traduce a
    // identidades. Son opacos (sin '.'), así el middleware del MID los valida
    // contra userinfo — que es el propio mock. Ver mocks/institucional-mock.feature.
    tokenEgresado: 'Bearer token-egresado',           // sub SGA-2  → usuario 2 / egresado 1 del seed
    tokenEgresadoNuevo: 'Bearer token-egresado-nuevo', // sin provisionar: lo crea el JIT
    tokenEmpresa: 'Bearer token-empresa',             // sub AGORA-1 → usuario 1, vinculado a empresa 1 del seed
    tokenEmpresaNueva: 'Bearer token-empresa-nueva',   // sin provisionar: caso 1:N (2 proveedores)
    tokenIntruso: 'Bearer token-intruso',             // autenticado pero SIN vínculo local (anti-IDOR)

    // Datos del seed (db/seed_pruebas.sql del CRUD) — el script de ejecución
    // re-siembra la BD antes de la corrida, por eso los ids son deterministas.
    empresaSeedId: 1,
    usuarioEmpresaSeedId: 1,   // usuario (tabla usuario) del representante de la empresa
    egresadoSeedId: 1,
    usuarioEgresadoSeedId: 2,  // usuario (tabla usuario) del egresado

    // PDF mínimo válido (cabecera %PDF) para subidas al gestor documental mockeado
    pdfBase64: 'JVBERi0xLjQKJSBBcmNoaXZvIGRlIHBydWViYSBLYXJhdGUgLSBTR0EgQmVuZWZpY2lvcyBFZ3Jlc2Fkb3MKJSVFT0Y=',
    // base64 de un texto plano: NO es PDF (para probar la validación esPdfBase64)
    noEsPdfBase64: 'aG9sYSBubyBzb3kgdW4gcGRm'
  };

  // Fechas para publicar beneficios vigentes (RN-008: fecha_fin >= hoy)
  var LocalDate = Java.type('java.time.LocalDate');
  config.hoy = LocalDate.now().toString();
  config.finVigencia = LocalDate.now().plusMonths(3).toString();

  // El MID hace varias llamadas encadenadas al CRUD por request (C-4b, anti-IDOR):
  // márgenes holgados para máquinas lentas.
  karate.configure('connectTimeout', 10000);
  karate.configure('readTimeout', 60000);
  return config;
}
