# BUCT Course Remind - Go Backend

这是一个基于 Go 和 Gin 框架的后端服务，采用模块化架构设计。

## 最近更新

本次后端更新包括以下行为调整：

- **任务 ID 与截止时间解耦**：任务 ID 由任务类型、课程、标题和内容生成。老师修改截止时间时，系统仍能识别为同一任务，避免重复任务和重复提醒；历史的截止时间 ID 会兼容迁移。
- **刷新结果保护**：刷新结果为空时不会立即删除本地作业。系统会先记录一次空结果，约 30 秒后再次确认；确认窗口为 30 分钟，确认前原有作业和完成状态都会保留。空结果会自动安排约 1 分钟后的重试。
- **刷新请求可控**：同一用户的并发刷新请求会合并为一个任务；后台使用有界 worker 和队列处理刷新，避免重复抓取和请求高峰拖垮服务。队列容量和 worker 数量可通过环境变量调整。
- **完成状态长期保留**：已完成的作业和待办不会因为过期时间被清理，也不会再发送提醒；作业状态会保留到任务从课程数据中消失或用户主动撤销完成，待办状态会保留到用户删除或撤销完成。启动时会清理旧版本遗留的完成状态过期字段。
- **提醒投递更可靠**：自动提醒按截止时间节点生成，并在节点前 5 秒发送；提醒发送使用 2 分钟处理租约和唯一领取标记，进程异常退出后可恢复超时任务；SMTP 连接、发送和写回结果均设置超时，避免发送队列长期阻塞。
- **任务删除联动清理**：课程刷新确认任务已消失后，会同时清理对应的作业完成状态，避免遗留无效记录。

这些行为依赖 MongoDB 中的状态字段和索引，部署时请让服务完成一次正常启动，以执行兼容迁移。

## 项目结构

```
go-backend-test/
├── main.go                 # 主入口文件
├── go.mod                  # Go module 文件
├── go.sum                  # Go dependencies 文件
├── README.md              # 本文档
├── .gitignore             # Git 忽略文件
├── app/                   # 应用初始化和启动逻辑
│   └── app.go
├── config/                # 配置管理模块
│   └── config.go
├── router/                # 路由管理模块
│   └── router.go
├── middleware/            # 中间件模块
│   └── middleware.go
├── models/                # 数据模型与 MongoDB Repository
│   ├── AssignmentStatus/  # 作业完成状态
│   ├── CourseData/        # 课程和作业数据、任务 ID、刷新保护
│   ├── Reminder/          # 定时提醒持久化
│   ├── Todo/              # 待办事项
│   └── User/              # 用户数据
├── api/                   # API层
│   └── handlers/          # HTTP 请求处理器
├── scheduler/             # 自动提醒调度和投递
├── services/              # 刷新队列、后台任务、提醒服务
├── scraper/               # 教务系统数据抓取
└── utils/                 # 工具函数和统一文案
```

## 模块说明

### 1. **app** - 应用初始化

- 创建Gin路由器实例
- 应用全局中间件
- 管理应用启动逻辑

### 2. **config** - 配置管理

- 加载环境变量配置
- 管理应用配置参数（端口、环境等）

### 3. **router** - 路由管理

- 统一管理所有API路由
- 支持路由分组
- 清晰的路由层次结构

### 4. **middleware** - 中间件

- 日志中间件（LoggingMiddleware）
- CORS 跨域中间件（CORSMiddleware）：仅放行同源与 `ALLOWED_ORIGINS` 白名单，不反射任意来源
- 恢复中间件（RecoveryMiddleware）
- 会话中间件（SessionMiddleware）：解析 JWT Cookie，注入用户信息
- 强制鉴权（AuthRequired）：用于 `/api/test`、`/api/admin` 等敏感路由组
- IP 限流（RateLimit）：固定窗口计数，作为兜底
- 账号失败锁定（loginguard.go）：同一账号连续 5 次登录失败锁定 15 分钟

> 为什么抗爆破依赖「账号锁定」而不是「IP 限流」：
> 校园网中大量用户共享同一 NAT 出口 IP，严格按 IP 计数会把正常用户
> 一并挡掉，而攻击者换 IP 成本极低。

