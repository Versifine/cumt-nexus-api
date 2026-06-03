# CUMT Nexus API

`cumt-nexus-api` 是一个社区内容平台后端。当前采用 Go + Gin + PostgreSQL，架构形态是模块化单体。

## 当前状态

阶段：`阶段 25 API 认证边界契约校验已完成`

代码已完成阶段 1 认证与用户基础闭环、阶段 2 社区申请与审批闭环、阶段 3 帖子发布和读取闭环、阶段 4 评论发布和读取闭环、阶段 5 完整真实冒烟、阶段 6 全站最新帖子流 + 帖子 upvote/downvote 基础、阶段 7 轻量举报与平台 staff 移除内容闭环、阶段 8 审核台最小闭环、阶段 9 hot feed / 内容分发闭环、阶段 10 审核台增强闭环、阶段 11 搜索闭环、阶段 12 通知闭环、阶段 13 内容系统增强闭环、阶段 14 内容编辑与删除闭环、阶段 15 R2 真实凭据 smoke 工具、阶段 16 工程验收入口、阶段 17 HTTP API 契约快照、阶段 18 配置契约清单校验、阶段 19 migration 契约与清单校验、阶段 20 配置语义契约校验、阶段 21 配置加载运行时契约测试、阶段 22 HTTP API request/response schema 契约快照和字段清单校验、阶段 23 HTTP 错误码、HTTP 状态码和错误响应形状契约校验、阶段 24 API schema 路由映射校验，以及阶段 25 API 认证边界契约校验。

阶段 13 已完成：已升级内容系统，支持 Reddit-style 评论树、Markdown-like 帖子/评论正文契约、图片附件和 Cloudflare R2 图片上传。阶段 13 不做前端 UI、富文本 HTML 编辑器、任意 HTML、任意 iframe、Bilibili/网易云播放器、评论投票、通知扩展、搜索扩展、生产真实密钥配置或对象物理删除任务。

阶段 14 已完成：作者可以编辑和软删除自己的帖子与评论。用户主动删除使用 `deleted`，平台审核移除继续使用 `removed`；删除不物理删除图片附件或 R2 对象。

阶段 15 已完成：新增真实 Cloudflare R2 dev bucket 上传 smoke 脚本，并固化凭据门禁和文档边界。当前本机没有 R2 凭据时只能验证 skipped 分支，不能把真实 R2 上传标记为通过。

阶段 16 已完成：新增当前基线验收脚本，聚合测试、构建、migration、Stage 13/14 HTTP smoke 和 Stage 15 R2 凭据门禁。本阶段不新增业务接口，不进入新的 feed、vote、moderation、notification 或 search 产品语义。

阶段 17 已完成：新增当前 HTTP API 契约快照和路由清单校验脚本。本阶段不新增业务接口，不改变错误码或响应格式。

阶段 18 已完成：新增配置契约清单校验脚本，确保配置加载代码、`.env.example` 和配置文档中的环境变量名保持同步。本阶段不改变运行时配置加载语义。

阶段 19 已完成：新增 migration 契约清单校验脚本，确保 `migrations/` 文件编号连续、up/down 成对、命名一致，并与内部 migration 文档清单同步。本阶段不新增或修改 schema migration。

阶段 20 已完成：新增配置语义契约校验脚本，确保配置文档中的必需性、默认值和枚举说明与 `internal/platform/config/load.go`、`internal/platform/config/validate.go` 同步。本阶段不新增配置变量，不改变运行时配置语义。

阶段 21 已完成：新增配置加载运行时契约测试，覆盖 `Load()` 的 local 默认值、R2 配置加载、R2 缺凭据失败和基础解析失败路径。本阶段不改变运行时配置语义。

阶段 22 已完成：新增当前 HTTP API schema 契约快照和 handler JSON 字段清单校验，并纳入当前基线脚本。本阶段不新增业务接口，不生成 OpenAPI，不改变错误码或响应格式。

