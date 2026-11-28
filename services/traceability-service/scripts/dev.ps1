param(
    [ValidateSet('Init','Start','Smoke','Tokens','Stop','Test','Race')]
    [string]$Action = 'Init',
    [switch]$Relay,
    [string]$WebBase = 'http://localhost:3000'
)
$ErrorActionPreference = 'Stop'
$taskServiceRoot = Split-Path -Parent $PSScriptRoot
$taskNames = @('POSTGRES_PASSWORD','GOOS','GOARCH','CGO_ENABLED','TRACEABILITY_DB_PASSWORD','DATABASE_URL','APP_MODE','HTTP_ADDR','PUBLIC_WEB_BASE_URL','OUTBOX_RELAY_ENABLED','TRACEABILITY_TEST_DATABASE_URL','TRACEABILITY_TEST_KAFKA_BROKERS')
$taskPrevious = @{}
foreach ($taskName in $taskNames) { $taskPrevious[$taskName] = [Environment]::GetEnvironmentVariable($taskName,'Process') }
function Invoke-Checked {
    param([string]$Program, [string[]]$Arguments)
    & $Program @Arguments
    if ($LASTEXITCODE -ne 0) { throw "$Program failed with exit code $LASTEXITCODE; credentials omitted" }
}
function Ensure-Network {
    $taskNetworks = @(& docker network ls --format '{{.Name}}')
    if ($LASTEXITCODE -ne 0) { throw 'Cannot list local Docker networks' }
    if ($taskNetworks -notcontains 'traceability-local') { Invoke-Checked docker @('network','create','--label','codex.task=traceability-local','traceability-local') }
}
function Assert-OwnedContainer {
    param([string]$Name)
    $taskOwner = & docker inspect --format '{{index .Config.Labels "codex.task"}}' $Name
    if ($LASTEXITCODE -ne 0 -or $taskOwner -ne 'traceability-local') { throw 'Refusing to change a container outside this local task' }
}
function Ensure-Container {
    param([string]$Name,[string[]]$Arguments)
    Ensure-Network
    $taskContainers = @(& docker ps -a --format '{{.Names}}')
    if ($LASTEXITCODE -ne 0) { throw 'Cannot list local Docker containers' }
    if ($taskContainers -contains $Name) { Assert-OwnedContainer $Name; Invoke-Checked docker @('start',$Name) }
    else { Invoke-Checked docker (@('run','-d','--name',$Name,'--label','codex.task=traceability-local','--network','traceability-local') + $Arguments) }
}
function Start-TestDatabase {
    $env:POSTGRES_PASSWORD = $taskPassword
    Ensure-Container 'pbl6-traceability-db' @('--network-alias','db','-p','127.0.0.1:15403:5432','-e','POSTGRES_DB=om_traceability_db','-e','POSTGRES_USER=trace_migrator','-e','POSTGRES_PASSWORD','-v','traceability-local-data:/var/lib/postgresql/data','postgres:16-alpine')
    for ($taskTry=0;$taskTry -lt 30;$taskTry++) {
        $null = & docker exec pbl6-traceability-db pg_isready -U trace_migrator -d om_traceability_db 2>$null
        if ($LASTEXITCODE -eq 0) { return }
        Start-Sleep -Seconds 1
    }
    throw 'Dedicated local PostgreSQL did not become ready'
}
function Start-TestBroker {
    Ensure-Container 'pbl6-traceability-kafka' @('--network-alias','kafka','-p','127.0.0.1:19093:19092','docker.redpanda.com/redpandadata/redpanda:v24.3.18','redpanda','start','--smp=1','--memory=512M','--reserve-memory=0M','--overprovisioned','--node-id=0','--check=false','--kafka-addr=internal://0.0.0.0:9092,external://0.0.0.0:19092','--advertise-kafka-addr=internal://kafka:9092,external://127.0.0.1:19093')
    for ($taskTry=0;$taskTry -lt 30;$taskTry++) {
        $null = & docker exec pbl6-traceability-kafka rpk cluster info --brokers 127.0.0.1:9092 2>$null
        if ($LASTEXITCODE -eq 0) { break }
        Start-Sleep -Seconds 1
    }
    $null = & docker exec pbl6-traceability-kafka rpk topic describe traceability.events.v1 --brokers 127.0.0.1:9092 2>$null
    if ($LASTEXITCODE -ne 0) { Invoke-Checked docker @('exec','pbl6-traceability-kafka','rpk','topic','create','traceability.events.v1','--brokers','127.0.0.1:9092','-p','1','-r','1') }
}
Push-Location -LiteralPath $taskServiceRoot
try {
    if (!(Test-Path -LiteralPath '.env')) {
        $taskBytes = New-Object byte[] 32
        $taskRandom = [Security.Cryptography.RandomNumberGenerator]::Create()
        try { $taskRandom.GetBytes($taskBytes) } finally { $taskRandom.Dispose() }
        $taskPassword = [BitConverter]::ToString($taskBytes).Replace('-','').ToLowerInvariant()
        $taskStream = [IO.File]::Open((Join-Path $taskServiceRoot '.env'),[IO.FileMode]::CreateNew,[IO.FileAccess]::Write)
        try {
            $taskWriter = New-Object IO.StreamWriter($taskStream,([Text.UTF8Encoding]::new($false)))
            try { $taskWriter.WriteLine('TRACEABILITY_DB_PASSWORD=' + $taskPassword) } finally { $taskWriter.Dispose() }
        } finally { $taskStream.Dispose() }
    }
    $taskLines = @(Get-Content -LiteralPath '.env' | Where-Object { $_ -match '^TRACEABILITY_DB_PASSWORD=' })
    if ($taskLines.Count -ne 1) { throw 'Expected one TRACEABILITY_DB_PASSWORD in the dedicated service .env' }
    $taskPassword = $taskLines[0].Substring('TRACEABILITY_DB_PASSWORD='.Length).Trim()
    if ($taskPassword.Length -ge 2 -and (($taskPassword.StartsWith('"') -and $taskPassword.EndsWith('"')) -or ($taskPassword.StartsWith("'") -and $taskPassword.EndsWith("'")))) { $taskPassword = $taskPassword.Substring(1,$taskPassword.Length-2) }
    if (!$taskPassword) { throw 'Dedicated database password is empty' }
    $env:TRACEABILITY_DB_PASSWORD = $taskPassword
    $env:APP_MODE = 'sandbox'
    $env:HTTP_ADDR = '127.0.0.1:8003'
    $env:PUBLIC_WEB_BASE_URL = $WebBase
    $taskEncodedPassword = [Uri]::EscapeDataString($taskPassword)
    $taskHostDSN = 'postgres://trace_migrator:' + $taskEncodedPassword + '@127.0.0.1:15403/om_traceability_db?sslmode=disable'
    switch ($Action) {
        'Init' {
            Start-TestDatabase
            $env:DATABASE_URL = $taskHostDSN
            if (Test-Path -LiteralPath '.sandbox/state.json') {
                $taskSavedState = Get-Content -LiteralPath '.sandbox/state.json' -Raw | ConvertFrom-Json
                if (!$taskSavedState.runtime_role) { throw 'Existing sandbox runtime role is missing' }
                Invoke-Checked go @('run','./cmd/migrate','-runtime-role',$taskSavedState.runtime_role)
            }
            Invoke-Checked go @('run','./cmd/sandbox','-action','init')
        }
        'Start' {
            Start-TestDatabase
            if (!(Test-Path -LiteralPath '.sandbox/state.json')) { throw 'Run -Action Init first' }
            $env:OUTBOX_RELAY_ENABLED = $Relay.IsPresent.ToString().ToLowerInvariant()
            if ($Relay) { Start-TestBroker }
            $taskRuntimeLines = @(Get-Content -LiteralPath '.sandbox/compose.env' | Where-Object { $_ -match '^TRACEABILITY_RUNTIME_DATABASE_URL=' })
            if ($taskRuntimeLines.Count -ne 1) { throw 'Run Init to create the dedicated runtime credential' }
            $env:DATABASE_URL = $taskRuntimeLines[0].Substring('TRACEABILITY_RUNTIME_DATABASE_URL='.Length).Trim()
            $env:GOOS='linux'; $env:GOARCH='amd64'; $env:CGO_ENABLED='0'
            Invoke-Checked go @('build','-trimpath','-o','build/traceability','./cmd/server')
            Invoke-Checked docker @('build','-t','traceability-service-local','.')
            $taskContainers = @(& docker ps -a --format '{{.Names}}')
            if ($taskContainers -contains 'pbl6-traceability-api') {
                Assert-OwnedContainer 'pbl6-traceability-api'
                Invoke-Checked docker @('stop','-t','15','pbl6-traceability-api')
                Invoke-Checked docker @('rm','pbl6-traceability-api')
            }
            Ensure-Container 'pbl6-traceability-api' @('--network-alias','traceability-service','-p','127.0.0.1:8003:8003','-p','127.0.0.1:9003:9003','-e','APP_MODE=sandbox','-e','HTTP_ADDR=0.0.0.0:8003','-e','GRPC_ADDR=0.0.0.0:9003','-e','SANDBOX_ISOLATED_CONTAINER=true','-e','DATABASE_URL','-e',('PUBLIC_WEB_BASE_URL='+$WebBase),'-e','SANDBOX_JWT_PUBLIC_KEY_FILE=/run/traceability/public.pem','-e','SANDBOX_FIXTURE_FILE=/run/traceability/batches.json','-e','OUTBOX_RELAY_ENABLED','-e','KAFKA_BROKERS=kafka:9092','--mount',('type=bind,src='+(Join-Path $taskServiceRoot '.sandbox/public.pem')+',dst=/run/traceability/public.pem,readonly'),'--mount',('type=bind,src='+(Join-Path $taskServiceRoot '.sandbox/batches.json')+',dst=/run/traceability/batches.json,readonly'),'traceability-service-local')
        }
        'Smoke' { Invoke-Checked go @('run','./cmd/sandbox','-action','smoke') }
        'Tokens' { Invoke-Checked go @('run','./cmd/sandbox','-action','tokens') }
        'Stop' { Assert-OwnedContainer 'pbl6-traceability-api'; Invoke-Checked docker @('stop','-t','15','pbl6-traceability-api') }
        'Test' {
            Start-TestDatabase
            Start-TestBroker
            $env:TRACEABILITY_TEST_DATABASE_URL = $taskHostDSN
            $env:TRACEABILITY_TEST_KAFKA_BROKERS = '127.0.0.1:19093'
            Invoke-Checked go @('test','-count=1','-coverpkg=./...','-coverprofile=coverage.out','./...')
            Invoke-Checked go @('tool','cover','-func=coverage.out')
            Invoke-Checked go @('vet','./...')
            Invoke-Checked go @('build','./...')
        }
        'Race' {
            Start-TestDatabase
            Start-TestBroker
            Invoke-Checked docker @('build','--target','test','-t','traceability-service-test','.')
            $env:TRACEABILITY_TEST_DATABASE_URL = 'postgres://trace_migrator:' + $taskEncodedPassword + '@db:5432/om_traceability_db?sslmode=disable'
            $env:TRACEABILITY_TEST_KAFKA_BROKERS = 'kafka:9092'
            Invoke-Checked docker @('run','--rm','--network','traceability-local','--env','TRACEABILITY_TEST_DATABASE_URL','--env','TRACEABILITY_TEST_KAFKA_BROKERS','traceability-service-test')
        }
    }
} finally {
    foreach ($taskName in $taskNames) {
        if ($null -eq $taskPrevious[$taskName]) { Remove-Item -LiteralPath ('Env:' + $taskName) -ErrorAction SilentlyContinue }
        else { [Environment]::SetEnvironmentVariable($taskName,$taskPrevious[$taskName],'Process') }
    }
    Pop-Location
}
