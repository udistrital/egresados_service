# Arranque de desarrollo del MID (localhost:8081).
# Uso:  .\run_dev.ps1   (desde la raíz del repo)
#
# EGRESADOS_SERVICE_PARAMETROS_LOCAL=true → usa el fallback local de
# parámetros (los TipoParametro/Parametro del módulo AÚN no existen en el
# servicio institucional — pendiente operativo C-1). Sin esta variable el
# catálogo responde 500 (401 del gateway de parámetros).
$env:EGRESADOS_SERVICE_RUNMODE = 'dev'
$env:EGRESADOS_SERVICE_PARAMETROS_LOCAL = 'true'
go run main.go
