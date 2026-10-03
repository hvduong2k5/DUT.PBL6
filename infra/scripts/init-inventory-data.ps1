# =============================================================================
# SCRIPT TU DONG KHOI TAO HA TANG DOCKER & NAP DU LIEU MAU CHO INVENTORY SERVICE
# Monorepo PBL6: O Ma Hue - Me xung OCOP
# =============================================================================

$ErrorActionPreference = "Stop"

# Thiet lap UTF-8 encoding khong co BOM cho console
$utf8NoBom = New-Object System.Text.UTF8Encoding $false
[Console]::OutputEncoding = $utf8NoBom
$OutputEncoding = $utf8NoBom

Write-Host "==========================================================" -ForegroundColor Cyan
Write-Host "   KHO DAC SAN HUE O MA - KHOI TAO DATABASE & SEED DATA    " -ForegroundColor Cyan
Write-Host "==========================================================" -ForegroundColor Cyan

# Xac dinh duong dan tuyet doi doc lap theo vi tri file script
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot = (Resolve-Path "$ScriptDir\..\..").Path
$ComposeFile = Join-Path $RepoRoot "infra\docker\docker-compose.infra.yml"
$SchemaFile = Join-Path $RepoRoot "services\inventory-service\migrations\000001_init_inventory_schema.up.sql"
$SeedFile = Join-Path $RepoRoot "services\inventory-service\migrations\seed_sample_data.sql"

if (-not (Test-Path $ComposeFile)) {
    Write-Host " -> Loi: Khong tim thay compose file tai: $ComposeFile" -ForegroundColor Red
    exit 1
}

# Ham kiem tra an toan trang thai Docker daemon
function Test-DockerRunning {
    $prevEA = $ErrorActionPreference
    $ErrorActionPreference = "SilentlyContinue"
    try {
        $null = docker info 2>&1
        return ($LASTEXITCODE -eq 0)
    } catch {
        return $false
    } finally {
        $ErrorActionPreference = $prevEA
    }
}

# Ham kiem tra ket noi PostgreSQL
function Test-PostgresReady {
    $prevEA = $ErrorActionPreference
    $ErrorActionPreference = "SilentlyContinue"
    try {
        $res = docker exec -i om-postgres psql -U postgres -tAc "SELECT 1" 2>&1
        return ($res.Trim() -eq "1")
    } catch {
        return $false
    } finally {
        $ErrorActionPreference = $prevEA
    }
}

# Ham kiem tra Database da ton tai
function Test-DbExists([string]$dbName) {
    $prevEA = $ErrorActionPreference
    $ErrorActionPreference = "SilentlyContinue"
    try {
        $raw = docker exec -i om-postgres psql -U postgres -tAc "SELECT 1 FROM pg_database WHERE datname = '$dbName'" 2>&1
        return ($raw.Trim() -eq "1")
    } catch {
        return $false
    } finally {
        $ErrorActionPreference = $prevEA
    }
}

# 1. Kiem tra Docker daemon & ho tro khoi dong
Write-Host "`n[1/5] Kiem tra trang thai Docker Desktop..." -ForegroundColor Yellow
if (-not (Test-DockerRunning)) {
    Write-Host " -> Docker daemon chua ket noi. Dang kiem tra service com.docker.service..." -ForegroundColor Yellow
    $dockerService = Get-Service "com.docker.service" -ErrorAction SilentlyContinue
    if ($dockerService -and $dockerService.Status -ne "Running") {
        Write-Host " -> Dang khoi dong Windows service com.docker.service..." -ForegroundColor Yellow
        Start-Service "com.docker.service" -ErrorAction SilentlyContinue
    }

    $dockerDesktopExe = "C:\Program Files\Docker\Docker\Docker Desktop.exe"
    if (Test-Path $dockerDesktopExe) {
        Write-Host " -> Dang khoi chay Docker Desktop ($dockerDesktopExe)..." -ForegroundColor Yellow
        Start-Process $dockerDesktopExe -ErrorAction SilentlyContinue
        Write-Host " -> Dang cho Docker daemon san sang (kiem tra trong 45 giay)..." -ForegroundColor Yellow
        
        $retries = 25
        $ready = $false
        while ($retries -gt 0) {
            Start-Sleep -Seconds 2
            if (Test-DockerRunning) {
                $ready = $true
                break
            }
            $retries--
        }
        if (-not $ready) {
            Write-Host " -> [CHUYEN TIET] Docker Desktop chua san sang tu dong." -ForegroundColor Red
            Write-Host "    Vui long MO UNG DUNG DOCKER DESKTOP tren Windows (hoac thanh Taskbar)," -ForegroundColor Yellow
            Write-Host "    cho bieu tuong Docker chuyen sang mau xanh (running) roi chay lai script nay." -ForegroundColor Yellow
            exit 1
        }
        Write-Host " -> Docker Desktop da khoi dong thanh cong!" -ForegroundColor Green
    } else {
        Write-Host " -> Khong tim thay Docker Desktop tai duong dan mac dinh. Vui long bat Docker thu cong." -ForegroundColor Red
        exit 1
    }
} else {
    Write-Host " -> Docker dang chay san sang." -ForegroundColor Green
}