### 5. **crypto** - 加密

- `RSA.go`：传输加密（PKCS#1 v1.5）。含 30 分钟密钥轮换 + 5 分钟旧密钥过渡期、
  600 秒时间戳窗口，以及**一次性密文防重放**（同一密文只能成功解密一次）
- `ECC.go`：P-256，用于加密存储学生的学校密码（`s_password`）
- `AES.go`：`EncryptAES` / `DecryptAES` 当前**无调用方**，属预留代码

### 6. **models** - 数据模型

- 用户模型 (User)
- 课程模型 (Course)
- 统一的API响应结构 (APIResponse)
- 错误响应结构 (ErrorResponse)

### 6. **api/handlers** - 请求处理器

- 处理所有HTTP请求
- 业务逻辑实现
- 用户管理处理函数
- 课程管理处理函数

### 7. **utils** - 工具函数

- 统一响应工具函数
- 简化的错误处理响应
- `defaults.go`：集中管理所有面向用户的中文提示文案

## 安全约定（改动前请先读）

1. **登录 / 注册 / 重置密码 / 收发验证码一律不接受明文。**
   前端 `createEncryptedRequest` 在加密失败时会直接抛错中止请求，
   后端对应 handler 会回 400「必须使用加密传输」。
   两侧都不能保留明文降级分支，否则攻击者只要阻断
   `/api/crypto/public-key` 就能诱导前端明文提交。

2. **`RSA_ENABLE` 必须为 true。** 设为 false 时前端会拒绝发送、
   后端也没有私钥，登录注册全部不可用。

3. **验证码校验有 5 次失败上限**（`services/vercode.go`）。
   6 位数字码若不限次，3 分钟有效期内足以被枚举，
   进而通过重置密码接管账号。达到上限后该验证码作废，需重新发送。

4. **`/api/test` 与 `/api/admin` 必须经过 `AuthRequired`。**
   `send-test-email` 会真实发信，`/api/test/config` 曾泄露 `MONGODB_URI`。

5. **ECC 密钥不可随意更换**，库中 `s_password` 依赖它解密。

## 前提条件

- Go 1.25 或更高版本
- Gin Web Framework v1.11.0

## 快速开始

### 1. 安装依赖

```bash
go mod download
```

### 2. 运行应用

```bash
go run main.go
```

应用将在 `http://localhost:5000` 启动。

### 3. 构建二进制文件

```bash
go build -o go-backend.exe
```

### 4. 运行测试

```bash
go test ./...
```

测试覆盖任务 ID 稳定性、截止时间变化、空刷新二次确认、提醒处理租约，
以及作业和待办完成状态的长期保留。

## API 路由

### 健康检查

- `GET /health`、`GET /api/health` - 检查服务器与数据库健康状态

### 认证 (`/api/auth`)

| 方法   | 路径                          | 说明                    | 需登录 |
|------|-----------------------------|-----------------------|-----|
| POST | `/api/auth/login`           | 登录（加密），签发 HttpOnly Cookie | 否   |
| POST | `/api/auth/register`        | 注册（加密）                | 否   |
| POST | `/api/auth/logout`          | 登出，清除 Cookie           | 否   |
| GET  | `/api/auth/status`          | 查询登录状态                | 否   |
| GET  | `/api/auth/user-info`       | 获取当前用户信息              | 是   |
| POST | `/api/auth/send-verification-code` | 发送验证码（加密）       | 否   |
| POST | `/api/auth/verify-code`     | 校验验证码（加密），5 次失败上限     | 否   |
| POST | `/api/auth/reset-password`  | 重置密码（加密）              | 否   |
| POST | `/api/auth/check-email`     | 查询邮箱是否已注册             | 否   |

### 业务接口

- `/api/assignments/*` - 作业列表、完成/删除/恢复、手动提醒
- `/api/courses/*` - 课程数据（`/api/course-data/list` 等）
- `/api/todos/*` - 待办事项
- `/api/reminders`、`/api/webhooks/*` - 提醒
- `/api/settings`、`/api/stats`、`/api/blacklist` - 设置、统计、黑名单
- `/api/crypto/public-key`、`/api/crypto/status` - 加密公钥与状态
- `/api/test/*` - **需登录**，调试用
- `/api/admin/*` - **需登录 + 管理员**，运维用

