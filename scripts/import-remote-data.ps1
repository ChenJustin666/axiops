param(
  [string]$HostName = "192.168.200.13",
  [string]$SshUser = "root",
  [string]$MysqlDatabase = "autoops",
  [string]$MysqlUser = "root",
  [string]$MysqlPassword = "REPLACE_ME",
  [string]$MongoDatabase = "autoops_devops",
  [string]$MongoUser = "root",
  [string]$MongoPassword = "REPLACE_ME"
)

$ErrorActionPreference = "Stop"

$root = Resolve-Path (Join-Path $PSScriptRoot "..")
$remoteDir = "/tmp/autoops-init"

Write-Host "Creating remote init directory..."
ssh "$SshUser@$HostName" "mkdir -p $remoteDir"

Write-Host "Uploading MySQL SQL and Mongo exports..."
scp (Join-Path $root "sql\db-init.sql") "$SshUser@$HostName`:$remoteDir/db-init.sql"
scp (Join-Path $root "sql\mongo-export\devops_pipeline_template.json") "$SshUser@$HostName`:$remoteDir/devops_pipeline_template.json"
scp (Join-Path $root "sql\mongo-export\devops_step_template.json") "$SshUser@$HostName`:$remoteDir/devops_step_template.json"

Write-Host "Importing MySQL database. This script contains DROP TABLE IF EXISTS statements."
ssh "$SshUser@$HostName" "mysql -u$MysqlUser -p'$MysqlPassword' -e 'CREATE DATABASE IF NOT EXISTS $MysqlDatabase DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;' && mysql -u$MysqlUser -p'$MysqlPassword' $MysqlDatabase < $remoteDir/db-init.sql"

Write-Host "Importing Mongo DevOps seed data if mongoimport is available..."
$mongoPasswordEscaped = $MongoPassword.Replace("'", "'\''")
ssh "$SshUser@$HostName" "command -v mongoimport >/dev/null 2>&1 && mongoimport --host 127.0.0.1 --port 27017 -u '$MongoUser' -p '$mongoPasswordEscaped' --authenticationDatabase admin --db '$MongoDatabase' --collection devops_pipeline_template --file $remoteDir/devops_pipeline_template.json --jsonArray --drop && mongoimport --host 127.0.0.1 --port 27017 -u '$MongoUser' -p '$mongoPasswordEscaped' --authenticationDatabase admin --db '$MongoDatabase' --collection devops_step_template --file $remoteDir/devops_step_template.json --jsonArray --drop || echo 'mongoimport not found or Mongo import failed; install MongoDB Database Tools and rerun.'"

Write-Host "Remote data import finished."
