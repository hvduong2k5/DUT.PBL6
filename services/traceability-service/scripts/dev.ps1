param(
    [ValidateSet('Init','Start','Smoke','Tokens','Stop','Test','Race')]
    [string]$Action = 'Init',
    [switch]$Relay,
    [string]$WebBase = 'http://localhost:3000'
)
$ErrorActionPreference = 'Stop'
$taskServiceRoot = Split-Path -Parent $PSScriptRoot
$taskNames = @('COMPOSE_PROJECT_NAME','TRACEABILITY_DB_PASSWORD','DATABASE_URL','APP_MODE','HTTP_ADDR','PUBLIC_WEB_BASE_URL','OUTBOX_RELAY_ENABLED','TRACEABILITY_TEST_DATABASE_URL','TRACEABILITY_TEST_KAFKA_BROKERS')
$taskPrevious = @{}
foreach ($taskName in $taskNames) { $taskPrevious[$taskName] = [Environment]::GetEnvironmentVariable($taskName,'Process') }
function Invoke-Checked {
    param([string]$Program, [string[]]$Arguments)
    & $Program @Arguments
    if ($LASTEXITCODE -ne 0) { throw "$Program failed with exit code $LASTEXITCODE; credentials omitted" }
}
function Start-TestBroker {
    Invoke-Checked docker @('compose','--profile','broker','up','-d','--wait','kafka')
    $taskTopicPreference = $ErrorActionPreference
    try {
        $ErrorActionPreference = 'Continue'
        $null = & docker compose exec -T kafka rpk topic describe traceability.events.v1 --brokers 127.0.0.1:9092 2>&1
        $taskTopicExists = $LASTEXITCODE -eq 0
    } finally { $ErrorActionPreference = $taskTopicPreference }
    if (!$taskTopicExists) { Invoke-Checked docker @('compose','exec','-T','kafka','rpk','topic','create','traceability.events.v1','--brokers','127.0.0.1:9092','-p','1','-r','1') }
}
Push-Location -LiteralPath $taskServiceRoot
try {
    $env:COMPOSE_PROJECT_NAME = 'traceability-service'
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
            Invoke-Checked docker @('compose','up','-d','--wait','db')
            $env:DATABASE_URL = $taskHostDSN
            Invoke-Checked go @('run','./cmd/sandbox','-action','init')
        }
        'Start' {
            if (!(Test-Path -LiteralPath '.sandbox/state.json')) { throw 'Run -Action Init first' }
            $env:OUTBOX_RELAY_ENABLED = $Relay.IsPresent.ToString().ToLowerInvariant()
            if ($Relay) { Start-TestBroker }
            Invoke-Checked docker @('compose','--env-file','.env','--env-file','.sandbox/compose.env','--profile','api','up','-d','--wait','--build','api')
        }
        'Smoke' { Invoke-Checked go @('run','./cmd/sandbox','-action','smoke') }
        'Tokens' { Invoke-Checked go @('run','./cmd/sandbox','-action','tokens') }
        'Stop' { Invoke-Checked docker @('compose','--env-file','.env','--env-file','.sandbox/compose.env','--profile','api','stop','api') }
        'Test' {
            Invoke-Checked docker @('compose','up','-d','--wait','db')
            Start-TestBroker
            $env:TRACEABILITY_TEST_DATABASE_URL = $taskHostDSN
            $env:TRACEABILITY_TEST_KAFKA_BROKERS = '127.0.0.1:19093'
            Invoke-Checked go @('test','-count=1','-coverpkg=./...','-coverprofile=coverage.out','./...')
            Invoke-Checked go @('tool','cover','-func=coverage.out')
            Invoke-Checked go @('vet','./...')
            Invoke-Checked go @('build','./...')
        }
        'Race' {
            Invoke-Checked docker @('compose','up','-d','--wait','db')
            Start-TestBroker
            Invoke-Checked docker @('build','-f','Dockerfile.test','-t','traceability-service-test','.')
            $env:TRACEABILITY_TEST_DATABASE_URL = 'postgres://trace_migrator:' + $taskEncodedPassword + '@db:5432/om_traceability_db?sslmode=disable'
            $env:TRACEABILITY_TEST_KAFKA_BROKERS = 'kafka:9092'
            Invoke-Checked docker @('run','--rm','--network','traceability-service_traceability','--env','TRACEABILITY_TEST_DATABASE_URL','--env','TRACEABILITY_TEST_KAFKA_BROKERS','traceability-service-test')
        }
    }
} finally {
    foreach ($taskName in $taskNames) {
        if ($null -eq $taskPrevious[$taskName]) { Remove-Item -LiteralPath ('Env:' + $taskName) -ErrorAction SilentlyContinue }
        else { [Environment]::SetEnvironmentVariable($taskName,$taskPrevious[$taskName],'Process') }
    }
    Pop-Location
}
