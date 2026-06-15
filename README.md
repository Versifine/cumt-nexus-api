# CUMT Nexus API

`cumt-nexus-api` 是一个社区内容平台后端，使用 Go、Gin、PostgreSQL 和 pgx 构建。项目采用模块化单体结构，按业务模块组织代码，重点覆盖用户身份、社区、帖子、评论、投票、搜索、通知、媒体上传和基础审核能力。

## 功能概览

- 用户注册、登录、Bearer JWT 认证、当前用户读取、公开资料更新和用户关注
- 社区列表、社区详情、社区创建申请、申请审核读取、平台 staff 审批、社区基础设置和社区规则管理
- 平台 staff 用户 / 社区 / 评论效果 / 积分运营 / 头衔管理、运行开关和审计日志
- 帖子发布、列表、详情、编辑、软删除
- 评论发布、列表、树状读取、编辑、软删除
- 帖子 upvote/downvote/cancel、保存、评论投票、社区关注和 `best` / `hot` / `new` / `top` / `rising` 排序
- 图片上传、帖子/评论图片附件绑定、结构化 `content_refs` 引用持久化，支持本地存储和 Cloudflare R2
- 内容举报、平台 staff 移除内容、举报列表和举报处理
- PostgreSQL 搜索，支持匿名读取公开社区、帖子和用户结果
- 站内通知读取、分类筛选、未读摘要和标记已读
- 统一错误响应、request id、请求日志、panic recovery、CORS 配置

## 技术栈

| 能力 | 选择 |
|---|---|
| Language | Go |
| HTTP | Gin |
| Database | PostgreSQL |
| Database driver | pgx |
| Auth | JWT access token |
| Migration | golang-migrate |
| Object storage | local filesystem / Cloudflare R2 |
| Logging | slog |

## 快速开始

### 1. 准备配置

```powershell
Copy-Item .env.example .env
```

macOS/Linux:

```bash
cp .env.example .env
```

本地默认使用 PostgreSQL 和 local object storage。生产或接近生产的图片上传应使用 Cloudflare R2；真实密钥只放在本地 `.env` 或部署平台 secret 中，不写入 README、提交信息或文档。

### 2. 启动 PostgreSQL

```bash
docker compose up -d postgres
```

### 3. 执行 migration

```bash
go run ./cmd/migrate up
```

查看 migration 版本：

```bash
go run ./cmd/migrate version
```

### 4. 启动 API

```bash
go run ./cmd/api
```

默认监听 `.env` 中的 `HTTP_ADDR`，示例配置为 `:8080`。

健康检查：

```bash
curl http://localhost:8080/healthz
```

预期响应：

```json
{"status":"ok"}
```

## 配置

项目启动时会自动加载当前目录下的 `.env`。完整配置样例见 [.env.example](.env.example)。

常用配置分组：

| 分组 | 变量前缀 | 说明 |
|---|---|---|
| App | `APP_` | 应用名、运行环境、启动超时 |
| PostgreSQL | `POSTGRES_` | 数据库连接和连接池参数 |
| HTTP | `HTTP_` | 监听地址、超时、CORS |
| Log | `LOG_` | 日志级别和格式 |
| Auth | `AUTH_` | JWT secret、access token TTL、邮箱验证码和登录频控 |
| Mail | `MAIL_` / `SMTP_` | 验证码邮件发送；本地默认 `MAIL_PROVIDER=log` |
| Storage | `OBJECT_STORAGE_` | local/R2 对象存储配置 |
| Upload | `UPLOAD_` | 图片大小和数量限制 |

### Cloudflare R2

R2 使用 S3-compatible 配置。endpoint 使用账号级地址，bucket 单独配置：

```env
OBJECT_STORAGE_PROVIDER=r2
OBJECT_STORAGE_ENDPOINT=https://<account-id>.r2.cloudflarestorage.com
OBJECT_STORAGE_REGION=auto
OBJECT_STORAGE_BUCKET=<bucket-name>
OBJECT_STORAGE_ACCESS_KEY_ID=<access-key-id>
OBJECT_STORAGE_SECRET_ACCESS_KEY=<secret-access-key>
OBJECT_STORAGE_PUBLIC_BASE_URL=<public-media-base-url>
OBJECT_STORAGE_FORCE_PATH_STYLE=true
```

