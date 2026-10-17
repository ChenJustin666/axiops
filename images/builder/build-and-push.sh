#!/bin/bash
# build-and-push.sh - 自动构建并推送 AxiOps Builder 镜像到阿里云
# 用法: ./build-and-push.sh

set -e

REGISTRY="${REGISTRY:-sinozo-registry.ap-southeast-1.cr.aliyuncs.com}"
IMAGE_NAME="${IMAGE_NAME:-platform/builder}"
TAG="${TAG:-v1.0.1}"

FULL_IMAGE_NAME="${REGISTRY}/${IMAGE_NAME}:${TAG}"
LATEST_IMAGE_NAME="${REGISTRY}/${IMAGE_NAME}:latest"

echo "=== AxiOps Builder 镜像构建脚本 ==="
echo "镜像: ${FULL_IMAGE_NAME}"
echo ""

# 检查是否在正确的目录
if [ ! -f "Dockerfile" ]; then
    echo "错误: 找不到 Dockerfile，请在 images/builder 目录下运行此脚本"
    exit 1
fi

# 检查 Docker 是否可用
if ! command -v docker &> /dev/null; then
    echo "错误: Docker 未安装，请先安装 Docker"
    exit 1
fi

# 构建镜像
echo "[1/3] 构建镜像..."
docker build -t "${FULL_IMAGE_NAME}" -t "${LATEST_IMAGE_NAME}" .
echo "✓ 构建成功"
echo ""

# 推送镜像 (带版本号)
echo "[2/3] 推送镜像到阿里云 (${TAG})..."
if ! docker push "${FULL_IMAGE_NAME}"; then
    echo "推送失败，请检查:"
    echo "  1. 是否已登录阿里云镜像仓库: docker login ${REGISTRY}"
    echo "  2. 镜像仓库权限是否正确"
    exit 1
fi
echo "✓ 推送成功 (${TAG})"
echo ""

# 推送 latest 标签
echo "[3/3] 推送 latest 标签..."
if ! docker push "${LATEST_IMAGE_NAME}"; then
    echo "警告: latest 标签推送失败，但版本标签已推送成功"
else
    echo "✓ latest 标签推送成功"
fi
echo ""

echo "=== 完成 ==="
echo "镜像已推送到: ${FULL_IMAGE_NAME}"
echo ""
echo "下一步:"
echo "  1. 打开系统管理 → 构建资源"
echo "  2. 启用平台 CI: 打开开关"
echo "  3. CI 集群: 选择 '测试集群-美国' (cluster_id=2)"
echo "  4. CI Namespace: 填 'axiops-cicd'"
echo "  5. 构建器镜像: 填 '${FULL_IMAGE_NAME}'"
echo "  6. 保存后重启 workflow-worker"
echo ""
