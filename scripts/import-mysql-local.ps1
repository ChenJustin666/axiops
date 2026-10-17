param(
  [string]$HostName = "192.168.200.13",
  [int]$Port = 3306,
  [string]$Database = "autoops",
  [string]$User = "root",
  [string]$Password = "REPLACE_ME"
)

$ErrorActionPreference = "Stop"

$root = Resolve-Path (Join-Path $PSScriptRoot "..")
$sqlPath = Join-Path $root "sql\db-init.sql"

if (!(Test-Path $sqlPath)) {
  throw "Missing SQL file: $sqlPath"
}

$mysql = Get-Command mysql -ErrorAction SilentlyContinue
if ($null -eq $mysql) {
  throw "mysql command not found. Install MySQL client or run this script on a machine with mysql in PATH."
}

$mysqlPasswordArg = "-p$Password"

Write-Host "Creating database $Database on $HostName..."
& $mysql.Source -h $HostName -P $Port -u $User $mysqlPasswordArg --default-character-set=utf8mb4 -e "CREATE DATABASE IF NOT EXISTS $Database DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

if ($LASTEXITCODE -ne 0) {
  throw "Failed to create database $Database."
}

$sourcePath = $sqlPath.Replace("\", "/")
Write-Host "Importing $sqlPath into $Database. This will recreate tables declared in the SQL file..."
& $mysql.Source -h $HostName -P $Port -u $User $mysqlPasswordArg --default-character-set=utf8mb4 $Database -e "source $sourcePath"

if ($LASTEXITCODE -ne 0) {
  throw "Failed to import SQL into $Database."
}

Write-Host "Checking required tables..."
& $mysql.Source -h $HostName -P $Port -u $User $mysqlPasswordArg --default-character-set=utf8mb4 -N -e "SELECT table_name FROM information_schema.tables WHERE table_schema='$Database' AND table_name IN ('sys_menu','sys_role','sys_user') ORDER BY table_name; SELECT COUNT(*) AS sys_menu_count FROM $Database.sys_menu; SELECT COUNT(*) AS sys_role_count FROM $Database.sys_role; SELECT COUNT(*) AS sys_user_count FROM $Database.sys_user;"

if ($LASTEXITCODE -ne 0) {
  throw "Import finished but validation query failed."
}

Write-Host "MySQL import and validation finished."