阶段 23 已完成：新增 HTTP 错误契约校验脚本，确保 `internal/apperr`、`internal/platform/httpserver` 和 `docs/internal/architecture/http-error-handling.md` 的错误码、HTTP 状态码和错误响应形状保持同步。本阶段不新增错误码，不改变错误响应格式。

阶段 24 已完成：补强 API schema 契约校验脚本，确保 `docs/internal/architecture/http-api-schema.md` 的接口 schema 映射覆盖 `docs/internal/architecture/http-api-contract.md` 的当前路由、没有过期路由、schema 引用真实存在，并约束成功状态码。本阶段不新增业务接口，不生成 OpenAPI，不改变响应格式。

阶段 25 已完成：补强 API 契约校验脚本，确保 `docs/internal/architecture/http-api-contract.md` 的 Auth 列与 `/healthz` public route、auth public group、local-only static route 和 `authhttp.RequireAuth` 保护分组保持同步。本阶段不新增业务接口，不改变认证中间件语义或响应格式。

2026-06-03 合同复核：用当前源码重新启动本地 API 后，前端 `npm run check:main-path` 严格模式已无评论树 warning；此前 warning 来自旧后端进程。当前 `view=tree` 合同仍是扁平前序遍历数组，父评论先于子评论。`PATCH/DELETE /api/v1/posts/:id` 和 `PATCH/DELETE /api/v1/comments/:id` 的前端实现合同已在下方接口说明和 `docs/internal/architecture/content-lifecycle.md` 收口。

已具备：

- 配置加载与校验
- PostgreSQL 连接池初始化
- migration 执行入口
- HTTP 服务启动、优雅关闭和 `/healthz`
- 统一错误响应、recovery、request id、请求日志
- 用户领域模型、密码哈希、用户仓储
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- JWT access token 签发
- Bearer JWT 认证中间件
- `GET /api/v1/me`
- 阶段 2 社区 schema migration
- 社区领域模型与 PostgreSQL repository
- API 启动期公共总版 bootstrap
- `GET /api/v1/communities`
- `GET /api/v1/communities/:slug`
- `POST /api/v1/community-applications`
- `POST /api/v1/community-applications/:id/approve`
- `POST /api/v1/community-applications/:id/reject`
- 审批通过后事务内创建社区和申请人 `owner` 成员关系
- 阶段 3 帖子 schema、domain、repository
- `POST /api/v1/communities/:slug/posts`
- `PATCH /api/v1/posts/:id`
- `DELETE /api/v1/posts/:id`
- `GET /api/v1/communities/:slug/posts`
- `GET /api/v1/posts/:id`
- 阶段 4 评论 schema、domain、repository
- `POST /api/v1/posts/:id/comments`
- `PATCH /api/v1/comments/:id`
- `DELETE /api/v1/comments/:id`
- `GET /api/v1/posts/:id/comments`
- 阶段 6 帖子投票 schema、domain、repository
- `PUT /api/v1/posts/:id/vote`
- `DELETE /api/v1/posts/:id/vote`
- `GET /api/v1/posts`
- 阶段 7 举报和审核 schema、domain、repository
- `POST /api/v1/posts/:id/reports`
- `POST /api/v1/comments/:id/reports`
- `POST /api/v1/posts/:id/moderation/remove`
- `POST /api/v1/comments/:id/moderation/remove`
- `GET /api/v1/moderation/reports`
- `GET /api/v1/moderation/reports/:id`
- `POST /api/v1/moderation/reports/:id/dismiss`
- `POST /api/v1/moderation/reports/:id/remove-target`
- 审核台举报列表和详情响应 `target_preview`
- `GET /api/v1/posts?sort=new|hot`
- `GET /api/v1/communities/:slug/posts?sort=new|hot`
- `GET /api/v1/search?q=...&scope=all|communities|posts`
- `GET /api/v1/notifications`
- `POST /api/v1/notifications/:id/read`
- 阶段 13 文档边界：评论树、Markdown-like 正文、图片附件和 Cloudflare R2 存储
- `POST /api/v1/uploads/images`
- 评论树读取契约和 `body_format=markdown`
- 发帖和发评论 `attachment_ids` 图片绑定
- 当前基线验收入口 `scripts/verify-current-baseline.ps1`
- HTTP API 路由与认证边界契约快照 `docs/internal/architecture/http-api-contract.md`
- HTTP API schema 契约快照 `docs/internal/architecture/http-api-schema.md`
- HTTP API schema 字段清单与路由映射校验 `scripts/verify-api-schema-doc.ps1`
- HTTP 错误契约校验 `scripts/verify-http-error-contract-doc.ps1`
- 配置契约清单校验 `scripts/verify-config-contract-doc.ps1`
- 配置语义契约校验 `scripts/verify-config-semantics-doc.ps1`
- 配置加载运行时契约测试 `internal/platform/config/load_test.go`
- migration 契约清单校验 `scripts/verify-migration-contract.ps1`
- 移除内容和审核动作写入同一 PostgreSQL 事务
- HTTP CORS 基础配置：`HTTP_CORS_ALLOWED_ORIGINS`

