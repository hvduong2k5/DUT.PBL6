param([switch]$KeepStack)
$ErrorActionPreference = 'Stop'
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '../../..')).Path
$revision = '21775d667bdf1373dce76c33d27f35aba1c8f5b0'
$runID = [guid]::NewGuid().ToString('N')
$project = 'proc-inv-e2e-' + $runID.Substring(0,10)
$runDir = Join-Path $repoRoot ('.tmp/receiving-e2e-' + $runID)
$compose = Join-Path $repoRoot 'services/procurement-service/docker-compose.receiving-test.yml'
function Assert-Exit([string]$step) { if ($LASTEXITCODE -ne 0) { throw "$step failed (exit $LASTEXITCODE)" } }
$savedEnv = @{}
$envNames = @('PROCUREMENT_TEST_PG_PORT','PROCUREMENT_TEST_KAFKA_PORT','PROCUREMENT_E2E_ISOLATED','PROCUREMENT_E2E_DATABASE_URL','PROCUREMENT_E2E_KAFKA_BROKERS','PROCUREMENT_E2E_PRODUCER_BIN','PROCUREMENT_E2E_REPORT','INVENTORY_TEST_DB_URL','GOMAXPROCS')
foreach ($name in $envNames) { $savedEnv[$name] = [Environment]::GetEnvironmentVariable($name,'Process') }
$started = $false
Push-Location $repoRoot
try {
    docker info --format '{{.ServerVersion}}'; Assert-Exit 'Docker availability'
    New-Item -ItemType Directory -Path $runDir | Out-Null
    $listeners = @([Net.Sockets.TcpListener]::new([Net.IPAddress]::Loopback,0),[Net.Sockets.TcpListener]::new([Net.IPAddress]::Loopback,0))
    try {
        foreach ($listener in $listeners) { $listener.Start() }
        $env:PROCUREMENT_TEST_PG_PORT = [string]$listeners[0].LocalEndpoint.Port
        $env:PROCUREMENT_TEST_KAFKA_PORT = [string]$listeners[1].LocalEndpoint.Port
    } finally { foreach ($listener in $listeners) { $listener.Stop() } }
    $archive = Join-Path $runDir 'inventory-pr118.zip'
    git archive --format=zip "--output=$archive" $revision services/inventory-service; Assert-Exit 'Pinned Inventory archive'
    Expand-Archive -LiteralPath $archive -DestinationPath $runDir
    $inventoryDir = Join-Path $runDir 'services/inventory-service'
    Copy-Item -LiteralPath (Join-Path $PSScriptRoot 'testdata/inventory_receiving_e2e_test.go.txt') -Destination (Join-Path $inventoryDir 'internal/presentation/kafka/procurement_producer_e2e_test.go')
    $started = $true
    docker compose -p $project -f $compose up -d --wait --wait-timeout 150; Assert-Exit 'Isolated infrastructure'
    $env:PROCUREMENT_E2E_ISOLATED = '1'
    $env:PROCUREMENT_E2E_DATABASE_URL = "postgres://e2e:isolated-test-only@127.0.0.1:$env:PROCUREMENT_TEST_PG_PORT/receiving_test?sslmode=disable"
    $env:INVENTORY_TEST_DB_URL = $env:PROCUREMENT_E2E_DATABASE_URL
    $env:PROCUREMENT_E2E_KAFKA_BROKERS = "127.0.0.1:$env:PROCUREMENT_TEST_KAFKA_PORT"
    $env:PROCUREMENT_E2E_PRODUCER_BIN = Join-Path $runDir 'procurement-producer.test.exe'
    if ($env:OS -ne 'Windows_NT') { $env:PROCUREMENT_E2E_PRODUCER_BIN = Join-Path $runDir 'procurement-producer.test' }
    $env:PROCUREMENT_E2E_REPORT = Join-Path $runDir 'published-receipts.json'
    $env:GOMAXPROCS = '2'
    Push-Location (Join-Path $repoRoot 'services/procurement-service')
    try { go test -p 1 -c -o $env:PROCUREMENT_E2E_PRODUCER_BIN ./internal/repository/postgres; Assert-Exit 'Producer test build' } finally { Pop-Location }
    Push-Location $inventoryDir
    try { go test -p 1 ./internal/presentation/kafka -run '^TestProcurementProducerConsumerE2E$' -count=1 -timeout=4m -v; Assert-Exit 'Producer-consumer integration' } finally { Pop-Location }
    Write-Output "PASS: pinned Inventory $revision; evidence: $runDir"
} finally {
    if ($started) {
        docker compose -p $project -f $compose logs --no-color --tail 40
        if (!$KeepStack) {
            # Only the uniquely named disposable project created by this invocation.
            docker compose -p $project -f $compose down --volumes
        } else { Write-Output "Retained isolated project: $project; source/evidence: $runDir" }
    }
    foreach ($name in $envNames) { [Environment]::SetEnvironmentVariable($name,$savedEnv[$name],'Process') }
    Pop-Location
}
