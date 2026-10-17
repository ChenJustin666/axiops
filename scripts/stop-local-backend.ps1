$names = @(
  "portal-rpc",
  "manager-rpc",
  "console-rpc",
  "portal-api",
  "manager-api",
  "workload-api",
  "console-api",
  "devops-manager-rpc",
  "devops-pipeline-rpc",
  "devops-api"
)

foreach ($name in $names) {
  Get-Process -Name $name -ErrorAction SilentlyContinue | Stop-Process -Force
}

Write-Host "Stopped local backend processes."
