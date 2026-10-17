param(
  [switch]$WithDevops
)

$ErrorActionPreference = "Stop"

$root = Resolve-Path (Join-Path $PSScriptRoot "..")
$logDir = Join-Path $root "logs\dev"
New-Item -ItemType Directory -Force $logDir | Out-Null

$services = @(
  @{ Name = "portal-rpc"; Exe = "bin\portal-rpc.exe"; Args = "-f config\local\portal-rpc.yaml" },
  @{ Name = "manager-rpc"; Exe = "bin\manager-rpc.exe"; Args = "-f config\local\manager-rpc.yaml" },
  @{ Name = "console-rpc"; Exe = "bin\console-rpc.exe"; Args = "-f config\local\console-rpc.yaml" },
  @{ Name = "portal-api"; Exe = "bin\portal-api.exe"; Args = "-f config\local\portal-api.yaml" },
  @{ Name = "manager-api"; Exe = "bin\manager-api.exe"; Args = "-f config\local\manager-api.yaml" },
  @{ Name = "workload-api"; Exe = "bin\workload-api.exe"; Args = "-f config\local\workload-api.yaml" },
  @{ Name = "console-api"; Exe = "bin\console-api.exe"; Args = "-f config\local\console-api.yaml" }
)

if ($WithDevops) {
  $services += @(
    @{ Name = "devops-manager-rpc"; Exe = "bin\devops-manager-rpc.exe"; Args = "-f config\local\devops-manager-rpc.yaml" },
    @{ Name = "devops-pipeline-rpc"; Exe = "bin\devops-pipeline-rpc.exe"; Args = "-f config\local\devops-pipeline-rpc.yaml" },
    @{ Name = "devops-api"; Exe = "bin\devops-api.exe"; Args = "-f config\local\devops-api.yaml" }
  )
}

$started = @()

foreach ($svc in $services) {
  $exe = Join-Path $root $svc.Exe
  if (!(Test-Path $exe)) {
    throw "Missing binary: $exe. Run go build first."
  }

  $stdout = Join-Path $logDir "$($svc.Name).out.log"
  $stderr = Join-Path $logDir "$($svc.Name).err.log"
  Write-Host "Starting $($svc.Name)..."
  $process = Start-Process -FilePath $exe `
    -ArgumentList $svc.Args `
    -WorkingDirectory $root `
    -RedirectStandardOutput $stdout `
    -RedirectStandardError $stderr `
    -WindowStyle Hidden `
    -PassThru

  $started += [pscustomobject]@{
    Name = $svc.Name
    Process = $process
    StdErr = $stderr
    StdOut = $stdout
  }

  Start-Sleep -Seconds 1
}

Start-Sleep -Seconds 2

$failed = @()
foreach ($item in $started) {
  $process = Get-Process -Id $item.Process.Id -ErrorAction SilentlyContinue
  if ($null -eq $process) {
    $failed += $item
  }
}

Write-Host "Backend startup requested. Logs: $logDir"
Write-Host "API ports: portal 8810, manager 8811, workload 8812, devops 8813, console 8818"

if ($failed.Count -gt 0) {
  Write-Warning "Some services exited shortly after startup:"
  foreach ($item in $failed) {
    Write-Warning "  - $($item.Name)"
    if (Test-Path $item.StdErr) {
      Write-Host "Last stderr lines for $($item.Name):"
      Get-Content $item.StdErr -Tail 30
    }
  }
  exit 1
}

Write-Host "All requested backend processes are still running."