不要把 bucket 名拼进 `OBJECT_STORAGE_ENDPOINT`，也不要提交真实 access key 或 secret key。`OBJECT_STORAGE_PUBLIC_BASE_URL` 必须是浏览器可公开读取图片的 base URL，例如 R2 public development URL 或自定义域名；不要填写 `https://<account-id>.r2.cloudflarestorage.com` 这个 S3 API endpoint。

## API 概览

写操作、权限操作和用户态读取需要：

```http
Authorization: Bearer <access_token>
```

公开帖子流、公开帖子详情、公开评论、公开社区、公开用户主页和搜索支持匿名读取；如果请求携带 Bearer token，后端会返回当前用户视角的 `my_vote`、`is_saved`、`viewer_is_following` 和 `viewer_permissions`，无 token 时这些 viewer 字段为匿名态。格式错误、过期或签名错误的 token 仍返回 `unauthenticated`，不会静默降级为匿名。

所有 `limit/offset` 列表响应统一包含 `limit`、`offset`、`next_offset` 和 `has_more`。响应数组最多返回 `limit` 条，前端应以 `has_more` 判断是否继续请求，并用 `next_offset` 作为下一页 offset。`GET /api/v1/communities` 也支持 `limit/offset`，默认 20、最大 50。

### Auth

```text
POST /api/v1/auth/register
POST /api/v1/auth/email-codes/register
POST /api/v1/auth/register-with-email
POST /api/v1/auth/login
POST /api/v1/auth/email-codes/login
POST /api/v1/auth/login-with-email-code
POST /api/v1/auth/email-codes/password-reset
POST /api/v1/auth/password-reset
GET  /api/v1/me
PATCH /api/v1/me/profile
GET  /api/v1/me/security
POST /api/v1/me/security/email-codes/change-email
POST /api/v1/me/security/email-codes/delete-account
PATCH /api/v1/me/security/email
PATCH /api/v1/me/security/password
DELETE /api/v1/me/account
POST /api/v1/auth/logout-all
GET  /api/v1/me/saved-posts
GET  /api/v1/me/followed-communities
GET  /api/v1/me/followed-users?limit=20&offset=0
GET  /api/v1/me/progression
GET  /api/v1/me/xp-events?limit=20&offset=0
GET  /api/v1/me/titles?limit=20&offset=0
PATCH /api/v1/me/title
GET  /api/v1/users/:username            # public, optional Bearer
POST /api/v1/users/:username/follow
DELETE /api/v1/users/:username/follow
GET  /api/v1/users/:username/posts      # public, optional Bearer
GET  /api/v1/users/:username/comments   # public, optional Bearer
```

邮箱验证码只发送到 `AUTH_EMAIL_ALLOWED_DOMAINS` 配置的域名。本地 `MAIL_PROVIDER=log` 时验证码写入 API 日志，不会返回到 HTTP 响应；生产可切到 `MAIL_PROVIDER=smtp` 并配置 `SMTP_*`。`POST /api/v1/auth/login` 兼容旧 `{ "username": "...", "password": "..." }`，也支持 `{ "identifier": "...", "password": "..." }`，其中 `identifier` 可为用户名或邮箱。

用户关注使用 `POST /api/v1/users/:username/follow` 和 `DELETE /api/v1/users/:username/follow`，成功均返回 204 且幂等；不能关注自己。公开用户主页返回 `stats.follower_count`、`stats.following_count` 和 `viewer_is_following`，匿名读取时 `viewer_is_following=false`。

### Communities

