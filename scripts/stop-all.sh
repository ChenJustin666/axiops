#!/bin/bash
# AxiOps 后端服务停止脚本

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
PROJECT_DIR=$(dirname "$SCRIPT_DIR")

echo "停止 AxiOps 后端服务..."

cd "$PROJECT_DIR"

stop_service() {
    local name=$1
    local pidfile="logs/${name}.pid"
    if [ -f "$pidfile" ]; then
        local pid=$(cat "$pidfile")
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid"
            echo "  ✓ 已停止 $name (PID: $pid)"
        else
            echo "  - $name 已不在运行"
        fi
        rm -f "$pidfile"
    else
        echo "  - $name PID 文件不存在"
    fi
}

stop_service "platform-api"
stop_service "cicd-api"

echo "完成"
