param([switch]$StartStack)
$ErrorActionPreference = 'Stop'
$orderTaskRoot = (Resolve-Path (Join-Path $PSScriptRoot '../../..')).Path
Push-Location $orderTaskRoot
try {
    if ($StartStack) {
        docker compose -f services/order-service/docker-compose.test.yml up -d --build
        if ($LASTEXITCODE -ne 0) { throw 'Order sandbox startup failed.' }
    }
    $orderReady = $false
    for ($orderAttempt = 0; $orderAttempt -lt 60; $orderAttempt++) {
        try { $orderReady = (Invoke-RestMethod http://127.0.0.1:18004/readyz).ready } catch { }
        if ($orderReady) { break }
        Start-Sleep -Milliseconds 500
    }
    if (-not $orderReady) { throw 'Order API readiness timed out.' }
    $orderCollection = 'postman/collections/MS-04 Order Regression'
    postman collection lint $orderCollection
    if ($LASTEXITCODE -ne 0) { throw 'Order collection lint failed.' }
    postman collection run $orderCollection --delay-request 100 --timeout 60000 --no-report-events *> docs/04_testing/order-service/postman-cli-output.txt
    $orderRunExit = $LASTEXITCODE
    Get-Content docs/04_testing/order-service/postman-cli-output.txt -Tail 36
    @{exitCode=$orderRunExit; target='http://127.0.0.1:18004'; cloudUpload=$false} |
        ConvertTo-Json | Set-Content -Encoding UTF8 docs/04_testing/order-service/postman-run-result.json
    if ($orderRunExit -ne 0) { throw 'Order Postman assertions failed.' }
} finally { Pop-Location }
