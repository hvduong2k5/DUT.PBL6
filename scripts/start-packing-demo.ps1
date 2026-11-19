param(
    [string]$Go = "go",
    [string]$Java = "java",
    [string]$Gradle = "gradle",
    [string]$OutputDirectory = ".system_generated/packing-demo"
)
$ErrorActionPreference = "Stop"
$repoRoot = Split-Path -Parent $PSScriptRoot
Push-Location -LiteralPath $repoRoot
try {
    foreach ($executable in @("docker", "python", "node", "npm", $Go, $Java, $Gradle)) {
        if (-not (Get-Command $executable -ErrorAction SilentlyContinue)) { throw "Missing executable: $executable" }
    }
    docker info --format '{{.ServerVersion}}'
    if ($LASTEXITCODE -ne 0) { throw "Docker daemon is unavailable." }
    if (-not (Test-Path -LiteralPath "apps/web-user/node_modules/next")) {
        Push-Location -LiteralPath "apps/web-user"
        try { npm ci; if ($LASTEXITCODE -ne 0) { throw "npm ci failed." } } finally { Pop-Location }
    }
    $pythonTools = Join-Path $repoRoot ".system_generated/packing-python"
    python -m pip install --target $pythonTools -r packages/contracts/mvp/requirements.txt
    if ($LASTEXITCODE -ne 0) { throw "Python dependency setup failed." }
    $previousPythonPath = $env:PYTHONPATH
    $env:PYTHONPATH = $pythonTools + [IO.Path]::PathSeparator + $previousPythonPath
    try {
        docker build -t omamx/fulfillment-service:integration services/fulfillment-service
        if ($LASTEXITCODE -ne 0) { throw "Fulfillment image build failed." }
        Write-Host "Starting isolated acceptance stack. After PASS, see $OutputDirectory/ready.json locally for the Web URL and test accounts."
        Write-Host "The fixture uses containers plus native Go/Java/Node processes; it does not deploy production."
        Write-Host "Create $OutputDirectory/stop to shut down only this fixture."
        python scripts/test-web-identity.py --go $Go --java $Java --gradle $Gradle --fulfillment --keep-running --output-dir $OutputDirectory
        if ($LASTEXITCODE -ne 0) { throw "Packing acceptance failed; inspect the local evidence directory." }
    } finally { $env:PYTHONPATH = $previousPythonPath }
} finally { Pop-Location }
