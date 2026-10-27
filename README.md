# BUCT课程提醒系统

一个基于 **Go (Gin)** + MongoDB + Vue.js 的课程作业提醒系统，支持自动获取教务系统信息并发送通知。

> 后端为 Go (Gin) 单一实现。早期版本曾提供 Python (Flask) 后端，因存在内存泄漏问题已移除（见下方说明）。

## 功能特性

- 🔐 用户注册登录系统（RSA 强制加密传输）
- 📚 学生信息管理（学号、外部系统密码 ECC 加密存储）
- 📧 邮箱验证码验证 + 忘记密码重置
- 🔔 邮件提醒（手动 / 定时 / DDL 前 24h、12h、1h 三档自动提醒）
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

**2026年10月版本更新**：

- v4.2.0 - 按用户独立调度与前后端性能优化。(2026-10-26)
  1. **独立刷新调度**：自动刷新由每 12 小时一轮调整为每个用户成功刷新后 3 小时；通过 `next_refresh_at` 索引查找到期账号，使用单次定时器按最近到期时间唤醒。定时器支持秒级唤醒，实际执行时间取决于队列和上游响应速度。
  2. **统一刷新队列**：登录、手动和定时刷新共用有界队列，同一用户的并发请求合并；默认 2 个 worker、128 个等待名额，可通过 `REFRESH_WORKERS` 和 `REFRESH_QUEUE_SIZE` 配置。移除每用户固定等待 2 秒的逻辑，抓取不再阻塞提醒调度；队列满时同步接口返回 HTTP 503。
  3. **课程增量同步**：只写入新增或变化的任务，成功写入后再清理上游已移除的任务及其待发送提醒。无变化时仅更新用户刷新时间，空结果也会记录更新时间；抓取失败不会作为空列表保存，新增数据写入失败时保留旧任务。
  4. **自动提醒按到期任务查询**：新增标准日期字段和到期索引，替代每小时遍历全部用户和全部作业；每批最多处理 200 条到期任务，批量读取账号、完成状态和已有提醒，并通过稳定提醒 ID 保证新建自动提醒的幂等性。
  5. **提醒发送与日期兼容**：最多同时执行 2 个邮件发送任务，发送前检查作业或待办是否已完成、删除；无时区的教务时间按北京时间解析。旧作业日期字段在后台分批迁移，迁移失败可在后续调度中重试。
  6. **首页读取与渲染优化**：列表和统计共享按用户缓存的摘要快照，上限为 128 个用户、估算 8 MiB、有效期 10 秒，课程、状态和黑名单写入后立即失效。详情正文改为打开预览时按需加载，移除额外的已完成状态请求，并复用分类和统计计算结果。
  7. **减少锁竞争与内存增长**：爬虫按用户串行保护会话，登出操作移出全局缓存锁；IP 限流改为 32 个分片并限制条目数量，分批清理过期记录；账号登录保护避免容量达到阈值后每次请求都遍历整个缓存。
  8. **账号查询索引**：为用户名、邮箱增加唯一索引；旧数据存在重复值时保留原数据、回退到普通查询索引，并在启动日志中提示处理重复值。
  9. **回归验证**：新增刷新请求合并、队列容量、缓存失效、增量写入失败保护、提醒用户隔离等测试，通过 Go 竞态检测、静态检查及前端检查和生产构建。

- v4.2.1 - 补齐三档自动提醒与秒级计划时间。(2026-10-26)
  1. 每条作业分别安排截止前 24 小时、12 小时、1 小时三个提醒任务，按各自计划时间唤醒，后续提醒无需等待课程刷新。
  2. 每档使用独立提醒 ID，兼容旧的单次 24 小时提醒；旧作业自动迁移并补齐剩余节点。
  3. 迟发现或停机恢复时仅补发当前有效节点，跳过已被后续节点替代的提醒及已截止任务，避免连续补发多封邮件。完成或删除的任务不发送，恢复任务后重新安排尚有效的提醒。
  4. 邮件标明提醒节点，并在实际发送前更新剩余时间；发送失败可重试，计划时间精确到秒，实际投递仍可能受队列和邮件服务延迟影响。

- v4.2.2 - 自动提醒增加 5 秒发送提前量。(2026-10-26)
  1. 三档自动提醒的计划发送时间统一提前 5 秒，用于补偿部分排队、网络和邮件服务延迟；首档发送前额外预留 30 秒生成任务，提前量不保证实际到达时间。
  2. 相邻提醒节点的失效边界同步提前，避免旧节点与新节点同时补发。
  3. 后台分批调整已有的待发送自动提醒，已发送及手动提醒不受影响；通过迁移标记和条件更新避免重复提前。