下一步：

- 需要真实 R2 验收时运行 `.\scripts\verify-current-baseline.ps1 -R2Mode Require`；后续新增 schema 时追加下一个 migration 版本并运行 `.\scripts\verify-migration-contract.ps1`；后续新增或调整配置时同时运行配置清单、配置语义校验和 `go test ./internal/platform/config`；进入新的 feed、vote、moderation、notification 或 search 产品语义前需要先确认边界。
- 当前仍不做富文本 HTML、任意 iframe、embed 播放器、评论投票、通知扩展、搜索扩展、编辑历史、草稿、附件重新绑定或对象物理删除。

## 接口

当前可用接口：

```text
GET  /healthz
POST /api/v1/auth/register
POST /api/v1/auth/login
GET  /api/v1/me
GET  /api/v1/communities
GET  /api/v1/communities/:slug
POST /api/v1/community-applications
POST /api/v1/community-applications/:id/approve
POST /api/v1/community-applications/:id/reject
POST /api/v1/communities/:slug/posts
GET  /api/v1/communities/:slug/posts
GET  /api/v1/posts
GET  /api/v1/posts/:id
PATCH /api/v1/posts/:id
DELETE /api/v1/posts/:id
POST /api/v1/posts/:id/comments
GET  /api/v1/posts/:id/comments
PATCH /api/v1/comments/:id
DELETE /api/v1/comments/:id
PUT  /api/v1/posts/:id/vote
DELETE /api/v1/posts/:id/vote
POST /api/v1/posts/:id/reports
POST /api/v1/comments/:id/reports
POST /api/v1/posts/:id/moderation/remove
POST /api/v1/comments/:id/moderation/remove
GET  /api/v1/moderation/reports
GET  /api/v1/moderation/reports/:id
POST /api/v1/moderation/reports/:id/dismiss
POST /api/v1/moderation/reports/:id/remove-target
GET  /api/v1/search
GET  /api/v1/notifications
POST /api/v1/notifications/:id/read
POST /api/v1/uploads/images
```

注册请求：

```bash
curl -i -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"alice\",\"password\":\"password123\"}"
```

成功响应返回 `access_token`、`token_type`、`expires_in` 和用户公开信息。响应不会包含 `password_hash`。

登录请求：

```bash
curl -i -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"alice\",\"password\":\"password123\"}"
```

成功响应结构与注册一致。

当前用户请求：

```bash
curl -i http://localhost:8080/api/v1/me \
  -H "Authorization: Bearer <access_token>"
```

成功响应返回当前用户公开信息：

```json
{
  "id": "uuid",
  "username": "alice",
  "status": "active",
  "created_at": "2026-05-30T00:00:00Z"
}
```

未带 token、token 格式错误、token 无效或 token 对应用户不存在时，统一返回 `401 unauthenticated`。

社区列表请求：

```bash
curl -i http://localhost:8080/api/v1/communities \
  -H "Authorization: Bearer <access_token>"
```

社区详情请求：

```bash
curl -i http://localhost:8080/api/v1/communities/public \
  -H "Authorization: Bearer <access_token>"
```

提交社区申请：

