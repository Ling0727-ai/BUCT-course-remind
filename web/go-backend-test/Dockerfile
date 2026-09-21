# ──────────────────────────────────────────────
# 阶段一：构建
# ──────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

# 镜像源可按环境覆盖：
#   本机（中国大陆）：用默认值，国内源更快
#   GitHub Actions：runner 在 Azure 美国节点，官方源反而快 10-20 倍
#                   由 workflow 传 --build-arg 覆盖
# 实测（Azure westus3）：dl-cdn 0.22s / ustc 2.85s / aliyun 0.88s
ARG APK_MIRROR=mirrors.ustc.edu.cn
ARG GOPROXY=https://goproxy.cn,direct

# 安装构建依赖
#   只需 git：GOPROXY 的 direct 回退靠它拉取依赖
#   不需要 gcc/musl-dev：下面用 CGO_ENABLED=0 编译纯静态二进制，
#   省掉 gcc 及其依赖（binutils/gmp/mpfr4/isl26 等约 200MB）可大幅缩短构建时间
# 挂载 apk 缓存：包列表变化时复用已下载的 .apk（故不再用 --no-cache）
RUN --mount=type=cache,target=/var/cache/apk,sharing=locked \
    sed -i "s|dl-cdn.alpinelinux.org|${APK_MIRROR}|g" /etc/apk/repositories && \
    apk add git

WORKDIR /build

# 先只复制依赖文件，利用 Docker 缓存层
# 只要 go.mod/go.sum 不变，下次构建直接用缓存，不重新下载依赖
COPY go.mod go.sum ./

# 配置 GOPROXY（默认国内镜像）
RUN go env -w GOPROXY=${GOPROXY} && \
    go mod download

# 复制全部源码
COPY . .

# 编译：
#   CGO_ENABLED=0  禁用 cgo，产出纯静态二进制
#   -ldflags="-w -s"  去掉调试信息和符号表，减小体积
#   -o /app/server  输出到 /app/server
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /app/server \
    .

# ──────────────────────────────────────────────
# 阶段二：运行
# 用 alpine 而不是 scratch，方便 curl 健康检查和 shell 调试
# ──────────────────────────────────────────────
FROM alpine:3.20 AS runner

# ARG 作用域限于单个构建阶段，runner 阶段需重新声明
ARG APK_MIRROR=mirrors.ustc.edu.cn

# 安装运行时依赖
#   ca-certificates  HTTPS 请求（发邮件、调外部 API）必需
#   curl             健康检查
#   tzdata           时区支持（北京时间）
#   haveged          熵守护进程：防止 alpine 容器内 crypto/rand 因熵不足而阻塞
#                    （RSA 2048 密钥生成需要大量随机熵）
# 复用 apk 缓存，理由同 builder 阶段
RUN --mount=type=cache,target=/var/cache/apk,sharing=locked \
    sed -i "s|dl-cdn.alpinelinux.org|${APK_MIRROR}|g" /etc/apk/repositories && \
    apk add ca-certificates curl tzdata haveged su-exec && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

# 创建非 root 用户
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/server .
RUN mkdir -p /app/logs && chown -R appuser:appgroup /app

COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

# 不在此处切换用户，由 entrypoint 在启动 haveged(root) 后 su-exec 到 appuser
# USER appuser

# 对应 Python 版本的 MONGO_URI 默认值格式
# 注意：不在此处写死连接串（含账号密码），由 docker-compose / .env 注入
ENV PORT=:5000
ENV ENV=production

EXPOSE 5000

# 健康检查，对应 Python 版本的 /api/health
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:5000/api/health || exit 1

ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["/app/server"]

