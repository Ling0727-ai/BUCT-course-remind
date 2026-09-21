# BUCT课程提醒系统

一个基于 **Go (Gin)** + MongoDB + Vue.js 的课程作业提醒系统，支持自动获取教务系统信息并发送通知。

> 后端为 Go (Gin) 单一实现。早期版本曾提供 Python (Flask) 后端，因存在内存泄漏问题已移除（见下方说明）。

## 功能特性

- 🔐 用户注册登录系统（RSA 强制加密传输）
- 📚 学生信息管理（学号、外部系统密码 ECC 加密存储）
- 📧 邮箱验证码验证 + 忘记密码重置
- 🔔 邮件提醒（手动 / 定时 / DDL 前 24h 自动提醒）
- 📝 待办事项管理（含优先级、软删除、回收站）
- ⚙️ 灵活的系统设置
- 📱 响应式前端界面（蓝色主色调，支持深色模式）
- 🚀 Go 高性能后端（镜像体积 ~20MB，启动 <1s）

## 更新日志

**2025年9到10月版本更新**：

- v0.0.0 - 初始版本发布，包含用户注册、学生信息管理、设置页面等核心功能。
- v1.0.0 - 完善爬虫库，支持自动化获取作业信息与测试信息。
- v1.1.0 - 增加todo list，规划后续功能开发方向。
- v1.2.0 - 优化前端界面，提升用户体验。
- v2.0.0 - 从实时获取数据改为数据库暂存，提升系统稳定性和性能，增加已完成以及删除标记。
- v2.1.0 - 增加密码强度检测功能，增加前端登录和注册时的表单加密，完善反爬措施，提升账户安全性。
- v2.2.0 - 增加手动刷新按钮，方便用户即时获取最新信息。
- v2.3.0 - 优化数据库结构，提升数据存取效率。
- v2.4.0 - 修复删除时的bug，改为软删除+硬删除双标记,增加回收站，提升系统稳定性。

2025年11月版本更新：

- v3.0.0 - 重构前端样式，改为清新明快的蓝色主色调配色方案，提升视觉体验；同时优化移动端的使用体验。(2025-11-11)
- v3.1.0 - 初步实现邮箱提醒功能，支持用户通过邮箱接收课程作业提醒。(2025-11-13)
- v3.2.0 - 邮箱自动化提醒功能完善，支持定时发送课程作业提醒邮件（ddl24h前自动发送）。(2025-11-14)
- v3.3.0 - 修复了一系列bug并美化界面，提升系统稳定性和用户体验。(2025-11-15 3:00 AM)
    1. 修复刷新数据库后已经删除的作业/测试重新出现的bug
    2. 修复提醒功能不生效的bug
    3. 修复更新教务系统账号密码不写入数据库的bug
    4. 重构了设置todo的时间设置格式，使其更加自由
    5. 优化了提醒设置界面的时间设置，优化页面样式
    6. 重构表单结构，提升代码可维护性，提升安全性
    7. 修复刷新数据库无弹窗提示的bug
    8. 重构数据生成逻辑，提升系统稳定性
    9. 优化邮箱提醒逻辑，减少不必要的邮件发送（已完成以及已删除标记的）
    10. 删除功能重复的字段
    11. 发现部分作业/测试抓取失败的bug，待修复
- v3.3.1 - 修复作业/测试抓取失败bug（其实是修复我的另一个项目https://github.com/Ling0727-ai/python-buct-course.git）
  （2025-11-15 13:08）
- v3.4.0 - 修复了一系列弹窗失效bug，以及滑动条异常显示问题，提升用户体验。（2025-11-15 16:40）
    1. 修复设置页面双滚动条问题，优化页面滚动体验
    2. 修复作业和测试类型显示统一为"作业"的bug，现可正确区分作业和测试
    3. 修复登录自动刷新异常问题：
        - 修正logout方法缺少user_id参数导致的会话清理失败
        - 消除会话资源泄漏，防止客户端对象累积
        - 增强异常处理和日志追踪机制
    4. 重构待办事项时间设置界面：
        - 改用日期时间选择器设置DDL（截止时间）
        - 添加快捷时间选项（1小时、3小时、6小时、12小时、1天、3天、1周）
        - 参考提醒设置界面的设计，提供更直观的用户体验
        - 移除原有的时长输入方式，改为具体截止日期时间设置