```bash
curl -i -X POST http://localhost:8080/api/v1/community-applications \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d "{\"requested_slug\":\"campus-life\",\"requested_name\":\"Campus Life\",\"reason\":\"Need a campus board\"}"
```

批准社区申请：

```bash
curl -i -X POST http://localhost:8080/api/v1/community-applications/<application_id>/approve \
  -H "Authorization: Bearer <staff_access_token>"
```

拒绝社区申请：

```bash
curl -i -X POST http://localhost:8080/api/v1/community-applications/<application_id>/reject \
  -H "Authorization: Bearer <staff_access_token>" \
  -H "Content-Type: application/json" \
  -d "{\"reject_reason\":\"duplicate slug\"}"
```

阶段 2 暂不提供 staff 管理接口。本地 demo 可以通过 SQL 设置审批者：

```sql
UPDATE users SET is_platform_staff = true WHERE username = 'reviewer';
```

发布帖子：

```bash
curl -i -X POST http://localhost:8080/api/v1/communities/public/posts \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d "{\"title\":\"Hello campus\",\"body\":\"First post body\"}"
```

阶段 13 起，发帖请求支持可选图片附件：

```json
{
  "title": "Hello campus",
  "body": "Markdown-like post body",
  "attachment_ids": ["uuid"]
}
```

社区帖子列表：

```bash
curl -i "http://localhost:8080/api/v1/communities/public/posts?sort=new&limit=20&offset=0" \
  -H "Authorization: Bearer <access_token>"
```

全站最新帖子流：

```bash
curl -i "http://localhost:8080/api/v1/posts?sort=new&limit=20&offset=0" \
  -H "Authorization: Bearer <access_token>"
```

全站 hot 帖子流：

```bash
curl -i "http://localhost:8080/api/v1/posts?sort=hot&limit=20&offset=0" \
  -H "Authorization: Bearer <access_token>"
```

社区 hot 帖子列表：

```bash
curl -i "http://localhost:8080/api/v1/communities/public/posts?sort=hot&limit=20&offset=0" \
  -H "Authorization: Bearer <access_token>"
```

不传 `sort` 时默认按 `new` 排序。`sort=hot` 使用现有帖子投票事实排序，按 `score DESC, upvote_count DESC, created_at DESC, id DESC`。帖子列表、帖子详情和全站帖子流都会返回 `upvote_count`、`downvote_count`、`score`、`my_vote`。`my_vote` 为 `1`、`-1` 或 `0`。

帖子详情：

```bash
curl -i http://localhost:8080/api/v1/posts/<post_id> \
  -H "Authorization: Bearer <access_token>"
```

编辑帖子：

```bash
curl -i -X PATCH http://localhost:8080/api/v1/posts/<post_id> \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d "{\"title\":\"Updated title\",\"body\":\"Updated Markdown-like body\"}"
```

软删除帖子：

```bash
curl -i -X DELETE http://localhost:8080/api/v1/posts/<post_id> \
  -H "Authorization: Bearer <access_token>"
```

只有作者可以编辑或删除自己的 visible 帖子。删除会把帖子状态改为 `deleted`，普通读取入口不再返回该帖子；不会物理删除附件或 R2 对象。

发布评论：

```bash
curl -i -X POST http://localhost:8080/api/v1/posts/<post_id>/comments \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d "{\"body\":\"First comment\"}"
```

阶段 13 起，评论请求继续支持 `parent_id`，并支持可选图片附件：

```json
{
  "body": "Markdown-like comment body",
  "parent_id": null,
  "attachment_ids": ["uuid"]
}
```

帖子评论列表：

```bash
curl -i "http://localhost:8080/api/v1/posts/<post_id>/comments?limit=20&offset=0" \
  -H "Authorization: Bearer <access_token>"
```

评论树读取：

```bash
curl -i "http://localhost:8080/api/v1/posts/<post_id>/comments?view=tree&sort=new&limit=20&offset=0&max_depth=6" \
  -H "Authorization: Bearer <access_token>"
```

`view=tree` 返回扁平前序遍历数组，根评论 `parent_id` 返回 `null`，comment 响应包含 `body_format`、`depth`、`reply_count` 和 `has_more_replies`。

