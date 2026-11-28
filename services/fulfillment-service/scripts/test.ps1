$ErrorActionPreference = 'Stop'
$workspace = (Resolve-Path (Join-Path $PSScriptRoot '../../..')).Path
$compose = Join-Path $workspace 'infra/docker/stacks/fulfillment-test.compose.yml'
$project = 'fulfillment-test-' + $PID
Push-Location -LiteralPath $workspace
try {
    docker build --build-arg SERVICE_PATH=services/fulfillment-service -f infra/docker/go-tools.Dockerfile -t omamx/fulfillment-go-tools:local .
    if ($LASTEXITCODE -ne 0) { throw 'Go toolchain build failed' }
    docker compose -p $project -f $compose run --rm tests
    if ($LASTEXITCODE -ne 0) { throw 'Fulfillment runtime tests failed' }
    python services/fulfillment-service/scripts/verify-runtime-contracts.py
    if ($LASTEXITCODE -ne 0) { throw 'Runtime contract validation failed' }
} finally {
    docker compose -p $project -f $compose down
    Pop-Location
}
