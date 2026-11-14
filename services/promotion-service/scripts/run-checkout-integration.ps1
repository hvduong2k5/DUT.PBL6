param([switch]$KeepStack, [switch]$SkipOrderRegressions)
$ErrorActionPreference = 'Stop'
$serviceRoot = Split-Path $PSScriptRoot -Parent
$repoRoot = (Resolve-Path (Join-Path $serviceRoot '..\..')).Path
$testOutput = Join-Path $repoRoot '.tmp\promotion-checkout'
New-Item -ItemType Directory -Force -Path $testOutput | Out-Null
$composeFile = Join-Path $serviceRoot 'docker-compose.checkout-test.yml'
$testToken = 'promotion-checkout-local-integration-token'
$database = 'postgres://checkout_test:checkout_test@127.0.0.1:15447/promotion_checkout_test?sslmode=disable'
$testEnvironment = @{
  PROMOTION_INTERNAL_TOKEN = $testToken
  PROMOTION_CHECKOUT_TEST_TOKEN = $testToken
  PROMOTION_DATABASE_URL = $database
  PROMOTION_TEST_DATABASE_URL = $database
  PROMOTION_HTTP_ADDRESS = '127.0.0.1:18017'
  PROMOTION_GRPC_ADDRESS = '127.0.0.1:19017'
  PROMOTION_ORDER_EVENTS_ENABLED = 'true'
  PROMOTION_KAFKA_BROKERS = '127.0.0.1:19047'
  ORDER_TEST_PROMOTION_GRPC = '127.0.0.1:19017'
  ORDER_CORE_TEST_DATABASE = $database
  ORDER_TEST_DATABASE = $database
  ORDER_TEST_SIM_DATABASE = $database
  ORDER_TEST_REDIS = '127.0.0.1:16347'
  ORDER_TEST_KAFKA = '127.0.0.1:19047'
}
$oldEnvironment = @{}
$promotionProcess = $null
Push-Location $repoRoot
try {
  foreach ($name in $testEnvironment.Keys) {
    $oldEnvironment[$name] = [Environment]::GetEnvironmentVariable($name, 'Process')
    [Environment]::SetEnvironmentVariable($name, $testEnvironment[$name], 'Process')
  }
  # Docker writes normal progress to stderr. In Windows PowerShell 5.1 this
  # must not become a terminating NativeCommandError when output is redirected.
  $ErrorActionPreference = 'Continue'
  docker compose -p promotion-checkout-test -f $composeFile up -d --wait --wait-timeout 180 2>&1 | ForEach-Object { $_.ToString() }
  $composeExit = $LASTEXITCODE
  $ErrorActionPreference = 'Stop'
  if ($composeExit -ne 0) { throw 'Checkout test dependencies failed to start.' }
  foreach ($topic in @('order.events.v3', 'payment.events.v1', 'promotion.events.v1', 'promotion.order.v3.dlq')) {
    $ErrorActionPreference = 'Continue'
    docker compose -p promotion-checkout-test -f $composeFile exec -T checkout-kafka /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --create --if-not-exists --topic $topic --partitions 1 --replication-factor 1 2>&1 | ForEach-Object { $_.ToString() }
    $topicExit = $LASTEXITCODE
    $ErrorActionPreference = 'Stop'
    if ($topicExit -ne 0) { throw "Failed to initialize test topic $topic." }
  }
  Push-Location $serviceRoot
  try {
    go build -o (Join-Path $testOutput 'promotion-service.exe') ./cmd/server
    if ($LASTEXITCODE -ne 0) { throw 'Promotion build failed.' }
  } finally { Pop-Location }
  $promotionProcess = Start-Process -FilePath (Join-Path $testOutput 'promotion-service.exe') -WindowStyle Hidden -PassThru -RedirectStandardOutput (Join-Path $testOutput 'promotion.stdout.log') -RedirectStandardError (Join-Path $testOutput 'promotion.stderr.log')
  $ready = $false
  for ($attempt = 0; $attempt -lt 40; $attempt++) {
    if ($promotionProcess.HasExited) { throw 'Promotion stopped; inspect .tmp/promotion-checkout logs.' }
    try { $ready = (Invoke-WebRequest -UseBasicParsing 'http://127.0.0.1:18017/health/ready' -TimeoutSec 2).StatusCode -eq 204 } catch { $ready = $false }
    if ($ready) { break }
    Start-Sleep -Milliseconds 500
  }
  if (-not $ready) { throw 'Promotion did not become ready.' }
  Push-Location $serviceRoot
  try {
    go test ./... -count=1 -timeout=3m
    if ($LASTEXITCODE -ne 0) { throw 'Promotion tests failed.' }
  } finally { Pop-Location }
  Push-Location (Join-Path $repoRoot 'services\order-service')
  try {
    if (-not $SkipOrderRegressions) {
      go test ./internal/persistence -count=1 -timeout=5m
      if ($LASTEXITCODE -ne 0) { throw 'Order PostgreSQL regressions failed.' }
    }
    go test ./tests/integration -run '^TestPromotionCheckout' -count=1 -timeout=5m -v
    if ($LASTEXITCODE -ne 0) { throw 'Order/Promotion checkout integration failed.' }
  } finally { Pop-Location }
} finally {
  if ($null -ne $promotionProcess -and -not $promotionProcess.HasExited) { Stop-Process -Id $promotionProcess.Id }
  if (-not $KeepStack) {
    $ErrorActionPreference = 'Continue'
    docker compose -p promotion-checkout-test -f $composeFile down --volumes 2>&1 | ForEach-Object { $_.ToString() }
    $ErrorActionPreference = 'Stop'
  }
  foreach ($name in $oldEnvironment.Keys) { [Environment]::SetEnvironmentVariable($name, $oldEnvironment[$name], 'Process') }
  Pop-Location
}