- v3.4.1 - 增加6小时未操作自动登出功能，提升账户安全性。 (2025-11-16)
- v3.4.2 - 完善作业DDL前24小时自动邮件提醒功能。 (2025-11-17)
    1. 自动检测即将到期的作业（截止前23-25小时范围内）
    2. 为每个用户的即将到期作业自动创建提醒
    3. 提醒邮件包含科目、标题、截止时间等详细信息
    4. 智能过滤已完成和已删除的作业，避免重复提醒
    5. 调度器每小时自动检查一次，无需手动干预
    6. 修复重新登录后立即切换页面误触发6小时自动退出的问题
- v3.4.3 - 改进登录和密码找回功能，提升用户体验。 (2025-11-17)
    1. 忘记密码邮件中包含用户名信息，解决用户同时忘记用户名和密码的问题
    2. 支持使用用户名或邮箱登录，提供更灵活的登录方式
    3. 优化登录错误提示，更清晰地指出问题所在
    4. 密码重置页面实时显示用户名提示
- v3.4.4 - 强制加密传输，提升系统安全性。 (2025-11-17)
    1. 移除所有接口的兼容模式，强制要求使用RSA加密传输
    2. 登录、注册、发送验证码、重置密码等接口必须使用加密数据
    3. 未使用加密的请求将被拒绝（返回400错误）
    4. 符合OWASP安全标准，防止中间人攻击和网络嗅探

**2026年3月版本更新**：

- v4.0.0 - 完整 Go (Gin) 后端实现，与 Python 版本功能完全对齐。(2026-03-06)
  1. 使用 Go 1.25 + Gin 重写全部后端接口（63 条路由，覆盖全部 Python 蓝图）
  2. 多阶段 Docker 构建，最终镜像体积 ~20MB（Python 版 ~300MB）
  3. 采用分层架构：handler → service → model/repository
  4. Snowflake 分布式 ID 生成，替代 MongoDB ObjectID
  5. goroutine + time.Ticker 实现调度器，替代 APScheduler
  6. 原生 `net/smtp` 实现多候选 SMTP 自动重试发送
  7. 新增 Docker Compose 一键部署（`docker-compose-go.yml`）

**2026年9月版本更新**：

- v4.1.0 - 前端可用性与可访问性重构。(2026-09-21)
  1. **修复深色模式**：卡片透明度原先直接覆写 `--bg` 内联变量，导致深色预设失效、白底浅灰字（对比度约 1.2:1）。改为 `--surface-rgb` + `--card-alpha` 组合，深浅色均达到 WCAG AA。
  2. **修复 XSS 风险**：Toast 原先用 `innerHTML` 拼接含后端数据的文本，改为 Vue 组件渲染；同时移除 `PreviewModal` 的 `v-html`。
  3. **统一 Toast**：删除登录/注册/找回密码各自实现的三套 Toast，合并为全局 `ToastHost`，支持堆叠、悬停暂停、错误用 `role="alert"`。
  4. **弹窗规范化**：新增 `BaseModal`，统一对话框语义（`role="dialog"` / `aria-modal`）、Esc 关闭、Tab 焦点陷阱、打开聚焦与关闭后焦点归还、页面滚动锁；消除 7 份重复的弹窗 CSS。
  5. **键盘可访问性**：全站补齐 `:focus-visible` 焦点环；卡片改为可聚焦并支持回车打开；所有图标按钮补 `aria-label`；表单控件关联 `<label>`。
  6. **移动端**：恢复页面缩放能力（移除 `user-scalable=no`）；触控目标放大到 44px 并加大间距；头部按钮在窄屏收进"更多"菜单。
  7. **首页信息架构**：卡片去掉固定 220px 最小高度、改用 CSS 行数截断；新增紧凑视图切换；首屏改骨架屏、后台刷新不再整页遮挡；筛选状态同步到 URL 便于分享；空结果提供"清除筛选"出口；统计卡改为按钮并补齐移动端。
  8. **内网部署**：Font Awesome 由 cdnjs 改为本地打包，内网无外网时图标不再丢失；新增首屏加载态与 `<noscript>` 提示。
  9. **工程**：路由改为懒加载并新增 404 兜底；清理约 2900 行死代码（两份重复 Modal/SecurityIndicator、未引用的 `public/styles.css`、调试页、从未生效的 canvas 粒子服务、重复的 `vue.config.ts`）；修复 ESLint 无法解析 TypeScript 的配置问题。

