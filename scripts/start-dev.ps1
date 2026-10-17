param(
  [switch]$SkipBuild,
  [switch]$NoWeb,
  [ValidateRange(1, 65535)][int]$PlatformPort = 18090,
  [ValidateRange(1, 65535)][int]$CicdPort = 8091
)

$ErrorActionPreference = "Stop"
# Keep Go output and PowerShell diagnostics UTF-8 so Chinese messages are not
# decoded with the legacy Windows code page in the console or log viewer.
$utf8 = New-Object System.Text.UTF8Encoding($false)
[Console]::OutputEncoding = $utf8
$OutputEncoding = $utf8
try { & chcp.com 65001 | Out-Null } catch { }
$root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$bin = Join-Path $root "bin"
$runtime = Join-Path $root "local\dev"
$stateFile = Join-Path $root "local\axiops-dev-processes.json"
Set-Location $root
New-Item -ItemType Directory -Force $bin, $runtime | Out-Null
$env:GOCACHE = Join-Path $root ".gocache"
$env:GOMODCACHE = Join-Path $root ".gomodcache"
$env:VITE_PLATFORM_API_TARGET = "http://127.0.0.1:$PlatformPort"
$env:VITE_CICD_API_TARGET = "http://127.0.0.1:$CicdPort"

function Test-PortFree([int]$port) {
  return $null -eq (Get-NetTCPConnection -LocalPort $port -State Listen -ErrorAction SilentlyContinue)
}

if (!(Test-PortFree $PlatformPort)) {
  throw "Platform API port $PlatformPort is already in use. Stop the existing AxiOps process or pass -PlatformPort with a free port."
}
if (!(Test-PortFree $CicdPort)) {
  throw "CI/CD API port $CicdPort is already in use. Stop the existing AxiOps process or pass -CicdPort with a free port."
}
if (!$NoWeb -and !(Test-PortFree 3006)) {
  throw "Frontend port 3006 is already in use. Stop the existing AxiOps Vite process before starting the local stack."
}

& .\scripts\init-local-secrets.ps1
if (!$SkipBuild) {
  $targets = @(
    @{ Name = "axiops-migrate"; Package = "axiops-migrate" },
    @{ Name = "axiops-platform-api"; Package = "platform-api" },
    @{ Name = "axiops-cicd-api"; Package = "cicd-api" },
    @{ Name = "axiops-workflow-worker"; Package = "workflow-worker" }
  )
  foreach ($target in $targets) {
    go build -buildvcs=false -o "$bin\$($target.Name).exe" ".\cmd\$($target.Package)"
    if ($LASTEXITCODE -ne 0) { throw "Build failed: $($target.Name)" }
  }
}

Write-Host "Applying ordered database migrations..." -ForegroundColor Cyan
& "$bin\axiops-migrate.exe" -f configs\platform-api.yaml
if ($LASTEXITCODE -ne 0) { throw "Database migration failed" }

$specs = @(
  @{ Label = "platform-api"; File = "$bin\axiops-platform-api.exe"; Args = @("-f", "configs\platform-api.yaml", "-host", "127.0.0.1", "-port", "$PlatformPort") },
  # Device Guard may block the locally built CI/CD binary; Go run is the
  # compatible development fallback and uses the same source/configuration.
  @{ Label = "cicd-api"; File = "go.exe"; Args = @("run", ".\cmd\cicd-api", "-f", "configs\cicd-api.yaml", "-host", "127.0.0.1", "-port", "$CicdPort") },
  # Same development launch path as the API: newly built unsigned binaries
  # can be blocked by local Device Guard even after a successful Go build.
  @{ Label = "workflow-worker"; File = "go.exe"; Args = @("run", ".\cmd\workflow-worker", "-f", "configs\workflow-worker.yaml") }
)
if (!$NoWeb) {
  $vite = Join-Path $root "axiops-web\node_modules\vite\bin\vite.js"
  if (!(Test-Path $vite)) { throw "Frontend dependencies are missing; run corepack pnpm install in axiops-web" }
  $specs += @{ Label = "web"; File = (Get-Command node.exe).Source; Args = @($vite, "--host", "0.0.0.0", "--configLoader", "runner"); WorkingDirectory = (Join-Path $root "axiops-web") }
}

