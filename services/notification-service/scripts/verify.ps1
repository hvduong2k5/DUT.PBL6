param([string]$Go = 'go', [switch]$Integration)
$ErrorActionPreference = 'Stop'
Push-Location (Join-Path $PSScriptRoot '..')
$oldRequired = $env:NOTIFICATION_REQUIRE_INTEGRATION
try {
    if ($Integration) {
        if (-not $env:NOTIFICATION_TEST_DATABASE_URL) { throw 'Dedicated NOTIFICATION_TEST_DATABASE_URL required' }
        $env:NOTIFICATION_REQUIRE_INTEGRATION = '1'
    }
    & $Go test -race -count=1 -v ./...
    if ($LASTEXITCODE -ne 0) { throw 'Go tests failed' }
    & $Go vet ./...
    if ($LASTEXITCODE -ne 0) { throw 'Go vet failed' }
    & $Go build ./...
    if ($LASTEXITCODE -ne 0) { throw 'Go build failed' }
} finally {
    $env:NOTIFICATION_REQUIRE_INTEGRATION = $oldRequired
    Pop-Location
}
