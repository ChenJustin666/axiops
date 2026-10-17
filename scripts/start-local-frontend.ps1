$ErrorActionPreference = "Stop"

$root = Resolve-Path (Join-Path $PSScriptRoot "..")
$webRoot = Join-Path $root "axiops-web"

if (!(Test-Path $webRoot)) {
  throw "Missing frontend directory: $webRoot"
}

Push-Location $webRoot
try {
  $env:COREPACK_HOME = Join-Path $webRoot ".corepack"
  if (!(Test-Path (Join-Path $webRoot "node_modules"))) {
    corepack.cmd pnpm install
  }
  corepack.cmd pnpm dev
}
finally {
  Pop-Location
}
