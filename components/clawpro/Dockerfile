FROM node:20-alpine

WORKDIR /app

# 复制构建产物
COPY dist/ ./dist/

# 只安装服务端运行需要的 express
RUN npm init -y && npm install express@4.21.2

# 暴露端口
EXPOSE 3000

# 设置环境变量
ENV NODE_ENV=production
ENV PORT=3000

# 启动服务
CMD ["node", "dist/index.js"]