> 所有接受密码、验证码的接口都只接受 `{"encrypted_data": "<RSA 密文>"}`，
> 传明文一律返回 400。

## 环境变量

| 变量                        | 说明                          | 默认值           |
|---------------------------|-----------------------------|---------------|
| `PORT`                    | 服务器监听地址                     | `:5000`       |
| `ENV`                     | 运行环境，`development` 时开启 Debug | `development` |
| `MONGODB_URI`             | MongoDB 连接串（含库名）             | 无，必填          |
| `SECRET_KEY`              | JWT 签名密钥                     | 无，必填          |
| `ECC_PRIVATE_KEY`         | P-256 私钥（十进制）                | 无，必填          |
| `ECC_PUBLIC_KEY`          | P-256 公钥 x 坐标（十进制）           | 无，必填          |
| `RSA_ENABLE`              | 是否启用传输加密，必须为 `true`          | `true`        |
| `RSA_KEY_SIZE`            | RSA 密钥位数                     | `2048`        |
| `RSA_KEY_EXPIRE_MINUTES`  | RSA 密钥轮换周期（分钟）               | `30`          |
| `ALLOWED_ORIGINS`         | CORS 白名单，逗号分隔；留空仅放行同源        | 空             |
| `SNOWFLAKE_NODE`          | 雪花 ID 节点号（0-1023），多实例必须不同    | `1`           |
| `MAIL_SMTP_SERVER`        | SMTP 服务器                    | 无             |
| `MAIL_SMTP_PORT`          | SMTP 端口                     | 无             |
| `MAIL_SENDER`             | 发件邮箱                        | 无             |
| `MAIL_PASSWORD`           | 邮箱授权码                       | 无             |
| `VERIFY_CODE_EXPIRE`      | 验证码有效期（秒）                   | `180`         |
| `REFRESH_WORKERS`         | 刷新后台 worker 数量                 | `2`           |
| `REFRESH_QUEUE_SIZE`      | 刷新队列容量                         | `128`         |

## 依赖

- **gin-gonic/gin** (v1.11.0) - Go Web 框架

## 开发指南

### 添加新路由

1. 在 `api/handlers/handlers.go` 中添加处理函数
2. 在 `router/router.go` 中定义路由

```go
// 在handlers.go中
func NewHandler(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

// 在router.go中
v1.GET("/new", handlers.NewHandler)
```

### 添加新中间件

在 `middleware/middleware.go` 中创建新的中间件函数，然后在 `app/app.go` 中应用。

### 使用响应工具函数

```go
// 成功响应
utils.SuccessResponseOK(c, "获取成功", data)

// 错误响应
utils.ErrorResponseBadRequest(c, "参数错误")
utils.ErrorResponseNotFound(c, "资源不存在")
utils.ErrorResponseServerError(c, "服务器内部错误")
```

## 项目扩展建议

1. **数据库名集中管理** - 目前仍有约 17 处硬编码 `client.Database("buct-course")`，
   可改用 `config.GetDBName()`（已实现但未被调用），便于通过 `MONGODB_URI` 切换库名。

2. **AES 代码清理** - `crypto/AES.go` 的 `EncryptAES`/`DecryptAES` 无任何调用方，
   且 `/api/crypto/status` 依据 `AES_KEY` 是否设置来上报状态，
   容易让人误以为存在 AES 加密链路。建议删除，或真正接入后同步更新状态接口。

3. **RSA 升级为 OAEP** - 当前使用 PKCS#1 v1.5（确定性加密）。
   已有一次性密文防重放兜底，但迁移到 OAEP 可以从根本上消除重放面；
   需前后端同步改动（JSEncrypt 不支持 OAEP，需换 `node-forge` 或 WebCrypto）。

4. **登录锁定持久化** - `middleware/loginguard.go` 目前是进程内内存计数，
   重启后清零，多实例部署时也不共享。如需更强的抗爆破能力可落库或接 Redis。

5. **日志系统** - 集成 `logrus` 或 `zap` 替换标准库 `log`。

## 许可证

MIT License

## 贡献

欢迎提交 PR 和 Issue！



