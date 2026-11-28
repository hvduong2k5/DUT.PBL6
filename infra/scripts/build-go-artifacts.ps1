param(
    [Parameter(Mandatory)][ValidateSet('order-service','fulfillment-service')][string]$Service,
    [ValidateSet('amd64','arm64')][string]$Architecture = 'amd64'
)
$ErrorActionPreference = 'Stop'
$artifactRepo = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path
$artifactPrevious = @{}
foreach ($artifactName in @('GOOS','GOARCH','CGO_ENABLED')) {
    $artifactPrevious[$artifactName] = [Environment]::GetEnvironmentVariable($artifactName, 'Process')
}
Push-Location -LiteralPath (Join-Path $artifactRepo "services/$Service")
try {
    $env:GOOS = 'linux'; $env:GOARCH = $Architecture; $env:CGO_ENABLED = '0'
    New-Item -ItemType Directory -Force -Path build | Out-Null
    $artifactTargets = if ($Service -eq 'order-service') {
        @(@('order-api','./cmd/server'), @('order-simulator','./cmd/simulator'))
    } else {
        @(@('fulfillment-server','./cmd/server'), @('fulfillment-migrate','./cmd/migrate'))
    }
    foreach ($artifactTarget in $artifactTargets) {
        & go build -trimpath -o "build/$($artifactTarget[0])" $artifactTarget[1]
        if ($LASTEXITCODE -ne 0) { throw "Linux artifact build failed: $($artifactTarget[0])" }
    }
} finally {
    foreach ($artifactName in $artifactPrevious.Keys) {
        [Environment]::SetEnvironmentVariable($artifactName, $artifactPrevious[$artifactName], 'Process')
    }
    Pop-Location
}
