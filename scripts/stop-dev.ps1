$ErrorActionPreference = "SilentlyContinue"
$root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$stateFile = Join-Path $root "local\axiops-dev-processes.json"

Write-Host "Stopping AxiOps CI/CD local services..." -ForegroundColor Yellow

if (Test-Path $stateFile) {
  $records = Get-Content $stateFile -Raw -Encoding UTF8 | ConvertFrom-Json
  foreach ($record in $records) {
    $process = Get-Process -Id ([int]$record.pid)
    if ($null -ne $process -and $process.ProcessName -eq $record.process_name) {
      if ($record.process_tree) {
        & taskkill.exe /PID $process.Id /T /F >$null 2>$null
        if (Get-Process -Id $process.Id -ErrorAction SilentlyContinue) {
          Stop-Process -Id $process.Id -Force
        }
      } else {
        Stop-Process -Id $process.Id -Force
      }
      Write-Host "Stopped $($record.label), PID $($process.Id)."
    }
  }
  Remove-Item $stateFile -Force
}

# Also handle APIs previously started manually with `go run`. The health
# response must identify the expected AxiOps service before its port is freed.
$apis = @(
  @{ Port = 18090; Expected = "axiops-platform-api" },
  @{ Port = 8090; Expected = "axiops-platform-api" },
  @{ Port = 8091; Expected = "axiops-cicd-api" }
)
foreach ($api in $apis) {
  try {
    $response = Invoke-RestMethod "http://127.0.0.1:$($api.Port)/api/v1/system/health" -TimeoutSec 2
    if ($response.data.service -ne $api.Expected) { continue }
    $line = netstat -ano | Select-String ":$($api.Port)\s+.*LISTENING" | Select-Object -First 1
    if ($line -match "\s+(\d+)\s*$") {
      Stop-Process -Id ([int]$Matches[1]) -Force
      Write-Host "Stopped $($api.Expected) on port $($api.Port)."
    }
  }
  catch {}
}

# Clean workers orphaned by an earlier cmd.exe launch. Scope by the exact
# repository binary so unrelated workflow workers are never terminated.
$workerPath = [System.IO.Path]::GetFullPath((Join-Path $root "bin\axiops-workflow-worker.exe"))
Get-Process -Name "axiops-workflow-worker" -ErrorAction SilentlyContinue |
  Where-Object { $_.Path -eq $workerPath } |
  ForEach-Object {
    Stop-Process -Id $_.Id -Force
    Write-Host "Stopped orphaned workflow-worker, PID $($_.Id)."
  }

Write-Host "AxiOps CI/CD local services are stopped." -ForegroundColor Green
