# 使用官方 Node.js 镜像作为基础镜像
FROM node:20-alpine

# npm 源可按环境覆盖：
#   本机（中国大陆）：默认 npmmirror
#   GitHub Actions：runner 在 Azure 美国节点，官方源快约 8 倍
# 实测（Azure westus3）：registry.npmjs.org 0.17s / npmmirror 1.27s
ARG NPM_REGISTRY=https://registry.npmmirror.com

# 设置工作目录
WORKDIR /app

# 复制 package.json 和 package-lock.json
COPY package*.json ./

# 安装依赖
RUN npm config set registry ${NPM_REGISTRY} && npm install --legacy-peer-deps

# 复制源代码
COPY . .

# 构建应用
RUN npm run build

# 使用 nginx 来服务静态文件
FROM nginx:alpine

# 复制构建的文件到 nginx 目录
COPY --from=0 /app/dist /usr/share/nginx/html

# 复制 nginx 配置
COPY nginx.conf /etc/nginx/conf.d/default.conf

# 移除 10-listen-on-ipv6-by-default.sh：
# 该脚本用 `apk manifest nginx` 校验 default.conf 是否为官方原版，
# 而 apk manifest 默认要联网读取包索引。在国内网络下会阻塞约 30 秒才超时，
# 导致容器启动后 nginx 迟迟不监听端口。
# 我们本来就用自定义配置覆盖了 default.conf，这个校验没有意义。
RUN rm -f /docker-entrypoint.d/10-listen-on-ipv6-by-default.sh

# 暴露端口
EXPOSE 80

# 启动 nginx
CMD ["nginx", "-g", "daemon off;"]