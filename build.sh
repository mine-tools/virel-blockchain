#!/bin/bash

# Virel Blockchain 构建脚本
# 支持指定输出文件名和目录

set -e  # 遇到错误时退出

# 默认配置
OUTPUT_DIR="bin"
NODE_NAME="virel-node"
WALLET_NAME="virel-wallet-cli"
VERSION="v3.1.11"

# 解析命令行参数
while [[ $# -gt 0 ]]; do
    case $1 in
        -o|--output)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        -n|--node-name)
            NODE_NAME="$2"
            shift 2
            ;;
        -w|--wallet-name)
            WALLET_NAME="$2"
            shift 2
            ;;
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        -h|--help)
            echo "用法: $0 [选项]"
            echo "选项:"
            echo "  -o, --output DIR     输出目录 (默认: bin)"
            echo "  -n, --node-name NAME 节点程序名称 (默认: virel-node)"
            echo "  -w, --wallet-name NAME 钱包程序名称 (默认: virel-wallet-cli)"
            echo "  -v, --version VER    版本号 (默认: v3.1.11)"
            echo "  -h, --help           显示帮助信息"
            exit 0
            ;;
        *)
            echo "未知参数: $1"
            echo "使用 -h 或 --help 查看帮助"
            exit 1
            ;;
    esac
done

echo "=== Virel Blockchain 构建脚本 ==="
echo "输出目录: $OUTPUT_DIR"
echo "节点程序: $NODE_NAME"
echo "钱包程序: $WALLET_NAME"
echo "版本: $VERSION"
echo ""

# 创建输出目录
mkdir -p "$OUTPUT_DIR"

# 构建节点程序
echo "构建节点程序..."
if go build -o "$OUTPUT_DIR/$NODE_NAME-$VERSION" ./cmd/virel-node/; then
    echo "✅ 节点程序构建成功: $OUTPUT_DIR/$NODE_NAME-$VERSION"
else
    echo "❌ 节点程序构建失败"
    exit 1
fi

# 构建钱包程序
echo "构建钱包程序..."
if go build -o "$OUTPUT_DIR/$WALLET_NAME-$VERSION" ./cmd/virel-wallet-cli/; then
    echo "✅ 钱包程序构建成功: $OUTPUT_DIR/$WALLET_NAME-$VERSION"
else
    echo "❌ 钱包程序构建失败"
    exit 1
fi

# 创建符号链接（可选）
echo "创建符号链接..."
ln -sf "$NODE_NAME-$VERSION" "$OUTPUT_DIR/$NODE_NAME"
ln -sf "$WALLET_NAME-$VERSION" "$OUTPUT_DIR/$WALLET_NAME"

echo ""
echo "=== 构建完成 ==="
echo "生成的文件:"
ls -la "$OUTPUT_DIR"/virel-*

echo ""
echo "使用方法:"
echo "  ./$OUTPUT_DIR/$NODE_NAME --help"
echo "  ./$OUTPUT_DIR/$WALLET_NAME --help"