编辑评论：

```bash
curl -i -X PATCH http://localhost:8080/api/v1/comments/<comment_id> \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d "{\"body\":\"Updated Markdown-like comment body\"}"
```

软删除评论：

```bash
curl -i -X DELETE http://localhost:8080/api/v1/comments/<comment_id> \
  -H "Authorization: Bearer <access_token>"
```

只有作者可以编辑或删除自己的 visible 评论。删除会把评论状态改为 `deleted`，普通评论列表和 tree view 不再返回该评论；不会物理删除附件或 R2 对象。

图片上传：

```bash
curl -i -X POST http://localhost:8080/api/v1/uploads/images \
  -H "Authorization: Bearer <access_token>" \
  -F "file=@image.png" \
  -F "alt_text=campus photo"
```

图片上传必须登录，生产/主方案文件进入 Cloudflare R2；local fallback 只用于本地无凭据验证。后端不接受用户提交任意第三方图片 URL 作为附件。

帖子 upvote：

```bash
curl -i -X PUT http://localhost:8080/api/v1/posts/<post_id>/vote \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d "{\"value\":1}"
```

帖子 downvote：

```bash
curl -i -X PUT http://localhost:8080/api/v1/posts/<post_id>/vote \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d "{\"value\":-1}"
```

取消帖子投票：

```bash
curl -i -X DELETE http://localhost:8080/api/v1/posts/<post_id>/vote \
  -H "Authorization: Bearer <access_token>"
```

举报帖子：

```bash
curl -i -X POST http://localhost:8080/api/v1/posts/<post_id>/reports \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d "{\"reason\":\"spam or abuse\"}"
```

举报评论：

```bash
curl -i -X POST http://localhost:8080/api/v1/comments/<comment_id>/reports \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d "{\"reason\":\"spam or abuse\"}"
```

移除帖子：

```bash
curl -i -X POST http://localhost:8080/api/v1/posts/<post_id>/moderation/remove \
  -H "Authorization: Bearer <staff_access_token>" \
  -H "Content-Type: application/json" \
  -d "{\"reason\":\"policy violation\"}"
```

移除评论：

```bash
curl -i -X POST http://localhost:8080/api/v1/comments/<comment_id>/moderation/remove \
  -H "Authorization: Bearer <staff_access_token>" \
  -H "Content-Type: application/json" \
  -d "{\"reason\":\"policy violation\"}"
```

阶段 7 暂不提供 staff 管理接口。本地 demo 可以继续通过 SQL 设置平台 staff：

```sql
UPDATE users SET is_platform_staff = true WHERE username = 'reviewer';
```

举报列表：

```bash
curl -i "http://localhost:8080/api/v1/moderation/reports?status=pending&limit=20&offset=0" \
  -H "Authorization: Bearer <staff_access_token>"
```

不传 `status` 时默认只返回 `pending` 举报。非 staff 返回 `403 forbidden`。

举报详情：

```bash
curl -i http://localhost:8080/api/v1/moderation/reports/<report_id> \
  -H "Authorization: Bearer <staff_access_token>"
```

dismiss 举报：

```bash
curl -i -X POST http://localhost:8080/api/v1/moderation/reports/<report_id>/dismiss \
  -H "Authorization: Bearer <staff_access_token>"
```

dismiss 只处理 `pending` 举报，成功后写入 `dismissed`、`reviewed_by`、`reviewed_at` 和 `updated_at`，不新增 `moderation_actions.action=dismiss`。

基于举报移除目标内容：

```bash
curl -i -X POST http://localhost:8080/api/v1/moderation/reports/<report_id>/remove-target \
  -H "Authorization: Bearer <staff_access_token>" \
  -H "Content-Type: application/json" \
  -d "{\"reason\":\"policy violation\"}"
```

remove-target 只处理 `pending` 举报，并复用内容移除事务：移除帖子或评论、写入 `moderation_actions(action=remove)`、解决同 target 的 pending 举报在同一 PostgreSQL 事务内完成。

