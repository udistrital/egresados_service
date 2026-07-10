# =============================================================================
# run_pruebas.ps1 — Orquesta la suite Karate del MID de Beneficios Egresados
# =============================================================================
# Qué hace:
#   1. Verifica/arranca PostgreSQL y re-siembra la BD (db/seed_pruebas.sql del CRUD)
#      para que los ids del seed sean deterministas.
#   2. Compila y levanta el CRUD (:8080) y el MID (:8081). El MID se apunta al
#      MOCK institucional (:8090) que la propia suite levanta dentro del JVM.
#   3. Ejecuta `mvn test` (reporte HTML en target/karate-reports/).
#   4. Detiene los servicios al terminar.
#
# Prerrequisitos: Go, Java 11+, Maven, PostgreSQL 16+ (psql se auto-detecta;
# si no, -PsqlPath). Asume que el repo del CRUD es hermano de este, con nombre
# sga_crud_beneficios_egresados o egresados_crud (si no, -CrudPath).
#
# Uso:  .\run_pruebas.ps1            (desde tests/karate)
#       .\run_pruebas.ps1 -NoReseed  (no tocar los datos)
# =============================================================================
param(
    [switch]$NoReseed,
    # Vacío = auto-detectar (PATH y luego C:\Program Files\PostgreSQL\<ver>\bin)
    [string]$PsqlPath   = '',
    # Vacío = buscar el repo del CRUD como hermano de este, por ambos nombres
    # (sga_crud_beneficios_egresados o egresados_crud, el nombre institucional)
    [string]$CrudPath   = '',
    [string]$DbUser     = $(if ($env:EGRESADOS_CRUD_DB_USER) { $env:EGRESADOS_CRUD_DB_USER } else { 'postgres' }),
    [string]$DbPassword = $(if ($env:EGRESADOS_CRUD_DB_PASS) { $env:EGRESADOS_CRUD_DB_PASS } else { '12345' }),
    # BD EXCLUSIVA de pruebas: la suite trunca/siembra datos, por eso NUNCA se
    # apunta a la BD de desarrollo (beneficios_egresados). Se crea sola si falta.
    [string]$DbName     = 'beneficios_egresados_pruebas'
)
$ErrorActionPreference = 'Stop'

$raizKarate = $PSScriptRoot
$raizMid    = (Resolve-Path (Join-Path $raizKarate '..\..')).Path

if ($CrudPath) {
    $raizCrud = (Resolve-Path $CrudPath).Path
} else {
    $raizCrud = $null
    foreach ($nombre in 'sga_crud_beneficios_egresados', 'egresados_crud') {
        $cand = Join-Path (Split-Path $raizMid -Parent) $nombre
        if (Test-Path (Join-Path $cand 'db\schema.sql')) { $raizCrud = (Resolve-Path $cand).Path; break }
    }
    if (-not $raizCrud) {
        throw 'No se encontró el repo del CRUD junto a este (se buscó ..\sga_crud_beneficios_egresados y ..\egresados_crud). Clónalo como hermano del MID o indica su ruta con -CrudPath.'
    }
    Write-Host "Usando CRUD: $raizCrud"
}

if (-not $PsqlPath) {
    $cmd = Get-Command psql -ErrorAction SilentlyContinue
    if ($cmd) { $PsqlPath = $cmd.Source }
    else {
        $PsqlPath = Get-ChildItem 'C:\Program Files\PostgreSQL\*\bin\psql.exe' -ErrorAction SilentlyContinue |
            Sort-Object { $v = 0; [int]::TryParse($_.Directory.Parent.Name, [ref]$v) | Out-Null; $v } -Descending |
            Select-Object -First 1 -ExpandProperty FullName
    }
    if (-not $PsqlPath) {
        throw 'No se encontró psql.exe (ni en el PATH ni en C:\Program Files\PostgreSQL\<ver>\bin). Instala PostgreSQL 16+ o indica la ruta con -PsqlPath.'
    }
    Write-Host "Usando psql: $PsqlPath"
}

function Esperar-Puerto([int]$puerto, [string]$nombre) {
    foreach ($i in 1..60) {
        if ((Test-NetConnection 127.0.0.1 -Port $puerto -WarningAction SilentlyContinue).TcpTestSucceeded) { return }
        Start-Sleep -Milliseconds 500
    }
    throw "$nombre no respondió en el puerto $puerto tras 30s"
}

# ── 1. PostgreSQL (si el puerto ya responde, no se toca el servicio) ──────────
if (-not (Test-NetConnection 127.0.0.1 -Port 5432 -WarningAction SilentlyContinue).TcpTestSucceeded) {
    $svc = Get-Service | Where-Object { $_.Name -match 'postgres' } | Select-Object -First 1
    if ($svc) {
        Write-Host "Arrancando servicio $($svc.Name)..."
        try { Start-Service $svc.Name -ErrorAction Stop } catch { throw "No se pudo arrancar PostgreSQL (¿ejecutar como administrador, o arrancarlo con pg_ctl?): $_" }
    }
}
Esperar-Puerto 5432 'PostgreSQL'

$env:PGPASSWORD = $DbPassword
$existe = & $PsqlPath -U $DbUser -h 127.0.0.1 -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='$DbName'"
if ($existe -ne '1') {
    Write-Host "Creando la BD de pruebas $DbName (schema.sql del CRUD)..."
    & $PsqlPath -U $DbUser -h 127.0.0.1 -d postgres -c "CREATE DATABASE $DbName" | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'No se pudo crear la BD de pruebas' }
    & $PsqlPath -U $DbUser -h 127.0.0.1 -d $DbName -v ON_ERROR_STOP=1 -f (Join-Path $raizCrud 'db\schema.sql') | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Falló la aplicación de db/schema.sql sobre la BD de pruebas' }
}

