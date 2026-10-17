param(
  [string]$Source = "sql\db-init.sql",
  [string]$Output = "cache\db-init.clean.sql"
)

$ErrorActionPreference = "Stop"

$root = Resolve-Path (Join-Path $PSScriptRoot "..")
$sourcePath = Join-Path $root $Source
$outputPath = Join-Path $root $Output
$outputDir = Split-Path $outputPath -Parent

if (!(Test-Path $sourcePath)) {
  throw "Missing source SQL: $sourcePath"
}

New-Item -ItemType Directory -Force $outputDir | Out-Null

$inCreateTable = $false
$cleaned = New-Object System.Collections.Generic.List[string]
$utf8Strict = New-Object System.Text.UTF8Encoding($false, $true)

foreach ($line in [System.IO.File]::ReadLines($sourcePath, $utf8Strict)) {
  $current = $line
  $trimmed = $line.TrimStart()

  if ($trimmed.StartsWith('CREATE TABLE `')) {
    $inCreateTable = $true
    $cleaned.Add($current)
    continue
  }

  if ($inCreateTable) {
    $commentIndex = $current.IndexOf(" COMMENT ")
    if ($commentIndex -ge 0) {
      $hasComma = $current.TrimEnd().EndsWith(",")
      $current = $current.Substring(0, $commentIndex)
      if ($hasComma) {
        $current = "$current,"
      }
    }

    $tableCommentIndex = $current.IndexOf(" COMMENT=")
    if ($tableCommentIndex -ge 0) {
      $hasSemicolon = $current.Contains(";")
      $current = $current.Substring(0, $tableCommentIndex)
      if ($hasSemicolon -and !$current.TrimEnd().EndsWith(";")) {
        $current = "$current;"
      }
    }

    if ($trimmed.StartsWith(")")) {
      $inCreateTable = $false
    }
  }

  $cleaned.Add($current)
}

$utf8NoBom = New-Object System.Text.UTF8Encoding($false)
[System.IO.File]::WriteAllLines($outputPath, $cleaned, $utf8NoBom)

Write-Host "Clean SQL written to $outputPath"