## 项目结构

```
BUCT-course-remind/
├── web/
│   ├── go-backend-test/        # Go (Gin) 后端
│   │   ├── api/handlers/       # Handler 层（接口定义）
│   │   │   ├── auth.go
│   │   │   ├── assignments.go
│   │   │   ├── todos.go
│   │   │   ├── reminder.go
│   │   │   ├── course_data.go
│   │   │   ├── admin.go
│   │   │   ├── settings.go
│   │   │   └── ...
│   │   ├── services/           # 业务逻辑层
│   │   ├── models/             # 数据层（按集合分包）
│   │   │   ├── User/
│   │   │   ├── CourseData/
│   │   │   ├── AssignmentStatus/
│   │   │   ├── Todo/
│   │   │   └── Reminder/
│   │   ├── scheduler/          # 定时调度器
│   │   ├── scraper/            # 教务系统爬虫
│   │   ├── config/             # 配置 + 邮件
│   │   ├── crypto/             # RSA / AES / ECC 加密
│   │   ├── router/router.go    # 路由注册
│   │   ├── middleware/         # 日志 / CORS / Recovery
│   │   ├── Dockerfile
│   │   └── main.go
│   │
│   ├── fronted/                # Vue.js 前端
│   │   └── src/
│   │       ├── views/
│   │       └── components/
│   │
│   └── docker-compose-go.yml   # Go 后端一键部署
└── README.md
```

## 安装和运行

### 方式一：Docker 一键部署（推荐）

#### 使用 Go 后端

```bash
cd web

# 复制环境变量模板并填入真实值（模板不含任何密钥）
cp .env.template .env

# 启动（MongoDB + Go后端 + 前端）
docker compose -f docker-compose-go.yml up -d

# 查看日志
docker compose -f docker-compose-go.yml logs -f backend
```

服务启动后访问：

- 前端：`http://localhost:3033`
- Go 后端：`http://localhost:5000`

#### 使用 CI 构建好的镜像

推送代码或打 tag 后，GitHub Actions 会自动构建并推送到 GHCR：

| 镜像 | 地址 |
|---|---|
| 后端 | `ghcr.io/ling0727-ai/buct-course-remind-backend` |
| 前端 | `ghcr.io/ling0727-ai/buct-course-remind-frontend` |

可用 tag：`latest`（main 分支）、`main`、`sha-xxxxxxx`、语义化版本（打 `v*` tag 时）。
镜像为 **public**，拉取无需登录。

```bash
docker pull ghcr.io/ling0727-ai/buct-course-remind-backend:latest
docker pull ghcr.io/ling0727-ai/buct-course-remind-frontend:latest
```

#### 使用 Python 后端（已移除）

> Python (Flask) 后端因存在内存泄漏问题已从仓库移除，请使用 Go 后端。
> 如需查看历史实现：见旧仓库 `KMT-CN/BUCT-course-remind` 的 `web/backend/`。

#### 环境变量说明（`.env`）

仓库只提供 `web/.env.template`，不含任何真实值。首次部署：

```bash
cd web
cp .env.template .env
# 然后编辑 .env 填入真实值
```

`.env` 已在 `.gitignore` 中忽略，**不会被提交**。

##### 自动生成密钥

`SECRET_KEY` / `AES_KEY` / ECC 密钥对可用脚本生成，避免手工填错：

```powershell
# Windows（推荐）：生成并直接写入 web/.env
pwsh web/scripts/gen-secrets.ps1 -OutFile web/.env

# 只看结果不写文件
pwsh web/scripts/gen-secrets.ps1
```

```bash
# Linux / macOS
bash web/scripts/gen-secrets.sh
```

生成后仍需手工填写 `MAIL_SENDER` 和 `MAIL_PASSWORD`（邮箱授权码，脚本无法生成）。

> ⚠️ ECC 密钥一旦生成就不要随意更换：库中 `s_password` 是用它加密的，
> 换密钥会导致已有数据无法解密，必须同步迁移。