- v4.2.3 - 稳定任务身份、刷新结果和完成状态。(2026-10-27)
  1. 任务 ID 不再包含截止时间；截止时间变化会沿用原任务及其完成状态，历史截止时间 ID 兼容迁移，避免重复任务和重复提醒。
  2. 刷新结果为空时先进入复核流程，约 30 秒后再次确认，确认前保留原有作业；空结果会安排约 1 分钟后的重试。
  3. 已完成的作业和待办长期保留，不因旧的过期字段被清理，也不会发送提醒；作业消失或待办被删除时再清理对应状态。
  4. 提醒投递增加 2 分钟处理租约、唯一领取标记和 SMTP 超时，进程异常退出后可恢复超时任务。
  5. 启动时清理旧版本遗留的完成状态过期字段，并为提醒处理租约建立索引。

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

### 方式一：使用已发布的镜像部署（最快）

CI 会把镜像推送到 GitHub Container Registry（GHCR），可直接拉取，
无需在服务器上装 Go / Node，也无需克隆仓库。

> **镜像是公开的，无需登录即可拉取。**
>
> 仓库与 GHCR 包（package）是两个独立对象：包最初默认是 private，
> 需要单独把可见性改为 public。本项目已改为 public，匿名 `docker pull` 可用。
>
> 若遇到 `denied`：先确认包可见性，再排查本机残留的失效凭据——
> docker 会优先使用已登录的凭据而不是匿名访问，凭据过期就会一直 `denied`。
> 执行 `docker logout ghcr.io` 清除后重试即可。
>
> ⚠️ 容器包的可见性**一旦改为 public 就不能改回 private**。

镜像地址（注意 owner 全小写）：

```
ghcr.io/ling0727-ai/buct-course-remind-backend:latest
ghcr.io/ling0727-ai/buct-course-remind-frontend:latest
```

#### 可选：同时发布到 Docker Hub（docker.io）

仓库的 GitHub Actions 支持一次构建，同时推送到 GHCR 和 Docker Hub，两个仓库使用相同的镜像标签。
在 GitHub 仓库 **Settings → Secrets and variables → Actions → Secrets** 中添加：

| Repository secret    | 内容                                                  |
|----------------------|-------------------------------------------------------|
| `DOCKERHUB_USERNAME` | 你的 Docker Hub 用户名（不是邮箱）                    |
| `DOCKERHUB_TOKEN`    | Docker Hub 的访问令牌，授予目标仓库 Read & Write 权限 |

在 Docker Hub 的个人命名空间下准备 `buct-course-remind-backend` 和 `buct-course-remind-frontend` 两个仓库；
若需要免登录拉取，将它们设为 Public。然后在 GitHub Actions 手动运行 **Build and Publish Docker Images**，
或通过原有的 `main` 分支推送 / `v*` 标签触发发布。发布后可使用：

```bash
docker pull docker.io/<你的DockerHub用户名>/buct-course-remind-backend:latest
docker pull docker.io/<你的DockerHub用户名>/buct-course-remind-frontend:latest
```

未同时配置上述两个 Secrets 时，只发布到 GHCR，日志会提示跳过 Docker Hub；PR 不登录、不推送到任何镜像仓库。
配置完整但凭据无效或无推送权限时，工作流会失败，请检查令牌和仓库权限。
使用 Docker Hub 部署时，将部署配置中的 GHCR 镜像地址替换为对应 Docker Hub 地址即可。

#### 1. 准备目录与 `.env`

```bash
mkdir -p ~/buct-deploy && cd ~/buct-deploy

# 生成 JWT 密钥
openssl rand -hex 32

# 创建 .env（把 <...> 换成真实值）
cat > .env <<'EOF'
# ── 必填：缺失则 compose 直接报错退出 ──
MONGO_INITDB_ROOT_USERNAME=<MongoDB 账号>
MONGO_INITDB_ROOT_PASSWORD=<MongoDB 口令>
SECRET_KEY=<上一步 openssl 的输出>
ECC_PRIVATE_KEY=<P-256 私钥十进制>
ECC_PUBLIC_KEY=<P-256 公钥 x 坐标十进制>

# ── 邮件：为空则验证码与提醒邮件发不出去 ──
MAIL_SMTP_SERVER=smtp.163.com
MAIL_SMTP_PORT=465
MAIL_SENDER=<你的邮箱>
MAIL_PASSWORD=<邮箱授权码，非登录密码>

# ── 端口 ──
GO_PORT=5000
FRONTEND_PORT=3033
EOF

chmod 600 .env
```

