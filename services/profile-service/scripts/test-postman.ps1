param([switch]$StartStack)
$ErrorActionPreference = 'Stop'
$taskRoot = (Resolve-Path (Join-Path $PSScriptRoot '../../..')).Path
if (-not $env:PROFILE_INTERNAL_TOKEN) { throw 'Set PROFILE_INTERNAL_TOKEN to a temporary local test token.' }
Push-Location $taskRoot
try {
    if ($StartStack) {
        docker compose -f services/profile-service/docker-compose.test.yml up -d --build
        if ($LASTEXITCODE -ne 0) { throw 'Test stack failed to start.' }
    }
    # This is the isolated profile-hardening database, never the development database.
    Get-Content -Raw -Encoding UTF8 services/profile-service/tests/fixtures/hardening_seed.sql |
        docker exec -i profile-hardening-db-1 psql -U profile_test -d profile_test -v ON_ERROR_STOP=1
    if ($LASTEXITCODE -ne 0) { throw 'Fixture setup failed.' }
    $collection = 'postman/collections/MS-15- Profile Service - Data Transfer & End-to-End API Test Sui'
    postman collection lint $collection
    if ($LASTEXITCODE -ne 0) { throw 'Collection lint failed.' }
    $report = 'docs/04_testing/profile-service/postman-cli-output.txt'
    postman collection run $collection --env-var "internalToken=$env:PROFILE_INTERNAL_TOKEN" --no-report-events *> $report
    $runExitCode = $LASTEXITCODE
    Get-Content -LiteralPath $report -Tail 38
    @{exitCode=$runExitCode; report=$report; target='http://localhost:18080'; cloudUpload=$false} |
        ConvertTo-Json | Set-Content -Encoding UTF8 docs/04_testing/profile-service/postman-run-result.json
    if ($runExitCode -ne 0) { throw "Postman assertions failed. See $report" }
} finally { Pop-Location }