```text
GET  /api/v1/communities?limit=20&offset=0 # public, optional Bearer
GET  /api/v1/communities/:slug          # public, optional Bearer
GET  /api/v1/communities/:slug/manage
GET  /api/v1/communities/:slug/manage/posts
GET  /api/v1/communities/:slug/manage/comments
GET  /api/v1/communities/:slug/manage/reports
GET  /api/v1/communities/:slug/manage/members
POST /api/v1/communities/:slug/manage/moderators
DELETE /api/v1/communities/:slug/manage/moderators/:user_id
GET  /api/v1/me/community-owner-transfers?status=pending&limit=20&offset=0
GET  /api/v1/communities/:slug/manage/owner-transfer
POST /api/v1/communities/:slug/manage/owner-transfer
GET  /api/v1/communities/:slug/owner-transfer/:transfer_id
POST /api/v1/communities/:slug/manage/owner-transfer/:transfer_id/accept
DELETE /api/v1/communities/:slug/manage/owner-transfer/:transfer_id
GET  /api/v1/communities/:slug/manage/settings
PATCH /api/v1/communities/:slug/manage/settings
GET  /api/v1/communities/:slug/manage/rules
POST /api/v1/communities/:slug/manage/rules
PATCH /api/v1/communities/:slug/manage/rules/:rule_id
DELETE /api/v1/communities/:slug/manage/rules/:rule_id
POST /api/v1/communities/:slug/follow
DELETE /api/v1/communities/:slug/follow
POST /api/v1/community-applications
GET  /api/v1/community-applications
GET  /api/v1/community-applications/:id
POST /api/v1/community-applications/:id/approve
POST /api/v1/community-applications/:id/reject
```

社区管理设置读取允许 owner/moderator 进入管理上下文；设置更新允许真实 owner 或平台 owner 覆盖修改 `name`、`description`、`avatar_url` 和 `banner_url`，媒体 URL 为空字符串表示清除，非空必须是 `http/https` 绝对 URL。社区规则列表和 CRUD 允许 owner/moderator 使用，按 `position`、创建时间和 ID 稳定排序。

Community governance write routes are Bearer-only. Appointing/removing moderators is allowed for the real community owner and for active platform `owner` override; creating a community owner transfer still requires the real community owner. Moderator caps are based on active member count: under 500 allows 5, 500 or more allows 10, and 2000 or more allows 20. Owner transfer is a two-step flow: current owner creates a pending transfer by target username, the management route can read or cancel it, the accept-page route returns status and both user summaries, and the target accepts it within 48 hours. Platform `owner` can manage all active communities without being written into `community_memberships`; responses keep `viewer_role` as the real community role and set `viewer_permissions.platform_owner_override=true`. Platform staff can take over a community owner through the admin owner route and include an audit reason.

### Admin

```text
GET  /api/v1/admin/users
PATCH /api/v1/admin/users/:id
PATCH /api/v1/admin/users/:id/platform-role
GET  /api/v1/admin/owner-transfer
POST /api/v1/admin/owner-transfer
DELETE /api/v1/admin/owner-transfer/:transfer_id
GET  /api/v1/admin/communities
PATCH /api/v1/admin/communities/:id
POST /api/v1/admin/communities/:id/owner
GET  /api/v1/admin/effects
PATCH /api/v1/admin/effects/:id
GET  /api/v1/admin/settings
PATCH /api/v1/admin/settings/:key
GET  /api/v1/admin/audit-logs
GET  /api/v1/admin/point-transactions
POST /api/v1/admin/users/:id/points/adjust
POST /api/v1/admin/users/:id/sanctions
GET  /api/v1/admin/users/:id/sanctions
POST /api/v1/admin/user-sanctions/:sanction_id/revoke
GET  /api/v1/admin/titles
POST /api/v1/admin/titles
PATCH /api/v1/admin/titles/:id
GET  /api/v1/admin/users/:id/titles
POST /api/v1/admin/users/:id/titles
DELETE /api/v1/admin/users/:id/titles/:grant_id
GET  /api/v1/owner-transfer/:transfer_id
POST /api/v1/owner-transfer/:transfer_id/accept
```

平台管理接口都需要 Bearer，并要求当前用户是 active 平台 staff。写操作会记录平台管理审计日志。运行开关包括 `registration_enabled`、`posting_enabled` 和 `upload_enabled`，分别控制注册、发帖 / 发评论和图片上传。手工调整用户积分使用 JSON `{"delta":15,"reason":"manual bonus"}`，`delta` 不能为 0，扣减后余额不能小于 0，成功后写入积分流水和审计日志。用户处罚接口当前支持 `type=account_ban` 和 `duration=1d|3d|7d|30d|permanent`；active 未过期封禁会阻止登录和受保护接口，解除封禁保留处罚记录并写审计。头衔目录和授予也由平台 staff 管理；头衔名称最多 20 字符，且不能使用官方、管理员、认证、平台、系统、版主、admin、official、verified 等保留词。

