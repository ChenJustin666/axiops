param([string]$BinaryName = 'axiops-cicd-api-service-assets.exe')

$ErrorActionPreference = 'Stop'
$taskRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$taskBin = (Resolve-Path (Join-Path $taskRoot 'bin')).Path
$taskExe = [IO.Path]::GetFullPath((Join-Path $taskBin $BinaryName))
if ([IO.Path]::GetDirectoryName($taskExe) -ne $taskBin -or !(Test-Path -LiteralPath $taskExe)) {
  throw 'Choose an existing executable directly inside the workspace bin directory.'
}
$listeners = @(Get-NetTCPConnection -LocalPort 8091 -State Listen -ErrorAction SilentlyContinue)
foreach ($listenerId in @($listeners.OwningProcess | Select-Object -Unique)) {
  $running = Get-Process -Id $listenerId -ErrorAction Stop
  if ([IO.Path]::GetDirectoryName($running.Path) -ne $taskBin -or $running.ProcessName -notlike 'axiops-cicd-api*') {
    throw 'Port 8091 belongs to another application; refusing to stop it.'
  }
  Stop-Process -Id $listenerId -ErrorAction Stop
  $running.WaitForExit(10000) | Out-Null
}
$taskLogs = Join-Path $taskRoot 'local\dev'
$stamp = Get-Date -Format 'yyyyMMdd-HHmmss'
$stdout = Join-Path $taskLogs "cicd-api.$stamp.out.log"
$stderr = Join-Path $taskLogs "cicd-api.$stamp.err.log"
# A separate hidden console avoids coupling the API lifetime to a transient
# interactive terminal. Never start a worker as a side effect of an API restart.
$apiProcess = Start-Process -FilePath $taskExe -ArgumentList @('-f', 'configs\cicd-api.yaml') `
  -WorkingDirectory $taskRoot -WindowStyle Hidden -RedirectStandardOutput $stdout `
  -RedirectStandardError $stderr -PassThru
$healthy = $false
for ($attempt = 0; $attempt -lt 20; $attempt++) {
  $apiProcess.Refresh()
  if ($apiProcess.HasExited) { throw "API exited ($($apiProcess.ExitCode)); inspect $stderr" }
  try {
    $health = Invoke-RestMethod 'http://127.0.0.1:8091/api/v1/system/health' -TimeoutSec 2
    if ($health.code -eq 0 -and $health.data.service -eq 'axiops-cicd-api') { $healthy = $true; break }
  } catch { }
  Start-Sleep -Milliseconds 500
}
if (!$healthy) { throw "API did not become healthy; inspect $stderr" }
$stateFile = Join-Path $taskRoot 'local\axiops-dev-processes.json'
$records = @()
if (Test-Path -LiteralPath $stateFile) {
  $records = @(Get-Content -Raw -Encoding UTF8 -LiteralPath $stateFile | ConvertFrom-Json | Where-Object { $_.label -ne 'cicd-api' })
}
$records += [pscustomobject]@{label='cicd-api';pid=$apiProcess.Id;process_name=$apiProcess.ProcessName;process_tree=$true}
$records | ConvertTo-Json -Depth 4 | Set-Content -Encoding UTF8 -LiteralPath $stateFile
Write-Output "CI/CD API healthy, PID=$($apiProcess.Id); worker unchanged. Logs: $stdout, $stderr"