## 本地运行

### 1. 准备配置

```bash
cp .env.example .env
```

Windows PowerShell:

```powershell
Copy-Item .env.example .env
```

### 2. 启动 PostgreSQL

```bash
docker compose up -d postgres
```

### 3. 执行 migration

```bash
go run ./cmd/migrate up
```

查看版本：

```bash
go run ./cmd/migrate version
```

### 4. 启动 API

```bash
go run ./cmd/api
```

默认监听 `.env` 中的 `HTTP_ADDR`，当前示例配置是 `:8080`。

浏览器前端跨域访问可配置 `HTTP_CORS_ALLOWED_ORIGINS`，例如：

```env
HTTP_CORS_ALLOWED_ORIGINS=http://localhost:5173
```

多个 origin 用英文逗号分隔。空值表示不启用 CORS，`*` 表示允许任意来源。

阶段 13 新增 Cloudflare R2 配置。生产/主方案对象存储直接使用 R2；本地如果没有 R2 凭据，可以使用 local fallback 验证上传流程。

```env
OBJECT_STORAGE_PROVIDER=r2
OBJECT_STORAGE_ENDPOINT=https://<account-id>.r2.cloudflarestorage.com
OBJECT_STORAGE_REGION=auto
OBJECT_STORAGE_BUCKET=
OBJECT_STORAGE_ACCESS_KEY_ID=
OBJECT_STORAGE_SECRET_ACCESS_KEY=
OBJECT_STORAGE_PUBLIC_BASE_URL=
OBJECT_STORAGE_FORCE_PATH_STYLE=true
OBJECT_STORAGE_LOCAL_ROOT=var/uploads
UPLOAD_IMAGE_MAX_BYTES=5242880
UPLOAD_IMAGE_MAX_COUNT_PER_POST=9
UPLOAD_IMAGE_MAX_COUNT_PER_COMMENT=1
```

### 5. 健康检查

```bash
curl http://localhost:8080/healthz
```

预期：

```json
{"status":"ok"}
```

## 测试

```bash
go test ./...
go build -buildvcs=false ./...
```

普通 `go build ./...` 在部分本机环境可能受 Git safe.directory / VCS stamping 影响，可用 `-buildvcs=false` 验证代码构建。

当前基线验收入口：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-current-baseline.ps1
```

该脚本会依次执行 API 路由/认证边界契约校验、API schema 契约校验、HTTP 错误契约校验、配置清单契约校验、配置语义契约校验、migration 契约校验、测试、构建、migration、Stage 13 内容系统 smoke、Stage 14 内容生命周期 smoke 和 Stage 15 R2 smoke/凭据门禁。默认 `-R2Mode SkipWhenMissing`：没有 R2 dev bucket 凭据时只验证 skipped 分支；如果当前环境或 `.env` 中存在 R2 dev bucket 凭据，则会执行真实 R2 上传并在 dev bucket 留下测试对象。

API 契约路由清单与认证边界校验：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-api-contract-doc.ps1
```

该脚本会从源码路由注册提取当前方法、路径和 Auth 边界，并与 `docs/internal/architecture/http-api-contract.md` 的路由表比对。Auth 边界来自 `/healthz` public route、local-only static route、auth public group 和 `authhttp.RequireAuth` 保护分组；它不校验完整请求/响应 schema，也不校验每个业务权限场景的 staff、作者或资源可见性判断。

HTTP API schema 契约校验：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-api-schema-doc.ps1
```

该脚本会扫描 delivery 层 handler JSON struct，并与 `docs/internal/architecture/http-api-schema.md` 的 package、Go type 和 JSON 字段清单比对；同时读取 `docs/internal/architecture/http-api-contract.md` 的路由表，校验 schema 文档中的接口映射覆盖当前路由、没有过期路由、schema 引用真实存在，并约束成功状态码为 `200/201/204`。它不生成 OpenAPI，不校验完整业务枚举、数值范围或错误消息全文。

HTTP 错误契约校验：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-http-error-contract-doc.ps1
```