$records = foreach ($spec in $specs) {
  # Some Windows Application Control policies reject direct PowerShell launches
  # of locally built Go executables; cmd.exe is the compatible launch boundary.
  $argText = (($spec.Args | ForEach-Object { '"{0}"' -f $_ }) -join ' ')
  $command = 'call "{0}" {1}' -f $spec.File, $argText
  $workingDirectory = if ($spec.WorkingDirectory) { $spec.WorkingDirectory } else { $root }
  $process = Start-Process -FilePath $env:ComSpec -ArgumentList @('/d', '/c', $command) -WorkingDirectory $workingDirectory `
    -RedirectStandardOutput "$runtime\$($spec.Label).out.log" `
    -RedirectStandardError "$runtime\$($spec.Label).err.log" -WindowStyle Hidden -PassThru
  [pscustomobject]@{ label = $spec.Label; pid = $process.Id; process_name = $process.ProcessName; process_tree = $true }
}
$records | ConvertTo-Json | Set-Content -Encoding UTF8 $stateFile
$deadline = (Get-Date).AddSeconds(60)
foreach ($endpoint in @(
  @{ Name = "platform API"; Port = $PlatformPort; Service = "axiops-platform-api" },
  @{ Name = "CI/CD API"; Port = $CicdPort; Service = "axiops-cicd-api" }
)) {
  $healthy = $false
  while ((Get-Date) -lt $deadline) {
    try {
      $health = Invoke-RestMethod "http://127.0.0.1:$($endpoint.Port)/api/v1/system/health" -TimeoutSec 3
      if ($health.code -eq 0 -and $health.data.service -eq $endpoint.Service) { $healthy = $true; break }
    } catch {
      Start-Sleep -Seconds 1
    }
  }
  if (!$healthy) { throw "$($endpoint.Name) health check timed out on port $($endpoint.Port); inspect local\\dev logs" }
}
$workerReady = $false
while ((Get-Date) -lt $deadline) {
  $workerLog = Join-Path $runtime 'workflow-worker.err.log'
  if ((Test-Path $workerLog) -and (Select-String -LiteralPath $workerLog -Pattern 'AxiOps workflow worker started:' -Quiet)) {
    $workerReady = $true
    break
  }
  Start-Sleep -Seconds 1
}
if (!$workerReady) { throw "Workflow worker did not start; inspect local\dev\workflow-worker.err.log" }
if (!$NoWeb) {
  $webHealthy = $false
  while ((Get-Date) -lt $deadline) {
    try {
      Invoke-WebRequest "http://127.0.0.1:3006" -UseBasicParsing -TimeoutSec 3 | Out-Null
      $platformProxy = Invoke-RestMethod "http://127.0.0.1:3006/api/platform/v1/system/health" -TimeoutSec 3
      $cicdProxy = Invoke-RestMethod "http://127.0.0.1:3006/api/cicd/v1/system/health" -TimeoutSec 3
      if ($platformProxy.code -eq 0 -and $platformProxy.data.service -eq "axiops-platform-api" -and
          $cicdProxy.code -eq 0 -and $cicdProxy.data.service -eq "axiops-cicd-api") {
        $webHealthy = $true
        break
      }
    } catch {
      Start-Sleep -Seconds 1
    }
  }
  if (!$webHealthy) { throw "Frontend health check timed out on port 3006; inspect local\\dev logs" }
}
Write-Host "AxiOps is ready: http://127.0.0.1:3006 (admin / REPLACE_ME)" -ForegroundColor Green
Write-Host "Logs: $runtime"
