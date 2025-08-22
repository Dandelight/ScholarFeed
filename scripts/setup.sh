#!/bin/bash

# reno-arxiv 项目设置脚本

set -e

echo "🚀 开始设置 reno-arxiv 项目..."

# 检查Go版本
if ! command -v go &> /dev/null; then
    echo "❌ Go 未安装，请先安装 Go 1.23+"
    exit 1
fi

GO_VERSION=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | cut -d'o' -f2)
REQUIRED_VERSION="1.23"

if ! printf '%s\n%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V -C; then
    echo "❌ Go 版本太低，需要 $REQUIRED_VERSION+，当前版本: $GO_VERSION"
    exit 1
fi

echo "✅ Go 版本检查通过: $GO_VERSION"

# 检查MySQL
if ! command -v mysql &> /dev/null; then
    echo "⚠️  MySQL 未安装，请确保MySQL服务可用"
    echo "   你可以使用 Docker: make mysql-start"
fi

# 创建环境配置文件
if [ ! -f .env ]; then
    echo "📝 创建环境配置文件..."
    cp .env.example .env
    echo "✅ 环境配置文件已创建，请编辑 .env 文件填入你的配置"
else
    echo "✅ 环境配置文件已存在"
fi

# 安装Go依赖
echo "📦 安装Go依赖..."
go mod tidy
go mod download
echo "✅ Go依赖安装完成"

# 创建必要目录
echo "📁 创建必要目录..."
mkdir -p reports
mkdir -p logs
echo "✅ 目录创建完成"

# 构建项目
echo "🔨 构建项目..."
go build -o arxiv-server cmd/server/main.go
go build -o arxiv-cli cmd/cli/main.go
echo "✅ 项目构建完成"

# 检查配置文件
echo "🔍 检查配置..."
if [ -f .env ]; then
    # 检查关键配置是否存在
    if grep -q "ANTHROPIC_API_KEY=" .env || grep -q "OPENAI_API_KEY=" .env; then
        echo "✅ AI API配置检查通过"
    else
        echo "⚠️  请在 .env 文件中配置AI API密钥"
    fi
    
    if grep -q "DB_PASSWORD=" .env; then
        echo "✅ 数据库配置检查通过"
    else
        echo "⚠️  请在 .env 文件中配置数据库密码"
    fi
fi

echo ""
echo "🎉 项目设置完成！"
echo ""
echo "📚 快速开始："
echo "  1. 编辑 .env 文件，填入你的配置"
echo "  2. 启动MySQL数据库服务"
echo "  3. 运行服务器: make run 或 ./arxiv-server"
echo "  4. 测试API: make health"
echo ""
echo "💡 有用的命令："
echo "  make help        - 查看所有可用命令"
echo "  make dev-start   - 启动完整开发环境"
echo "  make trigger-today - 手动触发今日总结"
echo "  ./arxiv-cli -help - 查看CLI工具帮助"
echo ""
echo "🔗 API地址: http://localhost:8080"
echo "📖 健康检查: http://localhost:8080/health"