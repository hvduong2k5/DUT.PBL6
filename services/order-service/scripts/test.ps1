param([switch]$StartStack)
$ErrorActionPreference = 'Stop'
$orderRepo = (Resolve-Path (Join-Path $PSScriptRoot '../../..')).Path
Push-Location $orderRepo
try {
    docker build -f services/order-service/Dockerfile.tools -t omamx/order-go-tools:local services/order-service
    if ($LASTEXITCODE -ne 0) { throw 'Go toolchain build failed.' }
    if ($StartStack) {
        docker compose -f services/order-service/docker-compose.test.yml up -d --build
        if ($LASTEXITCODE -ne 0) { throw 'Order stack startup failed.' }
    } else {
        docker compose -f services/order-service/docker-compose.test.yml up -d db redis kafka-init
        if ($LASTEXITCODE -ne 0) { throw 'Order test infrastructure startup failed.' }
    }
    docker compose -f services/order-service/docker-compose.test.yml run --rm tests sh -c 'test -z "$(gofmt -l cmd internal migrations tests)" && go vet ./... && go test -race -count=1 -v ./...' 2>&1 |
        Tee-Object -FilePath docs/04_testing/order-service/go-test-race-output.txt
    $orderGoExit = $LASTEXITCODE
    if ($orderGoExit -ne 0) { throw 'Order Go release gates failed.' }
    if (Test-Path -LiteralPath 'postman/collections/MS-04 Order Regression') {
        & services/order-service/scripts/test-postman.ps1
        if ($LASTEXITCODE -ne 0) { throw 'Order HTTP regression failed.' }
    } else {
        Write-Host 'Postman NOT_RUN: collection is local-only and absent from this checkout; Go HTTP integration tests ran.'
    }
} finally { Pop-Location }
