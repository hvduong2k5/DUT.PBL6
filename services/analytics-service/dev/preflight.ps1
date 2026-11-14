param(
    [switch]$Supply
)

$ErrorActionPreference = 'Stop'
$root = (Resolve-Path (Join-Path $PSScriptRoot '../../..')).Path
$compose = Join-Path $root 'services/analytics-service/compose.dev.yml'

function Require-Env([string]$name) {
    $value = [Environment]::GetEnvironmentVariable($name)
    if ([string]::IsNullOrWhiteSpace($value)) { throw "$name is required" }
    return $value
}

$image = Require-Env 'ANALYTICS_IMAGE'
if ($image -notmatch '@sha256:[0-9a-f]{64}$') { throw 'ANALYTICS_IMAGE must be pinned by digest' }
$config = (Resolve-Path -LiteralPath (Require-Env 'ANALYTICS_DEV_CONFIG')).Path
if ($config.StartsWith($root, [StringComparison]::OrdinalIgnoreCase)) {
    throw 'ANALYTICS_DEV_CONFIG must be outside the repository'
}
$network = Require-Env 'ANALYTICS_DEV_NETWORK'

foreach ($role in @('migration', 'collector', 'builder', 'api')) {
    $envFile = Join-Path $config "$role/service.env"
    if (-not (Test-Path -LiteralPath $envFile -PathType Leaf)) { throw "Missing $role/service.env" }
}

if ($Supply) {
    $inbox = (Resolve-Path -LiteralPath (Require-Env 'ANALYTICS_SUPPLY_INBOX')).Path
    if ($inbox.StartsWith($root, [StringComparison]::OrdinalIgnoreCase)) {
        throw 'ANALYTICS_SUPPLY_INBOX must be outside the repository'
    }
    foreach ($owner in @('order-service', 'inventory-service')) {
        if (-not (Test-Path -LiteralPath (Join-Path $inbox $owner) -PathType Container)) {
            throw "Missing supply owner directory: $owner"
        }
    }
}

docker image inspect $image --format '{{.Id}}' | Out-Null
if ($LASTEXITCODE -ne 0) { throw 'Pinned Analytics image is unavailable on this host' }
docker network inspect $network --format '{{.Name}}' | Out-Null
if ($LASTEXITCODE -ne 0) { throw 'Private dev network is unavailable on this host' }
docker compose -f $compose --profile migration --profile supply --profile pilot config -q
if ($LASTEXITCODE -ne 0) { throw 'Analytics Compose configuration is invalid' }

Write-Output 'Analytics pilot preflight passed. No service was started.'