ECC 密钥的生成方式见下方「生成密钥」。**不要复用示例值**，
`SECRET_KEY` 泄漏等于任何人都能伪造登录态。

#### 2. 创建 `docker-compose.yml`

直接用仓库里现成的 [`web/docker-compose-ghcr.yml`](web/docker-compose-ghcr.yml)
（已实测可启动），复制过来重命名即可：

```bash
curl -fsSL -o docker-compose.yml \
  https://raw.githubusercontent.com/Ling0727-ai/BUCT-course-remind/main/web/docker-compose-ghcr.yml
```

或在已克隆的仓库里：

```bash
cp <仓库路径>/web/docker-compose-ghcr.yml ./docker-compose.yml
```

该文件不含任何密钥，全部通过同目录 `.env` 注入。服务名固定为 `backend`——
前端 nginx 的上游地址在构建时固化，改名会导致 502。

#### 3. 启动

镜像已公开，**无需登录**：

```bash
docker compose up -d
docker compose logs -f backend
```

> 若返回 `denied`：说明本机有失效的 GHCR 凭据（docker 优先用凭据而非匿名）。
> 执行 `docker logout ghcr.io` 清除后重试。
>
> 只有在需要推送镜像、或包被改回 private 时，才需要登录：
> ```bash
> export CR_PAT=<带 read:packages 的 classic PAT>
> echo $CR_PAT | docker login ghcr.io -u <你的GitHub用户名> --password-stdin
> ```

访问 `http://<服务器IP>:3033`。

#### 4. 验证

```bash
# 后端健康检查，应返回 status: healthy
curl -s http://localhost:5000/api/health

# 三个容器都应是 Up
docker compose ps
```

`mail_configured` 为 `false` 说明 `MAIL_PASSWORD` 没配好，验证码和提醒邮件发不出去。

#### 5. 更新与回滚

> `docker compose up -d` **不会**检查远端是否有新镜像：本地已存在 `:latest`
> 时它直接复用旧的。所以「更新」必须显式 `pull`，否则会一直跑旧版本。

```bash
# 更新到最新
docker compose pull && docker compose up -d

# 一条命令等价写法（每次都先拉再启动）
docker compose up -d --pull always

# 回滚到某个历史版本（sha-<短哈希> 或 1.2.3）
# ⚠️ 前后端都要改，只改一个会导致版本错配
sed -i -e 's|backend:latest|backend:sha-abc1234|' \
       -e 's|frontend:latest|frontend:sha-abc1234|' docker-compose.yml
docker compose pull && docker compose up -d
```

可用标签：`latest`（main 最新）、`sha-<短哈希>`、`1.2.3` / `1.2`（打 `v1.2.3` 标签时生成）。

生产环境建议**固定到 `sha-<短哈希>`** 而不是 `latest`，避免某次更新后行为
意外变化；需要升级时再显式改标签。

#### 常见问题

**前端返回 502**：compose 里的后端服务名不是 `backend`。nginx 的上游地址
在构建时固化，只能通过改名回 `backend` 解决。

**`required variable ... is missing a value`**：`.env` 缺必填项。
检查 `SECRET_KEY`、`ECC_PRIVATE_KEY`、`ECC_PUBLIC_KEY`、两个 Mongo 变量。

**`denied` 拉取失败**：镜像已公开，正常情况下无需登录。若仍 `denied`，
先清掉本机残留的失效凭据（docker 优先用凭据而非匿名访问）：

```bash
docker logout ghcr.io
```

若清除后依然 `denied`，再去包的设置页确认可见性确实为 public。

**数据卷会不会因为换目录/换 compose 而丢**：不会。Mongo 用的是**命名卷**
（`mongodb_data:/data/db`），卷名由 compose 里的 `name:` 字段决定，与项目
目录无关。本项目两种 compose 都固定为 `buct_mongodb_data`，因此可互换。
⚠️ 但 `docker compose down -v` 和 `docker volume rm` 会真删数据。

**改了代码但容器还是旧行为**：`docker compose up -d` 不会检查远端新镜像，
本地已有 `:latest` 就直接复用。用 `docker compose pull && docker compose up -d`。

**数据库数据在哪**：`mongodb_data` 卷。`docker compose down` 不会删数据，
`docker compose down -v` 会。

#### 从源码版迁移到镜像版（保留数据）

两种 compose 的卷名都固定为 `buct_mongodb_data`，因此数据可以无缝沿用。
迁移前**先备份**：