if (-not $NoReseed) {
    Write-Host "Re-sembrando $DbName con db/seed_pruebas.sql..."
    & $PsqlPath -U $DbUser -h 127.0.0.1 -d $DbName -v ON_ERROR_STOP=1 -f (Join-Path $raizCrud 'db\seed_pruebas.sql') | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Falló la siembra de la BD de pruebas (¿credenciales?)' }
}

# ── 2. Compilar y levantar CRUD y MID ─────────────────────────────────────────
$binDir = Join-Path $raizKarate 'target\bin'
New-Item -ItemType Directory -Force $binDir | Out-Null

Write-Host 'Compilando CRUD y MID...'
Push-Location $raizCrud
go build -o (Join-Path $binDir 'crud_pruebas.exe') .
if ($LASTEXITCODE -ne 0) { Pop-Location; throw 'No compiló el CRUD' }
Pop-Location
Push-Location $raizMid
go build -o (Join-Path $binDir 'mid_pruebas.exe') .
if ($LASTEXITCODE -ne 0) { Pop-Location; throw 'No compiló el MID' }
Pop-Location

$procs = @()
try {
    Write-Host "Levantando CRUD (:8080) contra la BD $DbName..."
    # Nombres de env estandarizados por la universidad (conf/app.conf, 2026-07-09):
    # EGRESADOS_CRUD_* — sin default quemado, hay que setear TODAS.
    $env:EGRESADOS_CRUD_HTTPPORT  = '8080'
    $env:EGRESADOS_CRUD_RUNMODE   = 'dev'
    $env:EGRESADOS_CRUD_DB_USER   = $DbUser
    $env:EGRESADOS_CRUD_DB_PASS   = $DbPassword
    $env:EGRESADOS_CRUD_DB_URL    = '127.0.0.1'
    $env:EGRESADOS_CRUD_DB_PORT   = '5432'
    $env:EGRESADOS_CRUD_DB_NAME   = $DbName
    $env:EGRESADOS_CRUD_DB_SCHEMA = 'beneficios_egresados'
    $procs += Start-Process (Join-Path $binDir 'crud_pruebas.exe') -WorkingDirectory $raizCrud -PassThru -WindowStyle Hidden
    Esperar-Puerto 8080 'CRUD'

    Write-Host 'Levantando MID (:8081) apuntando al mock institucional (:8090)...'
    $env:EGRESADOS_SERVICE_HTTP_PORT = '8081'
    $env:EGRESADOS_SERVICE_RUNMODE   = 'dev'
    # Catálogos con la semilla local (mismos ids institucionales 7199+)
    $env:EGRESADOS_SERVICE_PARAMETROS_LOCAL = 'true'
    # Todos los servicios institucionales van al mock de la suite. El middleware
    # de token queda ACTIVO: los tokens ficticios son opacos y se validan contra
    # el userinfo del mock (misma rama de código que producción).
    $env:EGRESADOS_SERVICE_CRUD_URL              = 'http://localhost:8080/v1'
    $env:EGRESADOS_SERVICE_USERINFO_URL          = 'http://localhost:8090/oauth2/userinfo'
    $env:EGRESADOS_SERVICE_AUTENTICACION_URL     = 'http://localhost:8090/autenticacion_mid/v1'
    $env:EGRESADOS_SERVICE_AMAZON_URL            = 'http://localhost:8090/administrativa_amazon_api/v1'
    $env:EGRESADOS_SERVICE_TERCEROS_URL          = 'http://localhost:8090/terceros_crud/v1'
    $env:EGRESADOS_SERVICE_SGA_MID_URL           = 'http://localhost:8090/sga_mid/v1'
    $env:EGRESADOS_SERVICE_GESTOR_DOCUMENTAL_URL = 'http://localhost:8090/gestor_documental_mid/v1'
    # academica_jbpm (carrera, nuevo del merge 2026-07-09): best-effort, degrada sin mock
    $env:EGRESADOS_SERVICE_ACADEMICA_JBPM_URL    = 'http://localhost:8090/academica_jbpm/v1'
    $procs += Start-Process (Join-Path $binDir 'mid_pruebas.exe') -WorkingDirectory $raizMid -PassThru -WindowStyle Hidden
    Esperar-Puerto 8081 'MID'

    # ── 3. Suite Karate ───────────────────────────────────────────────────────
    Write-Host 'Ejecutando la suite Karate (mvn test)...'
    Push-Location $raizKarate
    mvn --% -q test
    $exit = $LASTEXITCODE
    Pop-Location

    if ($exit -eq 0) { Write-Host "`n✔ Suite Karate del MID: TODO EN VERDE" -ForegroundColor Green }
    else { Write-Host "`n✘ Suite Karate del MID: hay fallos. Reporte: tests\karate\target\karate-reports\karate-summary.html" -ForegroundColor Red }
    exit $exit
}
finally {
    # ── 4. Apagar servicios ───────────────────────────────────────────────────
    foreach ($p in $procs) {
        if ($p -and -not $p.HasExited) { Stop-Process -Id $p.Id -Force -ErrorAction SilentlyContinue }
    }
}
