#!/bin/bash

# 一键部署脚本
# Usage: ./deploy.sh

set -e

echo "🚀 开始部署加密货币趋势分析系统..."
echo ""

# 检查 Docker
if ! command -v docker &> /dev/null; then
    echo "❌ Docker 未安装，请先安装 Docker"
    exit 1
fi

# 检查 Docker Compose
if ! command -v docker-compose &> /dev/null; then
    echo "❌ Docker Compose 未安装，请先安装 Docker Compose"
    exit 1
fi

echo "✅ Docker 环境检查通过"
echo ""

# 停止旧容器
echo "🛑 停止旧容器..."
docker-compose down 2>/dev/null || true
echo ""

# 构建镜像
echo "🔨 构建 Docker 镜像..."
docker-compose build --no-cache
echo ""

# 启动服务
echo "🚀 启动服务..."
docker-compose up -d
echo ""

# 等待服务启动
echo "⏳ 等待服务启动..."
sleep 10

# 检查服务状态
echo "🔍 检查服务状态..."
docker-compose ps
echo ""

# 健康检查
echo "💊 健康检查..."
if curl -f http://localhost:8888/api/health > /dev/null 2>&1; then
    echo "✅ 后端服务正常"
else
    echo "⚠️  后端服务可能未就绪，请检查日志"
fi

if curl -f http://localhost:9999 > /dev/null 2>&1; then
    echo "✅ 前端服务正常"
else
    echo "⚠️  前端服务可能未就绪，请检查日志"
fi

echo ""
echo "🎉 部署完成！"
echo ""
echo "📊 访问地址:"
echo "   前端: http://localhost:9999"
echo "   后端: http://localhost:8888"
echo ""
echo "📝 查看日志:"
echo "   docker-compose logs -f"
echo ""
echo "🛑 停止服务:"
echo "   docker-compose stop"
echo ""