该脚本会从 `internal/apperr/apperr.go`、`internal/platform/httpserver/error.go` 和 `docs/internal/architecture/http-error-handling.md` 比对错误码集合、HTTP 状态码和错误响应形状。它不新增错误码，不改变错误响应格式。

配置契约清单校验：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-config-contract-doc.ps1
```

该脚本会从 `internal/platform/config/load.go` 提取当前加载的环境变量名，并与 `.env.example` 和 `docs/internal/engineering/configuration.md` 比对。它只校验变量名集合，不校验完整默认值或枚举语义。

配置语义契约校验：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-config-semantics-doc.ps1
```

该脚本会校验 `docs/internal/engineering/configuration.md` 中的必需性、默认值和枚举说明是否与 `internal/platform/config/load.go`、`internal/platform/config/validate.go` 同步。它不校验完整数值范围、跨字段约束或错误消息文本。

配置加载运行时契约测试：

```powershell
go test ./internal/platform/config -run TestLoad -v
```

该测试覆盖 `Load()` 的 local 默认值、R2 配置加载、R2 缺凭据失败和基础解析失败路径。

migration 契约清单校验：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-migration-contract.ps1
```

该脚本会校验 `migrations/` 中的版本编号连续、up/down 成对、命名一致，并与 `docs/internal/engineering/migrations.md` 的清单比对。它不校验 SQL 语义或 down migration 的可逆性。

常用快速检查：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-current-baseline.ps1 -SkipHttpSmoke -R2Mode Skip
```

需要强制真实 R2 上传验证时：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-current-baseline.ps1 -R2Mode Require
```

阶段 13 内容系统本地 smoke：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\smoke-stage-13-content-system.ps1
```

该脚本会执行 migration，临时启动 API，使用 local object storage fallback 验证图片上传、帖子图片绑定、根评论/子评论图片绑定、评论树读取和非法 `view` 失败路径。真实 R2 上传需要单独提供 R2 dev bucket 凭据。

阶段 14 内容生命周期本地 smoke：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\smoke-stage-14-content-lifecycle.ps1
```

该脚本会执行 migration，临时启动 API，验证帖子编辑、帖子删除、评论编辑、评论删除、非作者 forbidden 和删除后读取 `not_found`。

阶段 15 R2 真实上传 smoke：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\smoke-stage-15-r2-upload.ps1
```

运行前需要在当前环境或 `.env` 中提供 R2 dev bucket 配置：

```env
OBJECT_STORAGE_ENDPOINT=https://<account-id>.r2.cloudflarestorage.com
OBJECT_STORAGE_REGION=auto
OBJECT_STORAGE_BUCKET=<dev-bucket>
OBJECT_STORAGE_ACCESS_KEY_ID=<dev-access-key-id>
OBJECT_STORAGE_SECRET_ACCESS_KEY=<dev-secret-access-key>
OBJECT_STORAGE_PUBLIC_BASE_URL=https://<public-dev-media-base>
OBJECT_STORAGE_FORCE_PATH_STYLE=true
```

该脚本会强制使用 `OBJECT_STORAGE_PROVIDER=r2` 启动临时 API，注册用户，上传 1x1 PNG 到 R2，并把返回的 attachment 绑定到帖子。脚本不会输出 access key 或 secret key。它会在 dev bucket 留下测试对象；本阶段不做对象清理。