##### 通过 GitHub Secrets 下发配置

密钥可存在 GitHub 仓库 Secrets 中，由工作流生成 `.env` 供下载，**不进入仓库、也不烘焙进镜像**。

**1. 把本地 `.env` 批量推送到 Secrets**

```powershell
# 先看会写入哪些键（不实际写入）
pwsh web/scripts/set-github-secrets.ps1 -EnvFile web/.env -Repo <owner>/<repo> -WhatIf

# 实际写入
pwsh web/scripts/set-github-secrets.ps1 -EnvFile web/.env -Repo <owner>/<repo>
```

**2. 需要时生成 `.env`**

在 Actions 页面手动触发 **Deploy Config** 工作流，产物在 Artifacts 中（保留 1 天）。

##### 为什么不把密钥烘焙进 Docker 镜像

| 原因 | 说明 |
|---|---|
| 镜像公开 | 本仓库与 GHCR 包均为 public，镜像内任何内容都可匿名拉取 |
| 无法删除 | 镜像层一旦写入就永久留在 `docker history` 中 |
| 前端更危险 | `VUE_APP_*` 会被编译进 JS，等于把密钥发给每个访客 |
| 无法轮换 | 改密钥要重新构建推送整个镜像 |

因此密钥只在**运行时**由 compose 注入。构建期不需要任何密钥（后端只编译 Go，前端无 `VUE_APP_*` 依赖）。

```dotenv
# MongoDB（必填）
MONGO_INITDB_ROOT_USERNAME=your_mongo_username
MONGO_INITDB_ROOT_PASSWORD=your_mongo_password
MONGO_INITDB_DATABASE=buct-course

# 邮件（必填，否则邮件功能不可用）
MAIL_SMTP_SERVER=smtp.163.com
MAIL_SMTP_PORT=465
MAIL_SENDER=your_email@163.com
MAIL_PASSWORD=your_auth_code        # 邮箱授权码，非登录密码

# JWT 密钥（生产环境务必修改）
SECRET_KEY=change-this-in-production

# ECC 密钥（必填，用于加密存储学生的学校密码）
# 生成后请妥善保管；更换密钥必须同步迁移库中已加密的 s_password，否则无法解密
ECC_PRIVATE_KEY=<你的 P-256 私钥十进制值>
ECC_PUBLIC_KEY=<对应的公钥 x 坐标>

# 端口（可选，默认值如下）
GO_PORT=5000
FRONTEND_PORT=3033
```

---

### 方式二：本地开发运行

#### Go 后端

```bash
cd web/go-backend-test

# 配置环境变量
cp .env.example .env   # 或直接设置系统环境变量

# 下载依赖
go mod download

# 运行
go run main.go
```

#### 前端

```bash
cd web/fronted
npm install
npm run serve
```

---

## API 接口

> 所有路由前缀均为 `/api`。

### 认证

| 方法   | 路径                             | 说明           |
|------|--------------------------------|--------------|
| POST | `/auth/login`                  | 用户登录（RSA 加密） |
| POST | `/auth/logout`                 | 用户登出         |
| POST | `/auth/register`               | 用户注册（RSA 加密） |
| GET  | `/auth/status`                 | 登录状态检查       |
| GET  | `/auth/user-info`              | 获取当前用户信息     |
| POST | `/auth/update-email`           | 修改邮箱         |
| POST | `/auth/update-student-info`    | 更新学号/教务密码    |
| POST | `/auth/check-email`            | 检查邮箱是否已注册    |
| POST | `/auth/send-verification-code` | 发送邮箱验证码      |
| POST | `/auth/verify-code`            | 校验验证码        |
| POST | `/auth/reset-password`         | 重置密码（RSA 加密） |

### 作业

| 方法     | 路径                                  | 说明        |
|--------|-------------------------------------|-----------|
| GET    | `/assignments`                      | 获取作业列表    |
| GET    | `/assignments/stats`                | 统计信息      |
| GET    | `/assignments/completed`            | 已完成 ID 列表 |
| GET    | `/assignments/deleted`              | 回收站列表     |
| DELETE | `/assignments/clear-deleted`        | 清空回收站     |
| POST   | `/assignments/:id/complete`         | 标记完成      |
| POST   | `/assignments/:id/uncomplete`       | 撤销完成      |
| POST   | `/assignments/:id/delete`           | 软删除       |
| POST   | `/assignments/:id/restore`          | 恢复        |
| DELETE | `/assignments/:id/permanent-delete` | 永久删除      |
| POST   | `/assignments/:id/remind`           | 创建提醒      |

