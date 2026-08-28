#!/bin/bash
set -e

# ============================================================
# 服务器首次初始化脚本（在目标服务器上执行一次即可）
# 作用：安装 Node.js 20 + PM2、准备 /opt/openclaw-enterprise 目录、首次拉起服务
#
# 日常部署请使用 skill：openclaw-deploy.skill 中的 scripts/deploy.sh
# 部署目标已迁移到 DevCloud：clawprodesign.devcloud.woa.com:36000
# ============================================================

echo "=== [1/5] 安装 Node.js 20.x ==="
# 清理旧版本
rm -rf /usr/local/lib/nodejs
yum remove -y nodejs 2>/dev/null || true

# 下载并安装 Node.js 20 LTS
cd /tmp
if [ ! -f "node-v20.18.0-linux-x64.tar.xz" ]; then
    curl -fsSL https://nodejs.org/dist/v20.18.0/node-v20.18.0-linux-x64.tar.xz -o node-v20.18.0-linux-x64.tar.xz
fi
mkdir -p /usr/local/lib/nodejs
tar -xJf node-v20.18.0-linux-x64.tar.xz -C /usr/local/lib/nodejs

# 创建符号链接
ln -sf /usr/local/lib/nodejs/node-v20.18.0-linux-x64/bin/node /usr/local/bin/node
ln -sf /usr/local/lib/nodejs/node-v20.18.0-linux-x64/bin/npm /usr/local/bin/npm
ln -sf /usr/local/lib/nodejs/node-v20.18.0-linux-x64/bin/npx /usr/local/bin/npx

# 验证
node --version
npm --version

echo "=== [2/5] 安装 PM2 ==="
npm install -g pm2
ln -sf /usr/local/lib/nodejs/node-v20.18.0-linux-x64/bin/pm2 /usr/local/bin/pm2

echo "=== [3/5] 准备应用目录 ==="
mkdir -p /opt/openclaw-enterprise
cd /opt/openclaw-enterprise

echo "=== [4/5] 启动应用 ==="
# 停止旧进程
pm2 delete openclaw 2>/dev/null || true

# 用 PM2 启动
PORT=80 NODE_ENV=production pm2 start dist/index.js --name openclaw

# 保存 PM2 进程列表并设置开机自启
pm2 save
pm2 startup systemd -u root --hp /root 2>/dev/null || true

echo "=== [5/5] 验证 ==="
sleep 2
pm2 status
curl -sI http://localhost:80 | head -5

echo ""
echo "✅ 初始化完成！日常部署请使用 openclaw-deploy.skill 中的 scripts/deploy.sh"
