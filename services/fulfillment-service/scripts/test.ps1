$ErrorActionPreference = 'Stop'
$workspace = (Resolve-Path (Join-Path $PSScriptRoot '../../..')).Path
$compose = Join-Path $workspace 'services/fulfillment-service/docker-compose.test.yml'
$project = 'fulfillment-test-' + $PID
Push-Location -LiteralPath $workspace
try {
    docker build -f services/fulfillment-service/Dockerfile.tools -t omamx/fulfillment-go-tools:local services/fulfillment-service
    if ($LASTEXITCODE -ne 0) { throw 'Go toolchain build failed' }
    docker compose -p $project -f $compose run --rm tests
    if ($LASTEXITCODE -ne 0) { throw 'Fulfillment runtime tests failed' }
    python services/fulfillment-service/scripts/verify-runtime-contracts.py
    if ($LASTEXITCODE -ne 0) { throw 'Runtime contract validation failed' }
} finally {
    docker compose -p $project -f $compose down
    Pop-Location
}
