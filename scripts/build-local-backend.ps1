$ErrorActionPreference = "Stop"

$root = Resolve-Path (Join-Path $PSScriptRoot "..")
$cacheDir = Join-Path $root ".gocache"
$binDir = Join-Path $root "bin"

New-Item -ItemType Directory -Force $cacheDir, $binDir | Out-Null
$env:GOCACHE = $cacheDir

$services = @(
  "portal-rpc",
  "portal-api",
  "manager-rpc",
  "manager-api",
  "console-rpc",
  "console-api",
  "workload-api",
  "devops-manager-rpc",
  "devops-pipeline-rpc",
  "devops-api"
)

foreach ($service in $services) {
  Write-Host "Building $service..."
  go build -buildvcs=false -o (Join-Path $binDir "$service.exe") (Join-Path $root "application\$service")
  if ($LASTEXITCODE -ne 0) { throw "Build failed: $service" }
}

$axiopsCommands = @(
  @{ Name = "axiops-platform-api"; Package = "cmd/platform-api" },
  @{ Name = "axiops-cicd-api"; Package = "cmd/cicd-api" },
  @{ Name = "axiops-workflow-worker"; Package = "cmd/workflow-worker" },
  @{ Name = "axiops-migrate"; Package = "cmd/axiops-migrate" }
)

foreach ($command in $axiopsCommands) {
  Write-Host "Building $($command.Name)..."
  go build -buildvcs=false -o (Join-Path $binDir "$($command.Name).exe") (Join-Path $root $command.Package)
  if ($LASTEXITCODE -ne 0) { throw "Build failed: $($command.Name)" }
}

Write-Host "Backend binaries written to $binDir"
