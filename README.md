# CUMT Nexus API

`cumt-nexus-api` 是一个社区内容平台后端。当前采用 Go + Gin + PostgreSQL，架构形态是模块化单体。

## 当前状态

阶段：`阶段 9 hot feed / 内容分发已完成`

代码已完成阶段 1 认证与用户基础闭环、阶段 2 社区申请与审批闭环、阶段 3 帖子发布和读取闭环、阶段 4 评论发布和读取闭环、阶段 5 完整真实冒烟、阶段 6 全站最新帖子流 + 帖子 upvote/downvote 基础、阶段 7 轻量举报与平台 staff 移除内容闭环、阶段 8 审核台最小闭环，以及阶段 9 hot feed / 内容分发闭环。

阶段 9 已完成：在已有全站最新流和帖子投票事实之上，补齐最小 hot feed / 内容分发能力。全站帖子流和社区帖子列表支持 `sort=new|hot`，默认保持 `new`，`hot` 使用现有投票事实做简化热度排序。

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
- `GET /api/v1/communities/:slug/posts`
- `GET /api/v1/posts/:id`
- 阶段 4 评论 schema、domain、repository
- `POST /api/v1/posts/:id/comments`
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
- `GET /api/v1/posts?sort=new|hot`
- `GET /api/v1/communities/:slug/posts?sort=new|hot`
- 移除内容和审核动作写入同一 PostgreSQL 事务
- HTTP CORS 基础配置：`HTTP_CORS_ALLOWED_ORIGINS`

下一步：

- 阶段 10 进入审核台增强。
- 后续目标顺序是：审核台增强、搜索、通知。
- 当前仍不做个性化推荐、预计算时间线、推荐系统、反作弊、评论投票、搜索或通知。

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
POST /api/v1/posts/:id/comments
GET  /api/v1/posts/:id/comments
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

发布评论：

```bash
curl -i -X POST http://localhost:8080/api/v1/posts/<post_id>/comments \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d "{\"body\":\"First comment\"}"
```

帖子评论列表：

```bash
curl -i "http://localhost:8080/api/v1/posts/<post_id>/comments?limit=20&offset=0" \
  -H "Authorization: Bearer <access_token>"
```

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

## 文档

- `tasks.md`：当前阶段工单板
- `docs/internal/README.md`：内部文档索引
- `docs/internal/architecture/project-baseline.md`：架构基线
- `docs/internal/architecture/auth-user-v1.md`：阶段 1 认证设计
- `docs/internal/architecture/community-v1.md`：V1 社区业务架构与阶段 2 社区边界
- `docs/internal/engineering/workflow.md`：阶段、工单、分支、提交和目标模式长跑规则
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
- 阶段 9 正在推进 hot feed / 内容分发；暂不做个性化推荐、预计算时间线、推荐系统、反作弊、评论投票、通知或审核台增强。
- `/healthz` 只表示进程存活，不做数据库 readiness 检查。

## License

[MIT License](./LICENSE)
