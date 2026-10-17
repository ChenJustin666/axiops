# build-and-push.ps1 - 自动构建并推送 AxiOps Builder 镜像到阿里云
# 用法: .\build-and-push.ps1

param(
    [string]$Registry = "sinozo-registry.ap-southeast-1.cr.aliyuncs.com",
    [string]$ImageName = "platform/builder",
    [string]$Tag = "v1.0.1"
)

$ErrorActionPreference = "Stop"

$FullImageName = "${Registry}/${ImageName}:${Tag}"
$LatestImageName = "${Registry}/${ImageName}:latest"

Write-Host "=== AxiOps Builder 镜像构建脚本 ===" -ForegroundColor Cyan
Write-Host "镜像: $FullImageName" -ForegroundColor Yellow
Write-Host ""

# 检查是否在正确的目录
if (-not (Test-Path ".\Dockerfile")) {
    Write-Host "错误: 找不到 Dockerfile，请在 images/builder 目录下运行此脚本" -ForegroundColor Red
    exit 1
}

# 检查 Docker 是否可用
try {
    docker version | Out-Null
} catch {
    Write-Host "错误: Docker 未安装或未启动，请先安装 Docker Desktop" -ForegroundColor Red
    exit 1
}

# 构建镜像
Write-Host "[1/3] 构建镜像..." -ForegroundColor Green
docker build -t $FullImageName -t $LatestImageName .
if ($LASTEXITCODE -ne 0) {
    Write-Host "构建失败" -ForegroundColor Red
    exit 1
}
Write-Host "✓ 构建成功" -ForegroundColor Green
Write-Host ""

# 推送镜像 (带版本号)
Write-Host "[2/3] 推送镜像到阿里云 (${Tag})..." -ForegroundColor Green
docker push $FullImageName
if ($LASTEXITCODE -ne 0) {
    Write-Host "推送失败，请检查:" -ForegroundColor Red
    Write-Host "  1. 是否已登录阿里云镜像仓库: docker login $Registry" -ForegroundColor Yellow
    Write-Host "  2. 镜像仓库权限是否正确" -ForegroundColor Yellow
    exit 1
}
Write-Host "✓ 推送成功 ($Tag)" -ForegroundColor Green
Write-Host ""

# 推送 latest 标签
Write-Host "[3/3] 推送 latest 标签..." -ForegroundColor Green
docker push $LatestImageName
if ($LASTEXITCODE -ne 0) {
    Write-Host "警告: latest 标签推送失败，但版本标签已推送成功" -ForegroundColor Yellow
} else {
    Write-Host "✓ latest 标签推送成功" -ForegroundColor Green
}
Write-Host ""

Write-Host "=== 完成 ===" -ForegroundColor Cyan
Write-Host "镜像已推送到: $FullImageName" -ForegroundColor Green
Write-Host ""
Write-Host "下一步:" -ForegroundColor Yellow
Write-Host "  1. 打开系统管理 → 构建资源" -ForegroundColor White
Write-Host "  2. 启用平台 CI: 打开开关" -ForegroundColor White
Write-Host "  3. CI 集群: 选择 '测试集群-美国' (cluster_id=2)" -ForegroundColor White
Write-Host "  4. CI Namespace: 填 'axiops-cicd'" -ForegroundColor White
Write-Host "  5. 构建器镜像: 填 '$FullImageName'" -ForegroundColor White
Write-Host "  6. 保存后重启 workflow-worker" -ForegroundColor White
Write-Host ""