Platform role writes use `role=admin|staff|null` for non-owner users only. Current owner can start a platform owner transfer, the target accepts with their own password within 48 hours, and deployment-side `cmd/admin` commands handle first-owner bootstrap or compromised-owner recovery.

### Posts

```text
GET    /api/v1/communities/:slug/posts  # public, optional Bearer
GET    /api/v1/posts                    # public, optional Bearer
GET    /api/v1/posts/:id                # public, optional Bearer
GET    /api/v1/users/:username/posts    # public, optional Bearer
POST   /api/v1/communities/:slug/posts
PATCH  /api/v1/posts/:id
DELETE /api/v1/posts/:id
POST   /api/v1/posts/:id/save
DELETE /api/v1/posts/:id/save
PUT    /api/v1/posts/:id/vote
DELETE /api/v1/posts/:id/vote
```

全站 feed 支持：

```text
GET /api/v1/posts?source=all&sort=new&limit=20&offset=0
GET /api/v1/posts?source=recommended&sort=hot&t=day&limit=20&offset=0
GET /api/v1/posts?source=following&sort=new&limit=20&offset=0
```

`source=recommended` 当前是后端公开可解释推荐流，默认 `sort=hot`，匿名读取使用 `hot + new` 混排并做社区 rank 去重；携带有效 Bearer 时会给关注社区和互动过的社区加权。显式传 `sort=best|hot|new|top|rising` 时，该排序语义作为推荐基线；它不是机器学习推荐，也不是预计算时间线。`t` 支持 `hour|day|week|month|year|all`。
`source=following` 需要 Bearer，只返回当前用户已关注公开社区内的 visible 帖子；未登录访问返回 `unauthenticated`。

发帖和编辑帖子支持可选 `content_refs` 请求字段，元素结构为 `{ "kind": "image|link_preview|embed", "ref_id": "..." }`。`image` 引用必须指向同一次请求绑定或帖子当前已绑定的图片附件 ID；`embed` 引用必须使用 `POST /api/v1/embeds/resolve` 返回的 `embed.id`；编辑时省略 `content_refs` 保留原引用，传空数组清空原引用。

### Comments

```text
POST   /api/v1/posts/:id/comments
GET    /api/v1/posts/:id/comments       # public, optional Bearer
GET    /api/v1/users/:username/comments # public, optional Bearer
PATCH  /api/v1/comments/:id
DELETE /api/v1/comments/:id
PUT    /api/v1/comments/:id/vote
DELETE /api/v1/comments/:id/vote
```

评论树读取通过查询参数启用：

```text
GET /api/v1/posts/:id/comments?view=tree&sort=new&limit=20&offset=0&max_depth=6
```

发评论和编辑评论同样支持可选 `content_refs`，语义与帖子一致：按请求顺序返回，`image` 引用必须匹配评论已绑定图片附件，`embed` 引用必须使用已解析的 `embed.id`，省略保留、空数组清空。

### Media

```text
POST /api/v1/uploads/images
POST /api/v1/link-previews/resolve
POST /api/v1/embeds/resolve
```

上传使用 `multipart/form-data`，字段为 `file` 和可选 `alt_text`；PNG/JPEG 上传成功后会返回解析出的图片 `width` 和 `height`。大于 512px 边长的 PNG/JPEG 会同步生成最大边 512px 的独立 JPEG 缩略图，附件响应中的 `thumbnail_url` 指向该缩略图；小图、WebP 或缩略图生成 / 上传失败时，`thumbnail_url` 回退为原图 `url`。

`POST /api/v1/embeds/resolve` 支持 Bilibili、抖音、网易云音乐和 QQ 音乐白名单链接。请求体可以是 URL，也可以是包含 URL 的分享文本；后端会展开 `b23.tv` / `v.douyin.com` 短链、提取稳定资源 ID、返回受控 `embed_url`、`canonical_url`、`provider_ref`、元数据和 `status`，并持久化 `embed.id` 供帖子 / 评论 `content_refs.kind=embed` 引用。后端不接受用户提交的任意 iframe HTML。

未绑定和异常附件通过后台清理命令回收：