### 课程数据

| 方法   | 路径                     | 说明       |
|------|------------------------|----------|
| POST | `/course-data/refresh` | 手动触发爬虫刷新 |
| GET  | `/course-data/list`    | 获取课程数据列表 |
| GET  | `/course-data/status`  | 查看上次刷新时间 |

### 待办事项

| 方法     | 路径                            | 说明              |
|--------|-------------------------------|-----------------|
| GET    | `/todos`                      | 待办列表            |
| POST   | `/todos`                      | 创建待办            |
| GET    | `/todos/stats`                | 统计              |
| GET    | `/todos/deleted`              | 回收站             |
| DELETE | `/todos/clear-deleted`        | 清空回收站           |
| PUT    | `/todos/:id`                  | 更新              |
| POST   | `/todos/:id/complete`         | 标记完成（12h 后自动删除） |
| POST   | `/todos/:id/uncomplete`       | 撤销完成            |
| POST   | `/todos/:id/remind`           | 发送提醒邮件          |
| POST   | `/todos/:id/delete`           | 软删除             |
| POST   | `/todos/:id/restore`          | 恢复              |
| DELETE | `/todos/:id/permanent-delete` | 永久删除            |

### 提醒

| 方法     | 路径                  | 说明     |
|--------|---------------------|--------|
| GET    | `/reminders`        | 提醒列表   |
| POST   | `/reminders`        | 创建定时提醒 |
| DELETE | `/reminders/:id`    | 删除提醒   |
| POST   | `/reminders/manual` | 立即手动发送 |
| POST   | `/reminders/test`   | 发送测试邮件 |

### 设置 / 其他

| 方法       | 路径                             | 说明                |
|----------|--------------------------------|-------------------|
| GET/POST | `/settings`                    | 系统设置              |
| GET/POST | `/settings/email`              | 邮件通知设置            |
| GET      | `/health`                      | 健康检查（含 DB + 邮件状态） |
| GET      | `/stats`                       | 全局统计              |
| GET/POST | `/crypto/public-key` 等         | RSA/AES 加密接口      |
| POST     | `/admin/cleanup/trigger`       | 管理员手动触发刷新         |
| GET      | `/admin/completed-assignments` | 管理员查看完成记录         |
| GET      | `/admin/system/status`         | 系统状态              |

---

## 架构说明

### Go 后端分层

```
HTTP 请求
  → middleware（日志 / CORS / Recovery）
  → router（URL 分发）
  → handlers（解析请求 + 返回响应）
  → services（跨模型业务逻辑）
  → models/[集合名]/database.go（MongoDB 操作）
```

### 调度器

Go 后端使用 `goroutine + time.Ticker` 实现三个并发定时任务：

| 任务     | 间隔    | 说明                              |
|--------|-------|---------------------------------|
| 课程数据刷新 | 12 小时 | 自动爬取教务系统，更新数据库                  |
| 到期提醒检查 | 5 秒   | 发送 `scheduled_reminders` 中到期的邮件 |
| 自动提醒生成 | 1 小时  | 为 DDL 前 24h 内的作业自动创建提醒          |

### 后端技术指标

| 指标          | Go (Gin)               |
|-------------|------------------------|
| Docker 镜像大小 | ~20MB                  |
| 容器启动时间      | <1s                    |
| 并发模型        | goroutine              |
| 定时任务        | time.Ticker            |
| 路由前缀        | `/api`                 |
| 对外端口        | `5000`                 |

---

## 环境要求

| 组件    | 版本           |
|-------|--------------|
| 后端运行时 | Go 1.25+     |
| 数据库   | MongoDB 4.0+ |
| 前端    | Node.js 14+  |

---

## 贡献

欢迎提交 Issue 和 Pull Request 来改进这个项目。

## TODO list

- [x] JWT 鉴权中间件完善 — 已用 HttpOnly Cookie + JWT 实现
- [ ] Go 后端单元测试覆盖

## 许可证

MIT License