# 2. Khoi dong PostgreSQL va Redis qua docker-compose
Write-Host "`n[2/5] Khoi dong container Postgres & Redis qua docker-compose.infra.yml..." -ForegroundColor Yellow
docker compose -f $ComposeFile up -d postgres redis
if ($LASTEXITCODE -ne 0) {
    Write-Host " -> Loi khi khoi chay docker compose!" -ForegroundColor Red
    exit 1
}

# 3. Doi PostgreSQL san sang hoan toan
Write-Host "`n[3/5] Dang kiem tra ket noi PostgreSQL (om-postgres:5432)..." -ForegroundColor Yellow
$pgRetries = 40
$pgReady = $false
while ($pgRetries -gt 0) {
    if (Test-PostgresReady) {
        $pgReady = $true
        Write-Host " -> PostgreSQL da san sang chap nhan truy van SQL!" -ForegroundColor Green
        break
    }
    Start-Sleep -Seconds 2
    $pgRetries--
}
if (-not $pgReady) {
    Write-Host " -> Loi: Khong the ket noi toi PostgreSQL sau 80 giay." -ForegroundColor Red
    exit 1
}

# Dam bao database om_inventory_db da duoc tao
Write-Host " -> Kiem tra database om_inventory_db..." -ForegroundColor Yellow
if (-not (Test-DbExists "om_inventory_db")) {
    Write-Host " -> Dang tao database om_inventory_db..." -ForegroundColor Yellow
    docker exec -i om-postgres psql -U postgres -c "CREATE DATABASE om_inventory_db;" -v "ON_ERROR_STOP=1"
    docker exec -i om-postgres psql -U postgres -c "GRANT ALL PRIVILEGES ON DATABASE om_inventory_db TO postgres;" -v "ON_ERROR_STOP=1"
    Write-Host " -> Da tao database om_inventory_db." -ForegroundColor Green
} else {
    Write-Host " -> Database om_inventory_db da ton tai san." -ForegroundColor Green
}

# 4. Ap dung Schema Migration (dung docker cp de chong loi BOM va sai ma encoding)
Write-Host "`n[4/5] Ap dung Schema Migration (000001_init_inventory_schema.up.sql)..." -ForegroundColor Yellow
docker cp $SchemaFile om-postgres:/tmp/schema.sql
docker exec -i om-postgres psql -U postgres -d om_inventory_db -f /tmp/schema.sql -v "ON_ERROR_STOP=1"
if ($LASTEXITCODE -ne 0) {
    Write-Host " -> Loi khi nap schema migration!" -ForegroundColor Red
    exit 1
}
Write-Host " -> Nap cau truc bang thanh cong." -ForegroundColor Green

# 5. Nap Seed Data (dung docker cp de giu nguyen ven tieng Viet UTF-8)
Write-Host "`n[5/5] Nap Seed Data mau (seed_sample_data.sql)..." -ForegroundColor Yellow
docker cp $SeedFile om-postgres:/tmp/seed.sql
docker exec -i om-postgres psql -U postgres -d om_inventory_db -f /tmp/seed.sql -v "ON_ERROR_STOP=1"
if ($LASTEXITCODE -ne 0) {
    Write-Host " -> Loi khi nap seed data!" -ForegroundColor Red
    exit 1
}
Write-Host " -> Nap du lieu mau Me xung O Ma thanh cong!" -ForegroundColor Green

# Hien thi ket qua kiem tra truc quan
Write-Host "`n================ KET QUA DU LIEU TON KHO TRONG DB ================" -ForegroundColor Cyan
docker exec -i om-postgres psql -U postgres -d om_inventory_db -c "SELECT sku, physical_qty, reserved_qty, available_qty, status FROM inventory_items ORDER BY sku;"
Write-Host "`n================ DANH SACH LO HANG FEFO ================" -ForegroundColor Cyan
docker exec -i om-postgres psql -U postgres -d om_inventory_db -c "SELECT batch_code, sku, mfg_date, exp_date, physical_qty, reserved_qty, status FROM batches ORDER BY sku, exp_date ASC;"
Write-Host "`n================ PHIEU GIU CHO MAU ================" -ForegroundColor Cyan
docker exec -i om-postgres psql -U postgres -d om_inventory_db -c "SELECT r.id, r.order_id, r.status, r.expires_at, a.sku, a.allocated_qty FROM stock_reservations r JOIN stock_reservation_allocations a ON r.id = a.reservation_id;"

Write-Host "`n>>> HOAN TAT! Ban co the khoi dong Inventory Service va test qua Postman gRPC." -ForegroundColor Green