```bash
# 备份数据卷（推荐，与 mongodump 互为双保险）
docker run --rm -v buct_mongodb_data:/data -v "$(pwd)":/backup alpine \
  tar czf /backup/mongo-data-$(date +%F).tar.gz -C /data .

# 或导出为 mongodump 格式（可读性更好，便于单集合恢复）
docker exec buct-mongodb sh -c \
  'mongodump -u "$MONGO_INITDB_ROOT_USERNAME" -p "$MONGO_INITDB_ROOT_PASSWORD" \
   --authenticationDatabase admin --db=buct-course --out=/tmp/dump' \
&& docker cp buct-mongodb:/tmp/dump/buct-course ./buct-course-dump-$(date +%F)
```

然后迁移：

```bash
# 1. 停旧栈（不加 -v，卷会保留）
docker compose -f docker-compose-go.yml down

# 2. 换用镜像版 compose
mkdir -p ~/buct-deploy && cd ~/buct-deploy
cp <仓库>/web/docker-compose-ghcr.yml ./docker-compose.yml
cp <仓库>/web/.env.template ./.env

# 3. ⚠️ ECC 密钥必须与原来完全一致，否则库中已加密的 s_password 无法解密
vim .env

# 4. 启动（镜像已公开，无需登录）
docker compose up -d

# 5. 确认挂的是同一个卷
docker inspect buct-deploy-mongodb-1 --format '{{range .Mounts}}{{.Name}} -> {{.Destination}}{{"\n"}}{{end}}'
# 应输出：buct_mongodb_data -> /data/db
```

若第 5 步显示的是 `buct-deploy_mongodb_data`，说明 compose 里缺 `name:` 字段，
Mongo 挂到了空卷上（数据没丢，只是没挂上）。停掉后给 volumes 段补上
`name: buct_mongodb_data` 再启动即可：

```bash
docker compose down      # 不加 -v
# 补 name: 后
docker compose up -d
```

---

### 方式二：从源码构建部署

#### 使用 Go 后端

```bash
cd web

# 1. 从模板创建 .env，然后填入真实值（见下方「环境变量说明」）
cp .env.template .env

# 2. 启动（MongoDB + Go后端 + 前端）
docker compose -f docker-compose-go.yml up -d

# 3. 查看日志
docker compose -f docker-compose-go.yml logs -f backend
```

服务启动后访问：

- 前端：`http://localhost:3033`
- Go 后端：`http://localhost:5000`

#### 使用 Python 后端（已移除）

> Python (Flask) 后端因存在内存泄漏问题已从仓库移除，请使用 Go 后端。
> 如需查看历史实现：见旧仓库 `KMT-CN/BUCT-course-remind` 的 `web/backend/`。

#### ⚠️ 镜像不含任何密钥，必须自行配置

**本项目发布的 Docker 镜像里没有任何密钥或默认凭据**，只包含基础镜像自带的
环境变量与两个无关配置（后端 `PORT=:5000`、`ENV=production`）。
所有敏感配置都由部署者通过 `.env` 在运行时注入，不写入镜像层。

具体来说，以下变量**没有默认值**，缺失时 `docker compose` 会直接报错退出，
而不是静默使用一个公开的弱值：

| 变量 | 说明 | 缺失后果 |
|------|------|----------|
| `MONGO_INITDB_ROOT_USERNAME` | MongoDB 管理员账号 | 启动失败 |
| `MONGO_INITDB_ROOT_PASSWORD` | MongoDB 管理员口令 | 启动失败 |
| `SECRET_KEY` | JWT 签名密钥 | 启动失败 |
| `ECC_PRIVATE_KEY` | 学生学校密码的加密私钥 | 启动失败 |
| `ECC_PUBLIC_KEY` | 对应公钥 x 坐标 | 启动失败 |
| `MAIL_PASSWORD` | 邮箱授权码 | 邮件功能不可用 |

这样做是刻意的：如果给 `SECRET_KEY` 之类留一个 `change-this-in-production`
的默认值，未配置的部署就会用一个人人皆知的密钥签发登录态，任何人都能伪造它。


#### 环境变量说明（`.env`）

仓库只提供 `web/.env.template`，不含任何真实值。首次部署：

```bash
cd web
cp .env.template .env
# 然后编辑 .env 填入真实值
```

`.env` 已在 `.gitignore` 中忽略，**不会被提交**，也不会进入镜像。

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

# JWT 密钥（必填，无默认值）
# 用随机长字符串，例如：openssl rand -hex 32
SECRET_KEY=<随机长字符串>

# ECC 密钥（必填，用于加密存储学生的学校密码）
# 生成后请妥善保管；更换密钥必须同步迁移库中已加密的 s_password，否则无法解密
ECC_PRIVATE_KEY=<你的 P-256 私钥十进制值>
ECC_PUBLIC_KEY=<对应的公钥 x 坐标>