```bash
go run ./cmd/media-cleanup -dry-run
go run ./cmd/media-cleanup -unbound-ttl=24h -failed-ttl=24h -limit=100
```

清理候选只包含 `owner_type=none` 的附件：`ready` 状态按未绑定 TTL 回收，`failed` / `blocked` 状态按异常 TTL 回收；TTL 从当前状态的 `updated_at` 计算。实际清理会先领取并删除附件元数据，再删除原图和缩略图对象。生产环境应由 cron、systemd timer 或 Windows Task Scheduler 周期性运行该命令。

链接预览和嵌入解析使用 JSON `{"url":"https://..."}`，会校验公开 HTTP(S) URL、拦截本机/私网地址，并只对首批白名单 provider 返回嵌入结果。

完整接口合同见 [docs/contracts/http-api-contract.md](docs/contracts/http-api-contract.md)，请求/响应字段合同见 [docs/contracts/http-api-schema.md](docs/contracts/http-api-schema.md)。

### Effects / Points

```text
GET  /api/v1/effects/catalog       # public, optional Bearer
GET  /api/v1/me/points
GET  /api/v1/me/point-transactions?limit=20&offset=0
POST /api/v1/comments/:id/effects
GET  /api/v1/admin/point-transactions?user_id=<uuid>&limit=20&offset=0
POST /api/v1/admin/users/:id/points/adjust
```

评论效果应用使用 JSON `{"effect_id":"sparkle"}`。后端负责积分账户初始化、余额扣减、评论效果记录和积分流水审计。评论列表和评论树响应会返回 `effects[]` 历史效果摘要；效果停用后历史记录仍可展示，但不会再出现在可购买目录中。当前用户和平台 staff 都可以按分页读取积分流水，平台 staff 可按 `user_id` 过滤。

### Progression / Titles

```text
GET    /api/v1/me/progression
GET    /api/v1/me/xp-events?limit=20&offset=0
GET    /api/v1/me/titles?limit=20&offset=0
PATCH  /api/v1/me/title
GET    /api/v1/admin/titles?scope_type=all&active=all&limit=20&offset=0
POST   /api/v1/admin/titles
PATCH  /api/v1/admin/titles/:id
GET    /api/v1/admin/users/:id/titles?limit=20&offset=0
POST   /api/v1/admin/users/:id/titles
DELETE /api/v1/admin/users/:id/titles/:grant_id
```

全站等级只按 `xp_total` 计算，不做社区等级。自动经验来源包括发帖、发评论、收到帖子 / 评论 upvote 和帖子被收藏；经验事件按 `user_id + source_type + source_id` 去重，并受每日上限限制。公开用户响应包含 `progression` 对象，用户可以从自己已获得且未失效的头衔中选择 `active_title`。

### Moderation

```text
POST /api/v1/posts/:id/reports
POST /api/v1/comments/:id/reports
POST /api/v1/posts/:id/moderation/remove
POST /api/v1/comments/:id/moderation/remove
POST /api/v1/communities/:slug/moderation/posts/:id/remove
POST /api/v1/communities/:slug/moderation/comments/:id/remove
GET  /api/v1/moderation/reports
GET  /api/v1/moderation/reports/:id
POST /api/v1/moderation/reports/:id/dismiss
POST /api/v1/moderation/reports/:id/remove-target
GET  /api/v1/admin/mod-queues?queue=reports&limit=20&offset=0
POST /api/v1/admin/mod-queues/actions
GET  /api/v1/communities/:slug/mod-queues?queue=reports&limit=20&offset=0
POST /api/v1/communities/:slug/mod-queues/actions
POST /api/v1/communities/:slug/moderation/posts/:id/approve
POST /api/v1/communities/:slug/moderation/comments/:id/approve
POST /api/v1/communities/:slug/moderation/posts/:id/spam
POST /api/v1/communities/:slug/moderation/comments/:id/spam
POST /api/v1/communities/:slug/moderation/reports/:id/ignore
POST /api/v1/communities/:slug/moderation/posts/:id/lock
POST /api/v1/communities/:slug/moderation/posts/:id/pin
POST /api/v1/communities/:slug/moderation/posts/:id/mark-nsfw
POST /api/v1/communities/:slug/moderation/posts/:id/mark-spoiler
POST /api/v1/communities/:slug/moderation/posts/:id/flair
GET/POST/PATCH/DELETE /api/v1/communities/:slug/moderation/removal-reasons
POST /api/v1/communities/:slug/moderation/removal-reasons/:id/apply
GET/POST/PATCH/DELETE /api/v1/communities/:slug/moderation/saved-responses
GET/POST/DELETE /api/v1/communities/:slug/manage/banned-users
GET/POST/DELETE /api/v1/communities/:slug/manage/muted-users
GET/POST/DELETE /api/v1/communities/:slug/manage/approved-users
GET  /api/v1/communities/:slug/moderation/users/:user_id/profile
GET  /api/v1/communities/:slug/moderation/users/:user_id/notes
POST /api/v1/communities/:slug/moderation/users/:user_id/notes
DELETE /api/v1/communities/:slug/moderation/users/:user_id/notes/:note_id
GET  /api/v1/communities/:slug/moderation/logs?action=&actor_id=&target_type=&target_id=
```

