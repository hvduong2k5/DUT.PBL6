# Updated 2026-11-21: use the shared migration runner; never truncate stock.
param([switch]$SeedDemo)
$ErrorActionPreference = "Stop"
if (-not $env:DB_URL) { throw "Set DB_URL to the intended Inventory database first. See services/inventory-service/MIGRATIONS.md." }
$serviceDir = Join-Path $PSScriptRoot "../../services/inventory-service"
Push-Location $serviceDir
try {
    go run ./cmd/migrate
    if ($LASTEXITCODE -ne 0) { throw "Migration failed. Legacy databases need a reviewed one-time baseline; see MIGRATIONS.md." }
    if ($SeedDemo) {
        $previousSeedFlag = $env:INVENTORY_SEED_DEMO
        try {
            $env:INVENTORY_SEED_DEMO = "true"
            go run ./cmd/seed-demo
            if ($LASTEXITCODE -ne 0) { throw "Demo seed failed." }
        } finally { $env:INVENTORY_SEED_DEMO = $previousSeedFlag }
    }
} finally { Pop-Location }
