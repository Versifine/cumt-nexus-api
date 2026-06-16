# HTTP API 契约快照

本文记录当前 HTTP API 路由、认证边界和全局错误响应约定。它是当前路由契约快照，不替代 handler/usecase 测试，也不是完整 OpenAPI schema。当前 request/response schema 快照见 `docs/contracts/http-api-schema.md`。

路由清单、Auth 边界和查询参数清单需要通过以下脚本与源码中的 `RegisterRoutes`、`cmd/api/main.go` 路由分组、handler query 读取保持同步：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-api-contract-doc.ps1
```

## 全局约定

- `/healthz` 不需要认证，只表示进程存活。
- `/api/v1/auth/register`、`/api/v1/auth/login`、邮箱验证码发送、邮箱验证码注册/登录和找回密码入口不需要认证。
- `GET /api/v1/posts`、`GET /api/v1/posts/:id`、`GET /api/v1/communities/:slug/posts`、`GET /api/v1/posts/:id/comments`、`GET /api/v1/communities`、`GET /api/v1/communities/:slug`、`GET /api/v1/users/:username`、`GET /api/v1/users/:username/posts`、`GET /api/v1/users/:username/comments`、`GET /api/v1/search` 和 `GET /api/v1/effects/catalog` 支持匿名读取公开 visible 内容，也支持可选 Bearer 读取当前用户视角字段。
- 除 auth 入口和公开读取入口外，当前 `/api/v1` 业务接口都需要 Bearer access token。
- `GET /uploads/*filepath` 只在 `OBJECT_STORAGE_PROVIDER=local` 时注册，用于本地 local storage fallback 文件访问；生产/主方案使用 Cloudflare R2 public base URL。
- 错误响应统一为 `{"error":{"code":"...","message":"..."}}`。
- 认证失败统一返回 `unauthenticated`。
- 可选 Bearer 的公开读取接口：无 `Authorization` 时按匿名读取且 `my_vote=0`、`is_saved=false`、`viewer_is_following=false`、权限对象为匿名态；有合法 Bearer 时返回当前用户 `my_vote`、`is_saved`、`viewer_is_following` 和 `viewer_permissions`；有格式错误、过期或签名错误的 Bearer 时仍返回 `unauthenticated`。

该脚本校验 method/path 清单，也校验 Auth 列是否与源码中的 public route、local-only static route、`authhttp.OptionalAuth` 可选认证分组和 `authhttp.RequireAuth` 保护分组一致；同时扫描 handler 中的 `c.Query(...)` 和 `parseOptionalIntQuery(c, "...")` 等 query key 读取，校验“查询参数约定”表没有缺失、过期或参数集合漂移。它不定义 request/response JSON schema，不校验每个业务权限场景的 staff/作者/资源可见性判断，也不校验查询参数枚举值或数值范围。

## 路由清单

| Method | Path | Auth | 说明 |
|---|---|---|---|
| GET | /healthz | public | 进程存活检查 |
| GET | /uploads/*filepath | public, local only | local storage fallback 静态文件 |
| POST | /api/v1/auth/register | public | 注册并签发 access token |
| POST | /api/v1/auth/email-codes/register | public | 发送注册邮箱验证码 |
| POST | /api/v1/auth/register-with-email | public | 使用邮箱验证码注册并签发 access token |
| POST | /api/v1/auth/login | public | 登录并签发 access token |
| POST | /api/v1/auth/email-codes/login | public | 发送邮箱登录验证码 |
| POST | /api/v1/auth/login-with-email-code | public | 使用邮箱验证码登录并签发 access token |
| POST | /api/v1/auth/email-codes/password-reset | public | 发送找回密码验证码 |
| POST | /api/v1/auth/password-reset | public | 使用邮箱验证码重置密码 |
| GET | /api/v1/me | Bearer | 当前用户 |
| PATCH | /api/v1/me/profile | Bearer | 当前用户更新公开资料 |
| GET | /api/v1/me/security | Bearer | 当前用户账号安全信息 |
| POST | /api/v1/me/security/email-codes/change-email | Bearer | 发送修改邮箱验证码 |
| POST | /api/v1/me/security/email-codes/delete-account | Bearer | 发送注销账号确认验证码 |
| PATCH | /api/v1/me/security/email | Bearer | 修改当前用户绑定邮箱 |
| PATCH | /api/v1/me/security/password | Bearer | 修改当前用户密码 |
| DELETE | /api/v1/me/account | Bearer | 注销当前用户账号 |
| POST | /api/v1/auth/logout-all | Bearer | 当前用户退出所有会话 |
| GET | /api/v1/me/saved-posts | Bearer | 当前用户保存的公开帖子 |
| GET | /api/v1/me/followed-communities | Bearer | 当前用户关注的公开社区 |
| GET | /api/v1/me/community-owner-transfers | Bearer | 当前用户作为目标账号的社区负责人交接收件箱 |
| GET | /api/v1/me/followed-users | Bearer | 当前用户关注的 active 用户 |
| GET | /api/v1/me/points | Bearer | 当前用户积分账户 |
| GET | /api/v1/me/point-transactions | Bearer | 当前用户积分流水 |
| GET | /api/v1/me/progression | Bearer | 当前用户全站等级和经验概览 |
| GET | /api/v1/me/xp-events | Bearer | 当前用户经验事件流水 |
| GET | /api/v1/me/titles | Bearer | 当前用户可展示头衔列表 |
| PATCH | /api/v1/me/title | Bearer | 当前用户选择或清空展示头衔 |
| GET | /api/v1/users/:username | optional Bearer | 公开用户主页 |
| POST | /api/v1/users/:username/follow | Bearer | 关注用户 |
| DELETE | /api/v1/users/:username/follow | Bearer | 取消关注用户 |
| GET | /api/v1/communities | optional Bearer | 社区列表 |
| GET | /api/v1/communities/:slug | optional Bearer | 社区详情 |
| GET | /api/v1/communities/:slug/manage | Bearer | 社区管理上下文 |
| GET | /api/v1/communities/:slug/manage/posts | Bearer | 社区管理帖子列表 |
| GET | /api/v1/communities/:slug/manage/comments | Bearer | 社区管理评论列表 |
| GET | /api/v1/communities/:slug/manage/reports | Bearer | 社区管理举报列表 |
| GET | /api/v1/communities/:slug/manage/members | Bearer | 社区管理成员列表 |
| GET | /api/v1/communities/:slug/manage/settings | Bearer | 社区管理设置读取 |
| PATCH | /api/v1/communities/:slug/manage/settings | Bearer | 社区 owner 更新基础设置 |
| GET | /api/v1/communities/:slug/manage/rules | Bearer | 社区规则列表 |
| POST | /api/v1/communities/:slug/manage/rules | Bearer | 社区 owner/moderator 创建规则 |
| PATCH | /api/v1/communities/:slug/manage/rules/:rule_id | Bearer | 社区 owner/moderator 更新规则 |
| DELETE | /api/v1/communities/:slug/manage/rules/:rule_id | Bearer | 社区 owner/moderator 删除规则 |
| POST | /api/v1/communities/:slug/follow | Bearer | 关注社区 |
| DELETE | /api/v1/communities/:slug/follow | Bearer | 取消关注社区 |
| POST | /api/v1/community-applications | Bearer | 提交社区申请 |
| GET | /api/v1/community-applications | Bearer | 平台 staff 查看社区申请列表 |
| GET | /api/v1/community-applications/:id | Bearer | 平台 staff 查看社区申请详情 |
| POST | /api/v1/community-applications/:id/approve | Bearer | 平台 staff 审批通过社区申请 |
| POST | /api/v1/community-applications/:id/reject | Bearer | 平台 staff 拒绝社区申请 |
| GET | /api/v1/admin/users | Bearer | 平台 staff 用户管理列表 |
| PATCH | /api/v1/admin/users/:id | Bearer | 平台 owner/admin 按角色边界更新用户状态或平台 staff 标记 |
| PATCH | /api/v1/admin/users/:id/platform-role | Bearer | 平台 owner 更新非 owner 用户的平台角色 |
| GET | /api/v1/admin/owner-transfer | Bearer | 平台 owner/admin 查看当前站点负责人交接 |
| POST | /api/v1/admin/owner-transfer | Bearer | 当前平台 owner 发起站点负责人交接 |
| DELETE | /api/v1/admin/owner-transfer/:transfer_id | Bearer | 发起人或当前平台 owner 取消站点负责人交接 |
| GET | /api/v1/admin/communities | Bearer | 平台 staff 社区管理列表 |
| PATCH | /api/v1/admin/communities/:id | Bearer | 平台 staff 更新社区状态 |
| GET | /api/v1/admin/effects | Bearer | 平台 staff 评论效果管理列表 |
| PATCH | /api/v1/admin/effects/:id | Bearer | 平台 staff 启用或停用评论效果 |
| GET | /api/v1/admin/settings | Bearer | 平台 staff 读取平台运行开关 |
| PATCH | /api/v1/admin/settings/:key | Bearer | 平台 staff 更新平台运行开关 |
| GET | /api/v1/admin/audit-logs | Bearer | 平台 staff 查看平台管理审计日志 |
| GET | /api/v1/admin/point-transactions | Bearer | 平台 staff 查看积分流水 |
| POST | /api/v1/admin/users/:id/points/adjust | Bearer | 平台 staff 手工调整用户积分 |
| POST | /api/v1/admin/users/:id/sanctions | Bearer | 平台 owner/admin 创建用户账号处罚 |
| GET | /api/v1/admin/users/:id/sanctions | Bearer | 平台 staff 查看用户账号处罚记录 |
| POST | /api/v1/admin/user-sanctions/:sanction_id/revoke | Bearer | 平台 owner/admin 解除用户账号处罚 |
| GET | /api/v1/admin/titles | Bearer | 平台 staff 查看头衔目录 |
| POST | /api/v1/admin/titles | Bearer | 平台 staff 创建头衔 |
| PATCH | /api/v1/admin/titles/:id | Bearer | 平台 staff 更新头衔 |
| GET | /api/v1/admin/users/:id/titles | Bearer | 平台 staff 查看用户头衔授予 |
| POST | /api/v1/admin/users/:id/titles | Bearer | 平台 staff 授予用户头衔 |
| DELETE | /api/v1/admin/users/:id/titles/:grant_id | Bearer | 平台 staff 撤销用户头衔 |
| GET | /api/v1/owner-transfer/:transfer_id | Bearer | 目标用户或当前平台 owner 查看站点负责人交接 |
| POST | /api/v1/owner-transfer/:transfer_id/accept | Bearer | 目标用户接受站点负责人交接 |
| GET | /api/v1/communities/:slug/posts | optional Bearer | 社区帖子列表 |
| GET | /api/v1/posts | optional Bearer | 全站帖子流 |
| GET | /api/v1/posts/:id | optional Bearer | 帖子详情 |
| GET | /api/v1/users/:username/posts | optional Bearer | 用户公开帖子列表 |
| POST | /api/v1/communities/:slug/posts | Bearer | 发帖 |
| PATCH | /api/v1/posts/:id | Bearer | 作者编辑帖子 |
| DELETE | /api/v1/posts/:id | Bearer | 作者软删除帖子 |
| POST | /api/v1/posts/:id/save | Bearer | 保存帖子 |
| DELETE | /api/v1/posts/:id/save | Bearer | 取消保存帖子 |
| POST | /api/v1/posts/:id/comments | Bearer | 发布评论 |
| GET | /api/v1/posts/:id/comments | optional Bearer | 帖子评论列表或 tree view |
| GET | /api/v1/users/:username/comments | optional Bearer | 用户公开评论列表 |
| GET | /api/v1/search | optional Bearer | PostgreSQL 基础搜索 |
| PATCH | /api/v1/comments/:id | Bearer | 作者编辑评论 |
| DELETE | /api/v1/comments/:id | Bearer | 作者软删除评论 |
| POST | /api/v1/comments/:id/effects | Bearer | 给评论应用积分效果 |
| PUT | /api/v1/comments/:id/vote | Bearer | 设置评论投票 |
| DELETE | /api/v1/comments/:id/vote | Bearer | 取消评论投票 |
| PUT | /api/v1/posts/:id/vote | Bearer | 设置帖子投票 |
| DELETE | /api/v1/posts/:id/vote | Bearer | 取消帖子投票 |
| POST | /api/v1/posts/:id/reports | Bearer | 举报帖子 |
| POST | /api/v1/comments/:id/reports | Bearer | 举报评论 |
| POST | /api/v1/posts/:id/moderation/remove | Bearer | 平台 staff 移除帖子 |
| POST | /api/v1/comments/:id/moderation/remove | Bearer | 平台 staff 移除评论 |
| POST | /api/v1/communities/:slug/moderation/posts/:id/remove | Bearer | 社区 owner/moderator 移除本社区帖子 |
| POST | /api/v1/communities/:slug/moderation/comments/:id/remove | Bearer | 社区 owner/moderator 移除本社区评论 |
| GET | /api/v1/admin/mod-queues | Bearer | 平台级审核队列 |
| POST | /api/v1/admin/mod-queues/actions | Bearer | 平台级审核队列批量动作 |
| GET | /api/v1/communities/:slug/mod-queues | Bearer | 社区级审核队列 |
| POST | /api/v1/communities/:slug/mod-queues/actions | Bearer | 社区级审核队列批量动作 |
| POST | /api/v1/communities/:slug/moderation/posts/:id/approve | Bearer | 社区 owner/moderator 批准帖子 |
| POST | /api/v1/communities/:slug/moderation/comments/:id/approve | Bearer | 社区 owner/moderator 批准评论 |
| POST | /api/v1/communities/:slug/moderation/posts/:id/spam | Bearer | 社区 owner/moderator 标记帖子为 spam |
| POST | /api/v1/communities/:slug/moderation/comments/:id/spam | Bearer | 社区 owner/moderator 标记评论为 spam |
| POST | /api/v1/communities/:slug/moderation/reports/:id/ignore | Bearer | 社区 owner/moderator 忽略本社区举报 |
| POST | /api/v1/communities/:slug/moderation/posts/:id/lock | Bearer | 社区 owner/moderator 锁定或解锁帖子 |
| POST | /api/v1/communities/:slug/moderation/posts/:id/pin | Bearer | 社区 owner/moderator 置顶或取消置顶帖子 |
| POST | /api/v1/communities/:slug/moderation/posts/:id/mark-nsfw | Bearer | 社区 owner/moderator 标记或取消 NSFW |
| POST | /api/v1/communities/:slug/moderation/posts/:id/mark-spoiler | Bearer | 社区 owner/moderator 标记或取消 spoiler |
| POST | /api/v1/communities/:slug/moderation/posts/:id/flair | Bearer | 社区 owner/moderator 设置帖子 flair |
| GET | /api/v1/communities/:slug/moderation/removal-reasons | Bearer | 社区移除原因列表 |
| POST | /api/v1/communities/:slug/moderation/removal-reasons | Bearer | 社区创建移除原因 |
| PATCH | /api/v1/communities/:slug/moderation/removal-reasons/:id | Bearer | 社区更新移除原因 |
| DELETE | /api/v1/communities/:slug/moderation/removal-reasons/:id | Bearer | 社区删除移除原因 |
| POST | /api/v1/communities/:slug/moderation/removal-reasons/:id/apply | Bearer | 应用移除原因并批量移除内容 |
| GET | /api/v1/communities/:slug/moderation/saved-responses | Bearer | 社区保存回复列表 |
| POST | /api/v1/communities/:slug/moderation/saved-responses | Bearer | 社区创建保存回复 |
| PATCH | /api/v1/communities/:slug/moderation/saved-responses/:id | Bearer | 社区更新保存回复 |
| DELETE | /api/v1/communities/:slug/moderation/saved-responses/:id | Bearer | 社区删除保存回复 |
| GET | /api/v1/communities/:slug/manage/banned-users | Bearer | 社区封禁用户列表 |
| POST | /api/v1/communities/:slug/manage/banned-users | Bearer | 社区新增或更新封禁用户 |
| DELETE | /api/v1/communities/:slug/manage/banned-users/:user_id | Bearer | 社区移除封禁用户 |
| GET | /api/v1/communities/:slug/manage/muted-users | Bearer | 社区禁言用户列表 |
| POST | /api/v1/communities/:slug/manage/muted-users | Bearer | 社区新增或更新禁言用户 |
| DELETE | /api/v1/communities/:slug/manage/muted-users/:user_id | Bearer | 社区移除禁言用户 |
| GET | /api/v1/communities/:slug/manage/approved-users | Bearer | 社区批准用户列表 |
| POST | /api/v1/communities/:slug/manage/approved-users | Bearer | 社区新增或更新批准用户 |
| DELETE | /api/v1/communities/:slug/manage/approved-users/:user_id | Bearer | 社区移除批准用户 |
| GET | /api/v1/communities/:slug/moderation/users/:user_id/profile | Bearer | 社区审核用户画像 |
| GET | /api/v1/communities/:slug/moderation/users/:user_id/notes | Bearer | 社区 mod notes 列表 |
| POST | /api/v1/communities/:slug/moderation/users/:user_id/notes | Bearer | 社区新增 mod note |
| DELETE | /api/v1/communities/:slug/moderation/users/:user_id/notes/:note_id | Bearer | 社区删除 mod note |
| GET | /api/v1/communities/:slug/moderation/logs | Bearer | 社区 Mod Log |
| GET | /api/v1/moderation/reports | Bearer | 审核台举报列表 |
| GET | /api/v1/moderation/reports/:id | Bearer | 审核台举报详情 |
| POST | /api/v1/moderation/reports/:id/dismiss | Bearer | 驳回举报 |
| POST | /api/v1/moderation/reports/:id/remove-target | Bearer | 按举报移除目标内容 |
| GET | /api/v1/notifications/unread-summary | Bearer | 当前用户分类未读通知摘要 |
| GET | /api/v1/notifications | Bearer | 当前用户通知列表 |
| POST | /api/v1/notifications/:id/read | Bearer | 标记单条通知已读 |
| POST | /api/v1/notifications/read-all | Bearer | 标记当前用户全部通知已读 |
| GET | /api/v1/messages/summary | Bearer | 当前用户私信未读和请求摘要 |
| GET | /api/v1/messages/conversations | Bearer | 当前用户私信会话列表 |
| POST | /api/v1/messages/conversations | Bearer | 发起私信会话或陌生人请求 |
| GET | /api/v1/messages/conversations/:id/messages | Bearer | 当前用户读取私信会话消息 |
| POST | /api/v1/messages/conversations/:id/messages | Bearer | 当前用户发送私信消息 |
| POST | /api/v1/messages/conversations/:id/read | Bearer | 当前用户清理会话未读游标 |
| POST | /api/v1/messages/conversations/:id/archive | Bearer | 当前用户归档私信会话 |
| DELETE | /api/v1/messages/conversations/:id/archive | Bearer | 当前用户取消归档私信会话 |
| POST | /api/v1/messages/requests/:id/accept | Bearer | 当前用户接受陌生人私信请求 |
| POST | /api/v1/messages/requests/:id/reject | Bearer | 当前用户拒绝陌生人私信请求 |
| POST | /api/v1/messages/:id/recall | Bearer | 发送者撤回私信消息 |
| DELETE | /api/v1/messages/:id | Bearer | 当前用户本地删除私信消息 |
| POST | /api/v1/messages/:id/report | Bearer | 当前用户举报私信消息并提交有限上下文 |
| POST | /api/v1/users/:username/block | Bearer | 当前用户拉黑用户并禁用双方私信发送 |
| DELETE | /api/v1/users/:username/block | Bearer | 当前用户解除拉黑用户 |
| GET | /api/v1/me/privacy/messages | Bearer | 当前用户读取私信隐私设置 |
| PATCH | /api/v1/me/privacy/messages | Bearer | 当前用户更新私信隐私设置 |
| POST | /api/v1/realtime/tickets | Bearer | 当前用户创建私信实时连接 ticket |
| GET | /api/v1/realtime/messages | ticket | 使用实时 ticket 建立私信 WebSocket 事件通道 |
| POST | /api/v1/uploads/images | Bearer | 图片上传 |
| POST | /api/v1/link-previews/resolve | Bearer | 解析公开链接预览 |
| POST | /api/v1/embeds/resolve | Bearer | 解析白名单嵌入内容 |
| GET | /api/v1/effects/catalog | optional Bearer | 公开评论效果目录 |

## 查询参数约定

| 接口 | 参数 | 约定 |
|---|---|---|
| `GET /api/v1/community-applications` | `status`, `limit`, `offset` | `status=pending|approved|rejected`，平台 staff 视角 |
| `GET /api/v1/me/saved-posts` | `limit`, `offset` | 当前用户保存的公开 visible 帖子 |
| `GET /api/v1/me/followed-communities` | `limit`, `offset` | 当前用户关注的 active public 社区 |
| `GET /api/v1/me/community-owner-transfers` | `status`, `limit`, `offset` | 当前用户作为目标账号的社区负责人交接；`status=pending|accepted|cancelled|expired|all`，默认 `pending` |
| `GET /api/v1/me/followed-users` | `limit`, `offset` | 当前用户关注的 active 用户，按关注时间倒序返回 |
| `GET /api/v1/me/point-transactions` | `limit`, `offset` | 当前用户积分流水，按 `created_at DESC, id DESC` 返回 |
| `GET /api/v1/me/xp-events` | `limit`, `offset` | 当前用户经验事件流水，按 `created_at DESC, id DESC` 返回 |
| `GET /api/v1/me/titles` | `limit`, `offset` | 当前用户未撤销、未过期且仍启用的头衔授予 |
| `GET /api/v1/communities` | `limit`, `offset` | 公开社区索引，默认 20，最大 50，只返回 active public 社区 |
| `GET /api/v1/communities/:slug/manage/posts` | `status`, `limit`, `offset` | 社区 owner/moderator 视角；`status=all|visible|removed|deleted|locked|hidden`，默认 `all` |
| `GET /api/v1/communities/:slug/manage/comments` | `status`, `limit`, `offset` | 社区 owner/moderator 视角；`status=all|visible|removed|deleted|locked|hidden`，默认 `all` |
| `GET /api/v1/communities/:slug/manage/reports` | `status`, `limit`, `offset` | 社区 owner/moderator 视角；`status=pending|resolved|dismissed`，默认 `pending` |
| `GET /api/v1/communities/:slug/manage/members` | `limit`, `offset` | 社区 owner/moderator 视角，只返回 active 成员 |
| `GET /api/v1/admin/users` | `status`, `q`, `limit`, `offset` | Platform staff view; `q` searches username or user id |
| `GET /api/v1/admin/communities` | `status`, `q`, `limit`, `offset` | Platform staff view; `q` searches slug, name, description, id or creator id |
| `GET /api/v1/admin/effects` | `active`, `limit`, `offset` | 平台 staff 视角；`active=all|true|false`，默认 `all` |
| `GET /api/v1/admin/audit-logs` | `q`, `target_type`, `target_id`, `limit`, `offset` | 平台 staff 视角；可按关键词、目标类型和目标 ID 过滤 |
| `GET /api/v1/admin/point-transactions` | `user_id`, `limit`, `offset` | 平台 staff 视角；可按用户 ID 过滤，按 `created_at DESC, id DESC` 返回 |
| `GET /api/v1/admin/users/:id/sanctions` | `limit`, `offset` | 平台 staff 视角；查看指定用户处罚记录，按 `created_at DESC, id DESC` 返回 |
| `GET /api/v1/admin/titles` | `scope_type`, `active`, `limit`, `offset` | 平台 staff 视角；`scope_type=all|platform|system|community`，`active=all|true|false` |
| `GET /api/v1/admin/users/:id/titles` | `limit`, `offset` | 平台 staff 视角；查看指定用户当前有效头衔授予 |
| `GET /api/v1/communities/:slug/posts` | `sort`, `t`, `limit`, `offset` | `sort=best|hot|new|top|rising`，`t=hour|day|week|month|year|all`，分页默认由 usecase 收口 |
| `GET /api/v1/posts` | `source`, `sort`, `t`, `limit`, `offset` | `source=all|recommended|following`；`recommended` 当前为公开可解释推荐流，默认 `sort=hot`，匿名使用 `hot + new` 混排并做社区 rank 去重，登录态给关注/互动社区加权；`following` 需要 Bearer，只返回当前用户已关注公开社区内的 visible 帖子；`sort=best|hot|new|top|rising`；`t=hour|day|week|month|year|all` |
| `GET /api/v1/posts/:id/comments` | `view`, `sort`, `limit`, `offset`, `max_depth` | `view=flat|tree`，`sort=best|top|new|old|controversial`，tree view 返回前序遍历扁平数组 |
| `GET /api/v1/users/:username/posts` | `sort`, `limit`, `offset` | `sort=best|hot|new|top|rising`，只返回该用户在公开可读社区中的 visible 帖子 |
| `GET /api/v1/users/:username/comments` | `limit`, `offset` | 只返回该用户在公开可读社区 visible 帖子下的 visible 评论 |
| `GET /api/v1/moderation/reports` | `status`, `limit`, `offset` | 平台 staff 视角 |
| `GET /api/v1/admin/mod-queues` | `queue`, `limit`, `offset` | 平台 staff 视角 |
| `GET /api/v1/communities/:slug/mod-queues` | `queue`, `limit`, `offset` | 社区 owner/moderator 视角 |
| `GET /api/v1/communities/:slug/manage/banned-users` | `limit`, `offset` | 社区 owner/moderator 视角 |
| `GET /api/v1/communities/:slug/manage/muted-users` | `limit`, `offset` | 社区 owner/moderator 视角 |
| `GET /api/v1/communities/:slug/manage/approved-users` | `limit`, `offset` | 社区 owner/moderator 视角 |
| `GET /api/v1/communities/:slug/moderation/users/:user_id/notes` | `limit`, `offset` | 社区 owner/moderator 视角 |
| `GET /api/v1/communities/:slug/moderation/logs` | `action`, `actor_id`, `target_type`, `target_id`, `limit`, `offset` | 社区 owner/moderator 视角 |
| `GET /api/v1/search` | `q`, `scope`, `limit`, `offset` | `scope=all|communities|posts|users`；`all` 当前按 communities/posts/users 三个分区各返回最多 `limit` 条；PostgreSQL full-text search 结合字段权重、精确/前缀/子串命中和轻量时间衰减排序，帖子字段权重为标题 > 社区名/slug > 正文 |
| `GET /api/v1/notifications` | `category`, `status`, `limit`, `offset` | `category=all|interactions|replies|mentions|likes|system`；不传 `status` 默认 `all`，可显式传 `status=all|unread|read` |
| `GET /api/v1/messages/conversations` | `box`, `limit`, `offset` | `box=all|friends|requests|archived`，默认 `all`；列表返回会话摘要、参与者、最后消息、未读数、请求状态、拉黑和发送能力 |
| `GET /api/v1/messages/conversations/:id/messages` | `before_message_id`, `limit` | 私信消息倒序分页；`before_message_id` 用于读取更早消息，默认 30，最大 50 |
| `GET /api/v1/realtime/messages` | `ticket` | 使用 `POST /api/v1/realtime/tickets` 返回的短期 ticket 建立 WebSocket；ticket 只消费一次 |

所有使用 `limit/offset` 的列表响应都返回 `limit`、`offset`、`next_offset` 和 `has_more`。后端用 `limit + 1` 读取来判断是否还有下一页，响应数组最多返回 `limit` 条；当前页未满时 `has_more=false`，`next_offset=offset+当前返回条数`。`GET /api/v1/search?scope=all` 仍按 communities/posts/users 三个分区分别应用同一组 `limit/offset`，其 `has_more` 表示任一分区还有下一页。

评论、回复、帖子点赞、评论点赞和正文 `@username` 提及会写入站内通知；`category=interactions` 严格返回回复、提及和点赞类用户互动通知，不混入 `system`。点赞通知按收件人、通知类型、目标内容和小时窗口聚合未读计数，并在通知响应中返回 `aggregate_count`、`last_actor_id`、`actor` 和 `last_actor` 摘要。帖子/评论来源通知会尽量返回 `context.post_id`、`context.comment_id`、`context.permalink`、`context.post_title`、`context.comment_excerpt`、`context.comment_depth` 和 `context.community`，前端可直接跳转评论锚点。提及通知在帖子 / 评论发布时生成，编辑时只为新增提及生成，`source_type` 为 `post` 或 `comment`。

社区管理设置和规则接口均需要 Bearer。设置读取允许 owner/moderator 进入管理上下文，设置更新允许社区 owner 或平台 owner 覆盖权限修改 `name`、`description`、`avatar_url` 和 `banner_url`；四个字段均为可选字段但请求至少提供一项，媒体 URL 为空字符串表示清空，非空值必须是绝对 `http/https` URL 且最多 2048 字节。规则列表和 CRUD 允许 owner/moderator 使用，规则按 `position ASC, created_at ASC, id ASC` 返回。

平台 `platform_role=owner` 对所有 active 社区拥有管理覆盖权限：可以读取社区管理上下文、帖子 / 评论 / 举报管理列表、成员列表、设置、规则和 Reddit Mod Tools 社区级接口，并可执行设置更新、规则 CRUD、版主任免、内容审核动作、移除原因、保存回复、社区用户治理和 mod notes。覆盖权限不写入 `community_memberships`，`viewer_role` 继续表示真实社区成员身份；响应通过 `viewer_permissions.platform_owner_override=true` 表明覆盖来源。社区 owner transfer 仍要求真实社区 owner，平台 owner 需要改变社区 owner 时使用平台管理的 `POST /api/v1/admin/communities/:id/owner`。

平台管理接口均需要 Bearer，且 usecase 会从数据库确认当前用户是 active 平台 staff。`/api/v1/admin/*` 写操作会写入 `admin_audit_logs`，记录 actor、action、target、before 和 after；离线 owner bootstrap/recovery 使用固定 `actor_ref` 写入系统审计。平台运行开关目前包括 `registration_enabled`、`posting_enabled` 和 `upload_enabled`：关闭后分别阻止注册、发帖/发评论和图片上传；缺失设置行时运行时读取默认按 enabled 处理，避免 migration 未就绪时误关站。`PATCH /api/v1/admin/users/:id` 只能由 owner/admin 按角色边界更新用户状态或 legacy staff 标记：owner 不能通过该接口禁用、恢复或删除 active owner；admin 只能处理无平台角色的普通账号；staff 不能写。`PATCH /api/v1/admin/users/:id/platform-role` 只能由 owner 更新非 owner 用户为 `admin|staff|null`，请求 `role=owner` 或目标当前是 owner 都返回 `forbidden`，owner 创建、交接和恢复必须走 owner-transfer/bootstrap/recovery。管理员手工调分使用 `POST /api/v1/admin/users/:id/points/adjust`，请求体为 `delta` 和 `reason`；`delta` 不能为 0，扣减后余额不能小于 0，成功后写入 `point_transactions`，`source_type=admin_adjustment`，并同步写入平台管理审计日志。用户账号处罚使用 `POST /api/v1/admin/users/:id/sanctions` 创建，当前支持 `type=account_ban` 和固定 `duration=1d|3d|7d|30d|permanent`；非永久处罚的 `expires_at` 由后端计算，永久处罚 `expires_at=null`。active 且未过期的账号封禁会阻止登录和受保护接口；过期封禁按读取时语义返回 `expired`，提前解除使用 `POST /api/v1/admin/user-sanctions/:sanction_id/revoke` 写入 `revoked_by` 和 `revoked_at`，不删除历史记录。创建和解除处罚都写平台管理审计日志；owner 可处罚 owner 以外的用户，admin 只能处罚无平台角色用户，staff 只能查看处罚记录。平台 staff 可通过 `/api/v1/admin/titles` 管理头衔目录，并通过 `/api/v1/admin/users/:id/titles` 给用户授予或撤销头衔；头衔名称最多 20 字符，不能包含官方、管理员、认证、平台、系统、版主、owner、admin、official、verified 等保留词。

站点负责人交接使用 `GET/POST/DELETE /api/v1/admin/owner-transfer` 和 `GET/POST /api/v1/owner-transfer/:transfer_id[/accept]`。只有当前 active owner 能发起交接，发起请求包含 `target_user_id`、可选 `previous_owner_role=admin|null`、`reason` 和 `current_password`；目标用户必须 active 且不能是当前 owner，过期时间由后端固定生成为 48 小时。同一时间最多一个 pending transfer。目标用户接受时提交自己的 `current_password`，后端事务内锁定 transfer 和相关用户，把目标账号设为唯一 active `platform_role=owner`，把原 owner 改为发起时记录的 `previous_owner_role`，并刷新原 owner 的 `tokens_revoked_after`。首个 owner 和被盗号恢复只允许部署侧命令：`go run ./cmd/admin bootstrap-owner --user-id <uuid> --reason <text> --confirm` 和 `go run ./cmd/admin recover-owner --new-owner-user-id <uuid> --compromised-user-id <uuid> --reason <text> --revoke-sessions --disable-compromised --confirm`，不提供网页接管路由。

全站等级只按用户全站经验 `xp_total` 计算，不做社区等级；等级范围固定 `Lv.1-Lv.30`，后端按固定曲线返回 `level`、`level_name`、`current_level_xp`、`next_level_xp` 和 `level_progress`，前端不自行推导。经验不可消费，积分消费不影响等级。当前自动经验来源包括：每日首次登录 +5（每日最多 5）、发帖发布成功 +20（每日最多 100）、评论发布成功 +5（每日最多 80）、帖子被 upvote +3（每日最多 150）、评论被 upvote +2（每日最多 150）、帖子被收藏 +8（每日最多 120）。经验事件通过 `user_id + source_type + source_id` 去重；同一用户同一天重复登录不会重复获得登录经验；取消点赞、取消收藏、内容删除或审核移除不会撤销已产生的历史经验。

`PATCH /api/v1/me/profile` 需要 Bearer，允许当前用户更新公开资料字段 `display_name`、`avatar_url`、`banner_url`、`headline` 和 `bio`。五个字段都支持显式传空字符串清空；请求至少要包含一个字段；`display_name` 最多 40 字，`headline` 最多 80 字，`bio` 最多 300 字，`avatar_url` 和 `banner_url` 非空时必须是 `http/https` 绝对 URL。成功响应返回当前用户更新后的公开资料和实时公开计数。

用户关注关系使用 `user_follows` 事实表，`POST /api/v1/users/:username/follow` 和 `DELETE /api/v1/users/:username/follow` 均需要 Bearer；关注和取消关注都是幂等操作，不能关注自己，目标用户必须是 active。`GET /api/v1/me/followed-users` 返回当前用户关注的 active 用户列表和 `limit/offset/next_offset/has_more`。公开用户响应中的 `stats` 包含 `follower_count` 和 `following_count`，顶层 `viewer_is_following` 表示当前 Bearer 用户是否关注该公开用户；匿名读取时为 `false`，无效 Bearer 仍返回 `unauthenticated`。

私信系统独立于通知 Bell，所有 `/api/v1/messages/*` 写读接口都需要 Bearer。互关用户通过 `POST /api/v1/messages/conversations` 直接创建 `accepted` 会话并发送首条消息；非互关用户创建 `pending` 陌生人请求，pending 期间发起人不能连续追发，同一用户 24 小时内发起过多请求返回 `rate_limited`。收件人通过 `/messages/requests/:id/accept|reject` 处理请求；接受后会话可正常发送，拒绝后不可继续发送。`POST/DELETE /api/v1/users/:username/block` 维护双向发送禁用边界：任一方拉黑后历史会话保留但 `can_send=false`。消息类型支持 `text`、`image`、`share_post`、`share_comment`、`share_user` 和 `share_community`；分享消息持久化 `share_type/share_id/title/summary/thumbnail_url/target_url/snapshot_created_at`，上游内容不可见时前端按快照降级展示。会话列表支持 `box=all|friends|requests|archived`，响应不暴露用户可见“已读”，只返回未读数；打开会话后调用 `/read` 更新 read cursor。`DELETE /api/v1/messages/:id` 是当前用户本地隐藏，`POST /api/v1/messages/:id/recall` 仅允许发送者撤回。举报私信写入 `message_reports`，只保存被举报消息、参与者和前后有限上下文。`GET/PATCH /api/v1/me/privacy/messages` 管理 `allow_messages=everyone|mutuals|none` 和 `online_status_enabled`；在线状态默认关闭且只对 accepted 会话互关侧返回可见标记。实时通道先通过 Bearer 创建短期 ticket，再用 `GET /api/v1/realtime/messages?ticket=...` 建立 WebSocket；服务端发送持久化 `message_realtime_events` 的补发事件，HTTP 列表和详情仍是权威数据源。

账号安全接口支持校园邮箱验证码注册、邮箱验证码登录、找回密码、修改邮箱、修改密码和退出所有会话。邮箱会 trim 后小写存储和比较，只允许 `AUTH_EMAIL_ALLOWED_DOMAINS` 配置的域名；验证码只通过邮件或本地日志发送，不会出现在 HTTP 响应里。同一邮箱同一用途受重发间隔、24 小时发送次数和验证码错误次数限制，同一 IP 受小时级验证码发送次数限制；密码登录失败会写入 `auth_security_events`，并按账号标识和 IP 做 15 分钟滚动窗口限流。`POST /api/v1/auth/login` 兼容旧 `{ "username": "...", "password": "..." }`，也支持新 `{ "identifier": "...", "password": "..." }`，其中 `identifier` 可为用户名或邮箱。`POST /api/v1/auth/logout-all` 和密码变更会写入 `tokens_revoked_after`，旧 access token 后续访问会返回 `unauthenticated`。

`DELETE /api/v1/me/account` 需要当前登录态、`confirmation=DELETE`，并且当前密码或当前已验证邮箱收到的 `delete_account` 验证码任一通过。采用软删除：`users.status` 改为 `deleted`，`deleted_at` 和 `tokens_revoked_after` 写入当前时间，公开资料字段清空，平台 staff 标记清除；历史帖子、评论、举报、审核日志和安全事件保留原作者/actor 引用。删除后旧 token 立即返回 `unauthenticated`，公开用户主页和用户搜索不再返回该用户。注销时原 `username` 改写为内部墓碑名，原 `email` 置空，因此原用户名和邮箱会释放，可用于重新注册。

`GET /api/v1/search` 支持匿名读取公开可见结果，也支持可选 Bearer 的认证边界。`scope=users` 只返回 active 用户的公开资料摘要，不搜索也不返回 email、密码、权限或账号安全字段；`scope=all` 同时返回 `communities`、`posts` 和 `users` 三个数组，每个数组独立应用 `limit/offset`。搜索会优先按精确命中、前缀命中、子串命中和字段权重排序，并用 PostgreSQL full-text search 补充英文分词相关度；中文连续文本、slug、username 和短词会通过字面量子串匹配兜底。首版不返回 `highlights` 或 `match_reason`，前端可基于响应字段自行高亮。公开用户响应包含 `progression` 对象，字段为 `level`、`level_name`、`xp_total`、`current_level_xp`、`next_level_xp`、`level_progress`、`active_title` 和 `titles_count`；`active_title` 为用户选择且仍有效的头衔授予摘要。公开用户响应还包含 `stats.follower_count`、`stats.following_count` 和 `viewer_is_following`。

评论响应包含 `effects` 数组，列表、树和用户评论列表都会返回历史评论效果摘要。摘要字段包括 `id`、`effect_id`、`name`、`asset_url`、`animation_key`、`applied_by_user`、`points_spent` 和 `created_at`；效果被管理员停用后，历史记录仍会展示，但 `GET /api/v1/effects/catalog` 不再返回该效果用于新购买。`GET /api/v1/me/point-transactions` 返回当前用户积分流水，`GET /api/v1/admin/point-transactions` 返回平台 staff 视角积分流水；字段均为 `id`、`user_id`、`delta`、`balance_after`、`reason`、`source_type`、`source_id` 和 `created_at`，同样带 `limit/offset/next_offset/has_more`。

`POST /api/v1/embeds/resolve` 需要 Bearer，接受单个公开 URL 或包含 URL 的分享文本。嵌入解析只支持白名单 provider：Bilibili 视频、抖音视频、网易云音乐单曲 / 歌单 / 专辑、QQ 音乐单曲；`b23.tv` 和 `v.douyin.com` 会在白名单重定向边界内展开，禁止内网、localhost、用户信息 URL 和非 `http/https` scheme。抖音支持 canonical `/video/:id`、`/user/...?...modal_id=:id`、`open.douyin.com/player/video?vid=:id` 以及常见 `aweme_id/video_id/item_id` 参数。成功响应会 upsert `embeds` 记录并返回 `embed.id`、`provider_ref`、`canonical_url`、受控 `embed_url`、`iframe_allowed`、标题、描述、封面、作者和 `status=ready|unavailable`；第三方元数据抓取失败不阻断解析成功。

发帖、编辑帖子、发评论和编辑评论均支持可选 `content_refs` 请求字段，元素结构为 `{ "kind": "...", "ref_id": "..." }`，允许 `kind=image|link_preview|embed`，最多 50 条，并按请求顺序持久化和返回。`image` 引用必须指向同一次请求中已绑定或内容当前已绑定的图片附件 ID；`embed` 的 `ref_id` 必须使用 `/api/v1/embeds/resolve` 返回的 `embed.id`；`link_preview` 的 `ref_id` 是前端通过解析接口或客户端生成的稳定引用字符串。编辑请求省略 `content_refs` 时保留原引用，显式传空数组时清空原引用。

图片上传响应中的 `thumbnail_url` 是前端列表页缩略图读取字段。大于 512px 边长的 PNG/JPEG 上传会同步生成最大边 512px、质量 82 的独立 JPEG 缩略图并写入 `thumbnail_object_key`；小图、WebP 或缩略图生成 / 上传失败时，`thumbnail_url` 回退为原图 `url`。该回退不影响原图上传成功。

社区范围内容移除接口 `POST /api/v1/communities/:slug/moderation/posts/:id/remove` 和 `POST /api/v1/communities/:slug/moderation/comments/:id/remove` 需要 Bearer，且当前用户必须是目标社区 active owner 或 moderator。目标帖子 / 评论必须属于路径中的社区，否则返回 `not_found`，避免跨社区操作泄露资源存在性。两个接口请求体复用 `{ "reason": "..." }`，成功后复用审核动作响应并写入内容移除动作记录；账号处罚仍只能通过平台 admin sanctions 接口执行，社区版主不能封禁账号。

## 文本长度边界

所有外部请求写入或展示持久化的文本字段都由业务层校验，超过限制统一返回 `invalid_argument`；数据库 `CHECK` 约束作为兜底。当前上限如下：

| 字段 | 上限 |
|---|---:|
| `username` | 32 字符 |
| `password` | 256 bytes |
| `email` / `identifier` | 254 字符 |
| `code` | 12 字符 |
| `display_name` | 40 字符 |
| `headline` | 80 字符 |
| `bio` | 300 字符 |
| `avatar_url` / `banner_url` / 图片公开 URL / 嵌入 URL | 2048 bytes |
| 社区 `name` | 60 字符 |
| 社区 `description` | 300 字符 |
| 社区规则 `title` | 80 字符 |
| 社区规则 `body` | 500 字符 |
| 社区申请 `reason` / `reject_reason` | 500 字符 |
| 帖子 `title` | 120 字符 |
| 帖子 `body` | 20000 字符 |
| 评论 `body` | 5000 字符 |
| 举报 / 审核 `reason` | 500 字符 |
| 管理员积分调整 `reason` | 500 字符 |
| 用户处罚 `reason` | 500 字符 |
| 站点负责人交接 / 恢复 `reason` | 500 字符 |
| 头衔 `name` | 20 字符 |
| 头衔 `description` | 120 字符 |
| 头衔授予 `reason` | 500 字符 |
| 图片 `alt_text` | 200 字符 |
| `content_refs[].ref_id` | 2048 字符 |
| 搜索 `q` | 100 字符 |

## 错误边界

错误码和 HTTP 状态码映射以 `docs/contracts/http-error-handling.md` 为准。

常见边界：

- 需要认证的接口缺失 Bearer token：`unauthenticated`。
- 所有携带格式错误、过期或签名错误 Bearer token 的请求：`unauthenticated`。
- 无效 UUID、非法分页参数、非法枚举值或请求体格式错误：`invalid_argument`。
- 非作者编辑/删除内容、非平台 staff 审核操作：`forbidden`。
- 不存在或不可见的资源：`not_found`。
- slug、用户名、待审批申请等唯一约束冲突：`conflict`。

## 不在本快照内

- 不在本文中定义 request/response JSON schema；字段快照见 `docs/contracts/http-api-schema.md`。
- 不定义前端路由、页面或组件。
- 不新增 OpenAPI 生成流程。
- 不改变任何业务接口、错误码或响应格式。

## Stage 49 Community Member Governance Addendum

| Method | Path | Auth | Description |
|---|---|---|---|
| POST | /api/v1/communities/:slug/manage/moderators | Bearer | Community owner or platform owner override appoints a moderator |
| DELETE | /api/v1/communities/:slug/manage/moderators/:user_id | Bearer | Community owner or platform owner override removes a moderator |
| GET | /api/v1/communities/:slug/manage/owner-transfer | Bearer | Community owner or platform owner override reads current owner transfer |
| POST | /api/v1/communities/:slug/manage/owner-transfer | Bearer | Community owner creates an owner transfer |
| GET | /api/v1/communities/:slug/owner-transfer/:transfer_id | Bearer | Authenticated user reads owner transfer accept-page details |
| POST | /api/v1/communities/:slug/manage/owner-transfer/:transfer_id/accept | Bearer | Target user accepts an owner transfer |
| DELETE | /api/v1/communities/:slug/manage/owner-transfer/:transfer_id | Bearer | Initiator or platform owner override cancels a pending owner transfer |
| POST | /api/v1/admin/communities/:id/owner | Bearer | Platform staff transfers community owner |

Community member governance requires Bearer auth. `GET /api/v1/communities/:slug/manage/members` is readable by owner/moderator or active platform `owner` override. `PATCH /api/v1/communities/:slug/manage/settings` accepts any subset of `name`, `description`, `avatar_url` and `banner_url`; at least one field is required, empty media URL clears it, and non-empty media URLs must be absolute `http` or `https` URLs no longer than 2048 bytes. `POST /api/v1/communities/:slug/manage/moderators` accepts `{ "username": "alice" }`, requires the real community owner or platform owner override, can appoint an active user as an active moderator, and cannot appoint the current owner as moderator. `DELETE /api/v1/communities/:slug/manage/moderators/:user_id` requires the real community owner or platform owner override, demotes an active moderator back to member, is idempotent for an existing member, and cannot remove the owner. Moderator caps are based on active member count: under 500 members allows 5 moderators, 500 or more allows 10, and 2000 or more allows 20. `POST /api/v1/communities/:slug/manage/owner-transfer` accepts `{ "username": "alice" }`, still requires the real community owner, creates a pending transfer to an active target user, expires after 48 hours, and creates a system notification with `source_type=community_owner_transfer` and `source_id=<community_slug>:<transfer_id>`. `GET /api/v1/me/community-owner-transfers` lists transfers where the current user is `to_user_id`, includes each transfer's `community`, supports `status=pending|accepted|cancelled|expired|all`, and returns `limit/offset/next_offset/has_more`. `GET /api/v1/communities/:slug/manage/owner-transfer` returns the current transfer or `null`; `GET /api/v1/communities/:slug/owner-transfer/:transfer_id` returns accept-page details including usernames, display names, status, `expires_at`, `accepted_at`, `cancelled_at`, `viewer_is_target` and `viewer_can_cancel`; the target accepts through `POST /api/v1/communities/:slug/manage/owner-transfer/:transfer_id/accept`; acceptance promotes the target to owner and demotes the previous owner to member. `DELETE /api/v1/communities/:slug/manage/owner-transfer/:transfer_id` cancels a pending transfer only for the initiator or platform owner override. `POST /api/v1/admin/communities/:id/owner` accepts `{ "user_id": "...", "reason": "..." }`, requires the existing platform staff boundary, transfers the community owner, supports recovery when the community has no active owner by upserting the target active user as owner, and writes an admin audit log including the optional reason; in the no-active-owner case the audit `before` state is `{ "owner": null }`. Platform owner changes are no longer allowed through `PATCH /api/v1/admin/users/:id/platform-role`; that route can only set non-owner users to `admin|staff|null`, and owner creation/transfer/recovery must use the platform owner transfer or deployment CLI flow.

## Stage 51 Moderation Sanctions Addendum

Stage 51 closes the P1 moderation sanctions and community-scoped removal gap. The main route table now includes platform user sanctions, sanction revocation, and community-scoped post/comment removal. Account bans are durable `user_sanctions` records with explicit active/revoked/expired read semantics; temporary bans expire automatically by `expires_at`, and revocation preserves audit state. Community owner/moderator removal is scoped to the path community and does not grant account-level punishment authority.