平台 staff 接口适用于全站审核台。社区范围工具要求当前用户是路径社区的 owner 或 moderator，且目标帖子 / 评论必须属于该社区；P1 覆盖审核队列、批量内容动作、移除原因、保存回复、社区用户治理、mod notes 和社区 Mod Log。Modmail、Automod、flair 模板、scheduled posts、guides / digest / insights 仍属于后续增强。

### Search / Notifications

```text
GET  /api/v1/search                         # public, optional Bearer
GET  /api/v1/notifications/unread-summary
GET  /api/v1/notifications?category=interactions&limit=20&offset=0
POST /api/v1/notifications/:id/read
POST /api/v1/notifications/read-all
```

搜索基于 PostgreSQL full-text search、字段权重、精确/前缀/子串命中和轻量时间衰减排序；`scope=all|communities|posts|users`，其中 `all` 分区返回公开社区、visible 帖子和 active 用户公开资料摘要。首版不返回高亮片段或命中原因，前端可基于响应字段自行高亮。

评论、回复、帖子点赞、评论点赞和正文 `@username` 提及会写入站内通知；`category=interactions` 返回回复、提及和点赞类用户互动通知，`category=system` 返回系统通知，不传 `status` 默认返回 `all`。点赞通知按收件人、通知类型、目标内容和小时窗口聚合未读计数，响应包含 `aggregate_count`、`actor`、`last_actor` 和帖子/评论 `context`，评论通知可用 `context.permalink` 直达锚点。社区负责人交接会给目标账号写系统通知，`source_type=community_owner_transfer`，`source_id=<community_slug>:<transfer_id>`。提及通知在帖子 / 评论发布时生成，编辑时只为新增提及生成，并进入互动分类。

## 响应约定

错误响应统一为：

```json
{
  "error": {
    "code": "invalid_argument",
    "message": "..."
  }
}
```

当前错误码包括：

```text
invalid_argument
unauthenticated
forbidden
not_found
conflict
internal
```

`DELETE` 成功时返回 `204 No Content`。

错误响应、配置和 migration 的可校验合同文档集中放在 [docs/contracts/](docs/contracts/)。

## 本地开发

常用检查：

```bash
go test ./...
go build -buildvcs=false ./...
```

快速基线检查：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-current-baseline.ps1 -SkipHttpSmoke -R2Mode Skip
```

更细的本地 smoke 和契约校验脚本放在 `scripts/` 目录；README 只保留常用入口。

## 目录结构

```text
cmd/
  api/        HTTP API 启动入口
  migrate/    migration CLI
internal/
  admin/      平台管理、运行开关、审计日志
  auth/       认证、密码、token、HTTP auth
  user/       用户模型和当前用户
  community/  社区、申请、审批
  post/       帖子写入和读取
  comment/    评论写入和读取
  vote/       帖子投票
  moderation/ 举报和审核
  media/      图片附件
  storage/    local/R2 对象存储
  search/     搜索
  notification/ 通知
  platform/   配置、数据库、HTTP server、日志
migrations/   PostgreSQL migrations
scripts/      本地校验和 smoke 脚本
```

## License

[MIT License](./LICENSE)
