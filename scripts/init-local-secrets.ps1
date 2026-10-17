param(
  [switch]$Rotate
)

$ErrorActionPreference = "Stop"
$projectRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$localDirectory = Join-Path $projectRoot "local"
$keyFile = Join-Path $localDirectory "credential-encryption.key"

if ((Test-Path -LiteralPath $keyFile) -and -not $Rotate) {
  Write-Host "AxiOps local credential encryption key already exists: $keyFile"
  exit 0
}

New-Item -ItemType Directory -Force -Path $localDirectory | Out-Null
$random = New-Object byte[] 48
$generator = [System.Security.Cryptography.RandomNumberGenerator]::Create()
try {
  $generator.GetBytes($random)
}
finally {
  $generator.Dispose()
}

$key = [Convert]::ToBase64String($random)
[System.IO.File]::WriteAllText($keyFile, $key, [System.Text.UTF8Encoding]::new($false))

Write-Host "AxiOps local credential encryption key initialized: $keyFile"
Write-Host "Keep this file stable. Rotating it requires re-encrypting all stored credentials."
