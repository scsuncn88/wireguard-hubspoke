#!/bin/bash

# WireGuard SD-WAN GitHub Push Script
# 使用说明：
# 1. 先在GitHub上创建新仓库（不要初始化README）
# 2. 将下面的YOUR_USERNAME替换为您的GitHub用户名
# 3. 将REPOSITORY_NAME替换为您创建的仓库名
# 4. 运行此脚本

echo "🚀 WireGuard SD-WAN GitHub Push Script"
echo "======================================"

# 检查是否在正确的目录
if [ ! -f "go.mod" ] || [ ! -f "README.md" ]; then
    echo "❌ 请在项目根目录运行此脚本"
    exit 1
fi

# 检查是否有未提交的更改
if ! git diff --quiet; then
    echo "⚠️  检测到未提交的更改，正在提交..."
    git add .
    git commit -m "docs: 添加GitHub设置指南和推送脚本"
fi

# 设置变量（请修改这些值）
GITHUB_USERNAME="YOUR_USERNAME"
REPOSITORY_NAME="wg-hubspoke"

echo "请输入您的GitHub用户名："
read -r GITHUB_USERNAME

echo "请输入仓库名称 (默认: wg-hubspoke)："
read -r REPO_INPUT
if [ -n "$REPO_INPUT" ]; then
    REPOSITORY_NAME="$REPO_INPUT"
fi

# 构建远程仓库URL
REMOTE_URL="https://github.com/${GITHUB_USERNAME}/${REPOSITORY_NAME}.git"

echo "📋 配置信息："
echo "   GitHub用户名: ${GITHUB_USERNAME}"
echo "   仓库名称: ${REPOSITORY_NAME}"
echo "   远程URL: ${REMOTE_URL}"
echo ""

# 检查是否已经添加了远程仓库
if git remote get-url origin >/dev/null 2>&1; then
    echo "🔄 更新远程仓库URL..."
    git remote set-url origin "$REMOTE_URL"
else
    echo "➕ 添加远程仓库..."
    git remote add origin "$REMOTE_URL"
fi

# 确保主分支名称为main
echo "🔧 设置主分支为main..."
git branch -M main

# 推送代码
echo "📤 推送代码到GitHub..."
if git push -u origin main; then
    echo ""
    echo "✅ 成功推送到GitHub!"
    echo "🔗 仓库地址: https://github.com/${GITHUB_USERNAME}/${REPOSITORY_NAME}"
    echo ""
    echo "📋 接下来的步骤："
    echo "1. 访问您的GitHub仓库"
    echo "2. 查看GITHUB_SETUP.md了解更多配置选项"
    echo "3. 考虑设置CI/CD和发布版本"
    echo "4. 添加项目标签和描述"
else
    echo ""
    echo "❌ 推送失败！"
    echo "请检查："
    echo "1. GitHub仓库是否已创建"
    echo "2. 用户名和仓库名是否正确"
    echo "3. 是否有权限推送到该仓库"
    echo "4. 网络连接是否正常"
    echo ""
    echo "💡 您也可以手动运行以下命令："
    echo "   git remote add origin ${REMOTE_URL}"
    echo "   git branch -M main"
    echo "   git push -u origin main"
fi