当前没有 R2 凭据时可只验证缺凭据分支：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\smoke-stage-15-r2-upload.ps1 -SkipWhenMissingCredentials
```

## 文档

- `tasks.md`：当前阶段工单板
- `docs/internal/README.md`：内部文档索引
- `docs/internal/architecture/project-baseline.md`：架构基线
- `docs/internal/architecture/auth-user-v1.md`：阶段 1 认证设计
- `docs/internal/architecture/community-v1.md`：V1 社区业务架构与阶段 2 社区边界
- `docs/internal/architecture/http-api-contract.md`：当前 HTTP API 路由、认证边界和错误语义快照
- `docs/internal/architecture/http-api-schema.md`：当前 HTTP API request/response schema、接口 schema 映射和 handler JSON 字段清单快照
- `docs/internal/architecture/content-system.md`：阶段 13 内容系统增强边界
- `docs/internal/architecture/media-storage.md`：Cloudflare R2 媒体存储边界
- `docs/internal/architecture/content-lifecycle.md`：阶段 14 内容编辑与软删除边界
- `docs/internal/engineering/workflow.md`：阶段、工单、分支、提交和目标模式长跑规则
- `docs/internal/engineering/migrations.md`：migration 命名、配对和清单校验规则
- `docs/internal/engineering/stage-1-auth-playbook.md`：阶段 1 认证实现手册

## 当前限制

- 阶段 1 暂不做 refresh token、多端会话、第三方登录、邮箱验证、找回密码和复杂 RBAC。
- `/me` 只返回当前用户基础公开信息，不做资料编辑、头像、邮箱和权限列表。
- 认证中间件只验证 Bearer access token 并写入当前用户 ID，不查询数据库。
- 阶段 2 已落地社区申请与审批闭环，但暂不做 staff 管理接口、申请列表、申请取消、私密社区、邀请制和复杂成员加入/退出流程。
- 阶段 3 帖子主链路暂不做图片上传、标签、编辑、删除、草稿、搜索和审核台。
- 阶段 4 评论主链路暂不做评论树优化、评论编辑、删除、评论投票、审核台和通知。
- 阶段 6 已完成帖子 upvote/downvote 和全站最新流；暂不做 hot feed、推荐排序、评论投票、投票通知和防刷策略。
- 阶段 7 已完成轻量举报与平台 staff 移除内容闭环。
- 阶段 8 已完成审核台最小闭环；暂不做审核后台 UI、社区 moderator 权限、通知、防刷、自动审核、申诉、批量处理或 target 内容预览增强。
- 阶段 9 已完成 hot feed / 内容分发；暂不做个性化推荐、预计算时间线、推荐系统、反作弊、评论投票、通知或审核台增强。
- 阶段 13 已完成内容系统增强；暂不做前端 UI、富文本 HTML 编辑器、任意 HTML、任意 iframe、embed 播放器、评论投票、通知扩展、搜索扩展、生产真实密钥配置或对象物理删除任务。
- 阶段 14 已完成内容编辑与软删除；暂不做编辑历史、草稿、恢复、附件重新绑定、R2 对象物理删除、搜索索引刷新、通知扩展或审核动作扩展。
- 阶段 15 已补齐 R2 真实凭据 smoke 工具；真实 R2 上传需要 R2 dev bucket 凭据，本阶段不配置生产密钥或清理 R2 对象。
- 阶段 16 只补当前基线验收入口；不新增业务接口、schema migration、第三方依赖或新产品语义。
- 阶段 17 只补 HTTP API 契约快照和路由清单校验；不新增业务接口、不生成完整 OpenAPI、不改变错误码或响应格式。
- 阶段 18 只补配置契约清单校验；不新增配置变量、不改变运行时配置加载语义、不写入真实 R2 凭据。
- 阶段 19 只补 migration 契约和清单校验；不新增或修改 schema migration，不改变 migration runner 行为。
- 阶段 20 只补配置语义契约校验；不新增配置变量，不改变运行时配置语义，不写入真实 R2 凭据。
- 阶段 21 只补配置加载运行时契约测试；不新增配置变量，不改变运行时配置语义，不写入真实 R2 凭据。
- 阶段 22 只补 HTTP API schema 契约快照和 handler JSON 字段清单校验；不新增业务接口，不生成 OpenAPI，不改变错误码或响应格式。
- 阶段 23 只补 HTTP 错误契约校验；不新增错误码，不改变错误响应格式，不改变认证错误语义。
- 阶段 24 只补 API schema 路由映射校验；不新增业务接口，不生成 OpenAPI，不改变成功或错误响应格式。
- 阶段 25 只补 API 认证边界契约校验；不新增业务接口，不改变认证中间件语义或成功/错误响应格式。
- `/healthz` 只表示进程存活，不做数据库 readiness 检查。

## License

[MIT License](./LICENSE)
