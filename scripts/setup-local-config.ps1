$ErrorActionPreference = 'Stop'
$projectRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$examples = @(Get-ChildItem -LiteralPath (Join-Path $projectRoot 'configs') -Filter '*.example.yaml')
$examples += @(Get-ChildItem -Path (Join-Path $projectRoot 'application/*/etc/*.example.yaml') -ErrorAction SilentlyContinue)
foreach ($example in $examples) {
  $destination = $example.FullName.Replace('.example.yaml', '.yaml')
  if (!(Test-Path -LiteralPath $destination)) {
    Copy-Item -LiteralPath $example.FullName -Destination $destination
    Write-Host "Created $destination"
  }
}
Write-Host 'Edit local configuration and replace REPLACE_ME before starting services. Existing files were preserved.'
