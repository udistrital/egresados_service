# Arranque de desarrollo del MID (localhost:8081).
# Uso:  .\run_dev.ps1   (desde la raíz del repo)
#
# EGRESADOS_SERVICE_PARAMETROS_LOCAL=true → usa el fallback local de
# parámetros (los TipoParametro/Parametro del módulo AÚN no existen en el
# servicio institucional — pendiente operativo C-1). Sin esta variable el
# catálogo responde 500 (401 del gateway de parámetros).
$env:EGRESADOS_SERVICE_RUNMODE = 'dev'
$env:EGRESADOS_SERVICE_PARAMETROS_LOCAL = 'true'

# Las 9 URLs de servicios externos ya NO tienen default quemado en el código
# (conf/app.conf las expone vacías si no hay env var). Sin esto el login real
# contra WSO2 y el JIT provisioning fallan con "unsupported protocol scheme".
$env:EGRESADOS_SERVICE_CRUD_URL = 'http://localhost:8080/v1'
$env:EGRESADOS_SERVICE_AUTENTICACION_URL = 'https://autenticacion.portaloas.udistrital.edu.co/apioas/autenticacion_mid/v1'
$env:EGRESADOS_SERVICE_PARAMETROS_URL = 'https://autenticacion.portaloas.udistrital.edu.co/apioas/parametros/v1'
$env:EGRESADOS_SERVICE_AMAZON_URL = 'https://autenticacion.portaloas.udistrital.edu.co/apioas/administrativa_amazon_api/v1'
$env:EGRESADOS_SERVICE_USERINFO_URL = 'https://autenticacion.portaloas.udistrital.edu.co/oauth2/userinfo'
$env:EGRESADOS_SERVICE_JWKS_URL = 'https://autenticacion.portaloas.udistrital.edu.co/oauth2/jwks'
$env:EGRESADOS_SERVICE_TERCEROS_URL = 'https://autenticacion.portaloas.udistrital.edu.co/apioas/terceros_crud/v1'
$env:EGRESADOS_SERVICE_SGA_MID_URL = 'https://autenticacion.portaloas.udistrital.edu.co/apioas/sga_mid/v1'
$env:EGRESADOS_SERVICE_GESTOR_DOCUMENTAL_URL = 'https://autenticacion.portaloas.udistrital.edu.co/apioas/gestor_documental_mid/v1'

go run main.go
