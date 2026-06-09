# HTTP API 契约快照

本文记录当前 HTTP API 路由、认证边界和全局错误响应约定。它是当前路由契约快照，不替代 handler/usecase 测试，也不是完整 OpenAPI schema。当前 request/response schema 快照见 `docs/contracts/http-api-schema.md`。

路由清单、Auth 边界和查询参数清单需要通过以下脚本与源码中的 `RegisterRoutes`、`cmd/api/main.go` 路由分组、handler query 读取保持同步：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-api-contract-doc.ps1
```

## 全局约定

- `/healthz` 不需要认证，只表示进程存活。
- `/api/v1/auth/register` 和 `/api/v1/auth/login` 不需要认证。
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
| POST | /api/v1/auth/login | public | 登录并签发 access token |
| GET | /api/v1/me | Bearer | 当前用户 |
| GET | /api/v1/me/saved-posts | Bearer | 当前用户保存的公开帖子 |
| GET | /api/v1/me/followed-communities | Bearer | 当前用户关注的公开社区 |
| GET | /api/v1/me/points | Bearer | 当前用户积分账户 |
| GET | /api/v1/users/:username | optional Bearer | 公开用户主页 |
| GET | /api/v1/communities | optional Bearer | 社区列表 |
| GET | /api/v1/communities/:slug | optional Bearer | 社区详情 |
| POST | /api/v1/communities/:slug/follow | Bearer | 关注社区 |
| DELETE | /api/v1/communities/:slug/follow | Bearer | 取消关注社区 |
| POST | /api/v1/community-applications | Bearer | 提交社区申请 |
| GET | /api/v1/community-applications | Bearer | 平台 staff 查看社区申请列表 |
| GET | /api/v1/community-applications/:id | Bearer | 平台 staff 查看社区申请详情 |
| POST | /api/v1/community-applications/:id/approve | Bearer | 平台 staff 审批通过社区申请 |
| POST | /api/v1/community-applications/:id/reject | Bearer | 平台 staff 拒绝社区申请 |
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
| GET | /api/v1/moderation/reports | Bearer | 审核台举报列表 |
| GET | /api/v1/moderation/reports/:id | Bearer | 审核台举报详情 |
| POST | /api/v1/moderation/reports/:id/dismiss | Bearer | 驳回举报 |
| POST | /api/v1/moderation/reports/:id/remove-target | Bearer | 按举报移除目标内容 |
| GET | /api/v1/notifications/unread-summary | Bearer | 当前用户分类未读通知摘要 |
| GET | /api/v1/notifications | Bearer | 当前用户通知列表 |
| POST | /api/v1/notifications/:id/read | Bearer | 标记单条通知已读 |
| POST | /api/v1/notifications/read-all | Bearer | 标记当前用户全部通知已读 |
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
| `GET /api/v1/communities/:slug/posts` | `sort`, `t`, `limit`, `offset` | `sort=best|hot|new|top|rising`，`t=hour|day|week|month|year|all`，分页默认由 usecase 收口 |
| `GET /api/v1/posts` | `source`, `sort`, `t`, `limit`, `offset` | `source=all|recommended`；`recommended` 当前为公开可解释排序流，默认 `sort=best`；`sort=best|hot|new|top|rising`；`t=hour|day|week|month|year|all` |
| `GET /api/v1/posts/:id/comments` | `view`, `sort`, `limit`, `offset`, `max_depth` | `view=flat|tree`，`sort=best|top|new|old|controversial`，tree view 返回前序遍历扁平数组 |
| `GET /api/v1/users/:username/posts` | `sort`, `limit`, `offset` | `sort=best|hot|new|top|rising`，只返回该用户在公开可读社区中的 visible 帖子 |
| `GET /api/v1/users/:username/comments` | `limit`, `offset` | 只返回该用户在公开可读社区 visible 帖子下的 visible 评论 |
| `GET /api/v1/moderation/reports` | `status`, `limit`, `offset` | 平台 staff 视角 |
| `GET /api/v1/search` | `q`, `scope`, `limit`, `offset` | `scope=all|communities|posts` |
| `GET /api/v1/notifications` | `category`, `status`, `limit`, `offset` | `category=all|replies|mentions|likes|system`，`status=all|unread|read` |

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