# 传输加密（RSA）
# ⚠️ RSA_ENABLE 必须为 true：后端强制要求加密传输，
#    设为 false 会导致登录/注册/重置密码等接口全部返回 400。
RSA_ENABLE=true

# CORS 白名单（可选，逗号分隔）
# 生产走 nginx 同源代理，通常留空；留空时仅放行同源请求。
ALLOWED_ORIGINS=

# 端口（可选，默认值如下）
GO_PORT=5000
FRONTEND_PORT=3033
```

#### 生成密钥

`SECRET_KEY` 用任意随机长字符串即可：

```bash
openssl rand -hex 32
```

ECC 密钥使用 NIST P-256 曲线，私钥与公钥均为十进制整数：

```bash
# 使用 Python 的 cryptography 库生成
python -c "
from cryptography.hazmat.primitives.asymmetric import ec
k = ec.generate_private_key(ec.SECP256R1())
n = k.private_numbers()
print('ECC_PRIVATE_KEY=' + str(n.private_value))
print('ECC_PUBLIC_KEY=' + str(n.public_numbers.x))
"
```

> ⚠️ 更换 ECC 密钥前必须先用旧密钥解密、再用新密钥重新加密库中所有
> `s_password`，否则用户的自动抓取会全部失败。

---

### 方式三：本地开发运行

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
| GET    | `/assignments/standard?summary=true` | 获取不含正文的作业摘要 |
| GET    | `/assignments/:id/detail`            | 按需获取作业正文 |
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
| POST   | `/todos/:id/complete`         | 标记完成（保留到删除或撤销完成） |
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

Go 后端按数据库中的最近到期时间设置单次 `time.Timer`，写入后通过进程内通知重新调度：

| 任务     | 间隔    | 说明                              |
|--------|-------|---------------------------------|
| 课程数据刷新 | 每用户成功刷新后 3 小时 | 索引领取到期账号，统一队列执行，同一用户的并发刷新合并 |
| 到期提醒发送 | 按 `scheduled_time` | 分批领取提醒，最多 2 个邮件发送任务同时执行 |
| 自动提醒生成 | 按 `auto_reminder_at` | 首档发送前预留 30 秒安排 24h、12h、1h 三档任务，每批最多 200 条，批量查重和写入 |

刷新队列默认 2 个 worker、128 个等待名额，可通过 `REFRESH_WORKERS`（上限 8）和 `REFRESH_QUEUE_SIZE`（上限 1024）调整。队列满时同步刷新接口返回 HTTP 503 和 `Retry-After`；取消请求只停止等待，共享的刷新任务会继续完成。定时器按秒唤醒，实际开始和完成时间仍取决于排队及上游响应速度。刷新调度最多每 5 分钟、提醒调度最多每 1 分钟校准一次，以发现其他实例的写入。

三档自动提醒的计划发送时间分别为 `deadline_at - 24h - 5s`、`deadline_at - 12h - 5s`、`deadline_at - 1h - 5s`。提前 5 秒用于补偿部分发送延迟，不保证实际到达时间。首次发现任务较晚或停机恢复时，仅补发最近一个已到期且仍有效的节点，保留之后的计划任务；24h 节点在 12h 节点的提前发送时间失效，12h 节点在 1h 节点的提前发送时间失效，1h 节点在作业截止时失效。已有的待发送自动提醒会分批迁移，避免重复扣减提前量。

课程同步只写入新增或变化的任务，成功写入后才删除上游已移除的任务。没有变化时仅更新用户的 `course_refreshed_at` 和 `next_refresh_at`；空结果也会记录刷新时间。抓取失败不会作为空结果保存。

列表和统计共享按用户缓存的摘要快照，最多 128 个用户、估算 8 MiB、有效期 10 秒；状态修改、黑名单和课程写入后立即失效。首页直接使用列表中的完成状态，正文在打开预览时单独加载。多实例之间依靠短有效期收敛缓存。

首次启动会分批补齐旧作业的 BSON 日期字段，并在 `_migrations` 记录完成标记；迁移失败会在后续调度中重试。用户名和邮箱会尝试创建唯一索引；若旧数据存在重复值，保留数据并回退到普通查询索引，启动日志会提示处理重复值。

### 后端技术指标

| 指标          | Go (Gin)               |
|-------------|------------------------|
| Docker 镜像大小 | ~20MB                  |
| 容器启动时间      | <1s                    |
| 并发模型        | goroutine              |
| 定时任务        | time.Timer + 有界任务队列 |
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
- [x] Go 后端关键逻辑回归测试 — 已覆盖刷新、任务 ID、提醒和完成状态

## 许可证

MIT License
