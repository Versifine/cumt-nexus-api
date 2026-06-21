# CUMT Nexus API 当前阶段工单

## 当前阶段

- 阶段：`Stage 63 Backend API Gap Closure`
- 状态：`DONE`
- 分支：`main`
- 目标：关闭前端 `backend-api-needs.md` 中 2.0Q 内容作者身份标识字段缺口。

阶段退出标准：

- 帖子列表、帖子详情、用户帖子、保存帖子和帖子效果摘要的用户摘要返回 `community_role`、`platform_role` 和 `is_platform_staff`。
- 评论列表、评论树、评论发布 / 更新响应、用户评论和评论效果摘要的用户摘要返回同样字段。
- 公开用户响应返回 `platform_role` 与 `is_platform_staff`。
- `community_role` 基于内容所属社区内的作者 / 互动用户 active membership，不使用当前查看者的 `viewer_role`。
- 合同文档、前端缺口账本、基线验证和 Docker compose 重建通过。

### T63-001：内容作者身份标识字段

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`Stage 62 已完成`
- 当前结论：
  - `postusecase.UserSummary` 增加 `community_role`、`platform_role` 和 `is_platform_staff`，帖子作者和帖子效果 `applied_by_user` 共用该摘要。
  - 评论作者和评论效果 `applied_by_user` 复用同一身份摘要字段。
  - 帖子 / 评论仓储按内容所属社区查询 active `community_memberships.role`，平台身份来自 `users.platform_role` 和 `users.is_platform_staff`。
  - 公开用户与资料更新响应返回 `platform_role` 和 `is_platform_staff`。

## 上一阶段

- 阶段：`Stage 62 Backend API Gap Closure`
- 状态：`DONE`
- 分支：`main`
- 目标：关闭前端 `backend-api-needs.md` 中 2.0P 幽默 / 笑了内容互动目录缺口。

阶段退出标准：

- `GET /api/v1/effects/catalog` 返回 active `humor` 和 `laughed`。
- `humor` 字段为 `name=幽默`、`emoji=🎭`、`cost_points=5`。
- `laughed` 字段为 `name=笑了`、`emoji=😆`、`cost_points=5`。
- `POST /api/v1/posts/:id/effects` 和 `POST /api/v1/comments/:id/effects` 可使用 `effect_id=humor|laughed`，扣分和流水沿用现有内容互动。
- 合同文档、migration 清单、前端缺口账本、基线验证和 Docker compose 重建通过。

### T62-001：幽默 / 笑了内容互动目录

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`Stage 61 已完成`
- 当前结论：
  - 新增 `migrations/000042_add_humor_and_laughed_effects.*.sql`，upsert active `humor` 和 `laughed` 到 `effects` 目录。
  - `humor` 使用 `name=幽默`、`emoji=🎭`、`cost_points=5`、空 `description` 和 `animation_key=humor`。
  - `laughed` 使用 `name=笑了`、`emoji=😆`、`cost_points=5`、空 `description` 和 `animation_key=laughed`。
  - 下行迁移只停用 `humor` 和 `laughed`，不删除行，避免已有历史帖子 / 评论互动被外键阻断。
  - 帖子 / 评论购买与历史摘要复用 2.0N 已有通用内容互动链路。
  - 帖子 `hot`、`rising` 和 `recommended` 排序已把 `post_effects.points_spent` 作为付费内容互动信号纳入计算，`best` / `top` 仍保持投票语义为主并用效果积分做同分兜底。

## 上一阶段

- 阶段：`Stage 61 Backend API Gap Closure`
- 状态：`DONE`
- 分支：`main`
- 目标：关闭前端 `backend-api-needs.md` 中 2.0O FakeNews 内容互动目录缺口。

阶段退出标准：

- `GET /api/v1/effects/catalog` 返回 active `fake_news`。
- `fake_news` 字段为 `name=FakeNews`、`emoji=📰`、`cost_points=8`。
- `POST /api/v1/posts/:id/effects` 和 `POST /api/v1/comments/:id/effects` 可使用 `effect_id=fake_news`，扣分和流水沿用现有内容互动。
- 合同文档、migration 清单、前端缺口账本、基线验证和 Docker compose 重建通过。

### T61-001：FakeNews 内容互动目录

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`Stage 60 已完成`
- 当前结论：
  - 新增 `migrations/000041_add_fake_news_effect.*.sql`，upsert active `fake_news` 到 `effects` 目录。
  - `fake_news` 使用 `name=FakeNews`、`emoji=📰`、`cost_points=8`、空 `description` 和 `animation_key=fake_news`。
  - 下行迁移只停用 `fake_news`，不删除行，避免已有历史帖子 / 评论互动被外键阻断。
  - 帖子 / 评论购买与历史摘要复用 2.0N 已有通用内容互动链路。

## 上一阶段

- 阶段：`Stage 60 Backend API Gap Closure`
- 状态：`DONE`
- 分支：`main`
- 目标：关闭前端 `backend-api-needs.md` 中 2.0N 帖子/评论中文积分互动后端缺口。

阶段退出标准：

- `GET /api/v1/effects/catalog` 返回中文内容互动目录，并为每项返回 `emoji`。
- 新增 `POST /api/v1/posts/:id/effects`，成功后扣减积分并返回更新后的 `points`。
- 帖子读取响应新增 `effects[]`，覆盖全站列表、详情、社区列表、用户帖子和保存帖子列表。
- 评论读取响应的 `effects[]` 与帖子统一摘要字段，新增 `emoji`。
- 旧英文装饰型默认目录停用，新中文目录通过 migration 落地；历史互动不按目录 active 状态过滤。
- 帖子互动消费流水使用 `reason=post_effect`、`source_type=post_effect`、`source_id=<post_effect_id>`。
- `go test ./internal/effect/... ./internal/comment/... ./internal/post/...`、`go test ./...`、合同基线和 Docker compose 重建通过。

### T60-001：帖子/评论中文内容互动

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`Stage 59 已完成`
- 当前结论：
  - 新增 `migrations/000040_create_post_effects_and_emoji_catalog.*.sql`：给 `effects` 增加 `emoji`，停用 `sparkle/spotlight/campus_star/neon_ring`，种入 `useful/cant_hold/classic/following_up/verified_true/abstract/godlike/clown`，并创建 `post_effects`。
  - `effectusecase.ApplyPostEffect` 校验登录、帖子可见、目录 active 后，由仓储事务创建帖子互动、扣减积分并写 `post_effect` 流水。
  - `POST /api/v1/posts/:id/effects` 与评论互动使用同样请求体 `effect_id`，成功响应返回 `post_effect` 和更新后的 `points`。
  - 帖子和评论历史互动摘要统一包含 `id/effect_id/name/emoji/asset_url/animation_key/applied_by_user/points_spent/created_at`。
  - 历史互动读取联表 `effects` 但不按 `effects.is_active` 过滤，目录停用只影响新购买。

## 上一阶段

- 阶段：`Stage 59 Backend API Gap Closure`
- 状态：`DONE`
- 分支：`main`
- 目标：关闭前端 `backend-api-needs.md` 中 2.0M 自动积分获得落账后端缺口。

阶段退出标准：

- 自动积分获得由后端写入 `point_transactions` 并更新 `user_points.balance/lifetime_earned`。
- 新增 `point_reward_claims`，自动奖励按 `user_id + source_type + source_id` 幂等。
- 接入每日登录 / 积分账户读取、发帖、评论、帖子 / 评论 upvote、帖子收藏和举报采纳。
- 自动积分奖励按来源每日封顶；积分副作用失败不影响已成功主动作。
- 取消点赞、取消收藏、删除和审核移除不回滚历史积分流水。
- `go test ./internal/effect/...`、相关 usecase 测试、`go test ./...`、合同基线和 Docker compose 重建通过。

### T59-001：自动积分获得落账

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`Stage 58 已完成`
- 当前结论：
  - 新增 `migrations/000039_create_point_reward_claims.*.sql`，用于自动积分奖励幂等 claim。
  - `effectusecase.GrantPoints` 统一执行自动积分策略，当前策略包括每日登录 / 积分账户读取 +5、发帖 +5、评论 +1、帖子 upvote +1、评论 upvote +1、帖子收藏 +3、举报采纳 +5。
  - `PostgresEffectRepository.GrantPoints` 在事务内创建 / 锁定积分账户、claim 去重、按天封顶、更新余额和插入积分流水。
  - 登录、发帖、评论、投票、收藏和举报采纳动作已接入积分 recorder；积分写入失败只作为副作用忽略，不影响主动作成功返回。
  - 同一用户对同一目标的一次 upvote / 收藏只给作者发放一次积分；同一 report 只给 reporter 发放一次采纳积分。

## 上一阶段

- 阶段：`Stage 58 Backend API Gap Closure`
- 状态：`DONE`
- 分支：`main`
- 目标：关闭前端 `backend-api-needs.md` 中 2.0L 私信撤回 2 分钟窗口后端兜底缺口。

### T58-001：私信撤回 2 分钟窗口后端兜底

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`Stage 57 已完成`
- 当前结论：
  - `POST /api/v1/messages/:id/recall` 只允许发送者在消息 `created_at + 2m` 以内撤回。
  - 超过窗口返回 `message_recall_expired`，HTTP 状态为 `409`，消息保持原状态且不发送 realtime 撤回事件。
  - 已撤回、当前用户本地删除或非 `visible` 消息返回状态冲突，不重复生成撤回事件。
  - 非发送者继续返回 `forbidden`。

## 更早阶段

- 阶段：`Stage 57 Backend API Gap Closure`
- 状态：`DONE`
- 分支：`main`
- 目标：关闭前端 `backend-api-needs.md` 中 2.0K 私信忽略后重开和 2.0J 社区公开读带 Bearer 500 两个后端缺口。

### T57-001：私信忽略后由收件人主动重开

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`Stage 56 已完成`
- 当前结论：
  - `messagehttp.conversationResponse` 新增 `viewer_can_reopen`。
  - 只有原请求收件人在 rejected/disabled 会话、双方未拉黑时得到 `viewer_can_reopen=true`。
  - 原请求发起人继续发送或重新发起返回 `message_request_rejected`，不再返回 `201` disabled 空成功。
  - 原请求收件人主动发送会在事务内把 request/conversation 更新为 `accepted`，插入本次 message，并发送 conversation realtime 事件。

### T57-002：社区公开读带有效 Bearer 不再 500

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`Stage 56 已完成`
- 当前结论：
  - `PostgresPlatformStaffRepository.IsPlatformOwner` 对 `users.platform_role=NULL` 明确返回 `false`，避免普通用户公开读社区时扫描 SQL NULL 到 bool 触发 internal。
  - `GET /api/v1/communities` 和 `GET /api/v1/communities/:slug` 保持 optional Bearer 合同：匿名返回匿名 viewer 字段，有效 Bearer 返回当前 viewer 的 follow/role/permissions 字段。

## 上一阶段

- 阶段：`Stage 56 Notification Interaction Context Contract`
- 状态：`DONE`
- 分支：`main`
- 目标：关闭前端缺口台账中抖音式消息中心、互动通知分类和评论通知上下文缺口。

阶段退出标准：

- `GET /api/v1/notifications?category=interactions|system&limit=20&offset=0` 可用。
- 不传 `status` 时通知列表默认返回 `all`，前端无需展示 read/unread 过滤也能读取完整消息流。
- `category=interactions` 仅包含回复、提及和点赞类用户互动通知，不混入系统通知。
- 通知响应包含 `actor`、`last_actor`、`aggregate_count` 和帖子/评论 `context`。
- 评论类通知上下文提供 `post_id/comment_id/permalink/post_title/community/comment_excerpt/comment_depth`。
- 私信/会话仍未开放正式入口；前端如展示只能保持 disabled/FUTURE。
- README、HTTP route/schema 合同、阶段记录和前端 `backend-api-needs.md` 同步当前合同。
- `go test ./...`、`go build -buildvcs=false ./...`、API route/schema/config 合同校验和 Docker HTTP 冒烟通过。

## 当前推进位

### T46-001：管理端积分流水和手工调分

- 状态：`DONE`
- 优先级：`P2`
- 前置依赖：`阶段 45 已完成`
- 目标：让后台运营可以读取积分流水并通过正式合同手工调整用户积分。
- 当前结论：
  - 新增 `GET /api/v1/admin/point-transactions?user_id=<uuid>&limit=20&offset=0`，平台 staff 可查看全站或指定用户积分流水。
  - 新增 `POST /api/v1/admin/users/:id/points/adjust`，请求体为 `delta` 和 `reason`。
  - `delta` 不能为 0；扣减后余额不能小于 0；不存在或已注销用户不能调分。
  - 成功调分写入 `point_transactions`，`source_type=admin_adjustment`、`source_id=<actor_id>`，并同步写入 `admin_audit_logs`。
  - 本切片不包含全站等级、经验事件或头衔授予。

### T47-001：全站等级、经验事件和头衔授予

- 状态：`DONE`
- 优先级：`P2`
- 前置依赖：`T46-001`
- 目标：让前端可以接入真实全站成长系统和头衔展示，不再需要伪造等级或头衔能力。
- 当前结论：
  - 新增 `migrations/000024_create_progression_titles.*.sql`：创建 `user_progressions`、`xp_event_claims`、`xp_events`、`titles` 和 `title_grants`。
  - 新增 `GET /api/v1/me/progression`、`GET /api/v1/me/xp-events?limit=20&offset=0`、`GET /api/v1/me/titles?limit=20&offset=0` 和 `PATCH /api/v1/me/title`。
  - 公开用户响应新增 `progression` 对象，包含等级、经验、进度、展示头衔和已获头衔数。
  - 等级固定为 `Lv.1-Lv.30`，只做全站等级，不做社区等级；后端返回固定曲线计算结果。
  - 自动经验来源包括每日首次登录、发帖、发评论、收到帖子/评论 upvote 和帖子被收藏；经验事件按 `user_id + source_type + source_id` 去重，并按来源每日封顶。
  - 新增平台 staff 头衔管理与授予接口：`GET/POST/PATCH /api/v1/admin/titles`、`GET/POST /api/v1/admin/users/:id/titles`、`DELETE /api/v1/admin/users/:id/titles/:grant_id`。
  - 头衔写操作与 `admin_audit_logs` 在同一事务内完成；用户只能选择自己未撤销、未过期且仍启用的头衔。

### T48-001：用户关注和公开用户社交计数

- 状态：`DONE`
- 优先级：`P2`
- 前置依赖：`T47-001`
- 目标：让前端可以接入真实用户关注关系、公开用户粉丝/关注计数和 viewer 关注状态。
- 当前结论：
  - 新增 `migrations/000025_create_user_follows.*.sql`，创建 `user_follows` 事实表、禁止自关注约束和 follower/following 查询索引。
  - 新增 `GET /api/v1/me/followed-users?limit=20&offset=0`，返回当前用户关注的 active 用户和统一分页元信息。
  - 新增 `POST /api/v1/users/:username/follow` 和 `DELETE /api/v1/users/:username/follow`，成功返回 204，关注和取消关注均幂等。
  - 公开用户响应新增 `stats.follower_count`、`stats.following_count` 和 `viewer_is_following`；匿名读取时 viewer 状态为 false，有效 Bearer 返回当前用户视角。
  - 不能关注自己；目标用户不存在、被禁用或已注销时按公开用户边界返回 `not_found`。

### T44-001：全站列表分页元信息 P1 补齐

- 状态：`DONE`
- 优先级：`P1`
- 前置依赖：`阶段 43 已完成`
- 目标：让社区索引、帖子流、评论、搜索、通知、审核和管理列表都能返回前端增量加载所需的分页元信息。
- 当前结论：
  - `GET /api/v1/communities` 新增 `limit/offset` query 参数，公开社区索引不再一次性返回全量。
  - 已为社区索引、关注社区、社区管理成员/帖子/评论/举报、社区申请、帖子列表、收藏列表、用户帖子、评论列表、用户评论、搜索、通知、审核举报、后台用户/社区/效果/审计列表补齐 `next_offset` 和 `has_more`。
  - usecase 统一用 `limit+1` 读取并截断响应，避免前端根据返回条数自行猜测是否还有下一页。
  - `scope=all` 搜索保持三分区返回，`has_more` 表示任一分区仍有下一页。
  - 本阶段不引入 cursor，不新增数据库 migration，不改排序和权限语义。
  - 已同步 README、HTTP route/schema 合同和前端缺口台账。

### T44-002：关注帖子流 source=following 补齐

- 状态：`DONE`
- 优先级：`P2`
- 前置依赖：`T44-001`
- 目标：让前端 `/following` 可以读取真实关注社区帖子流，不用普通公开帖子冒充关注流。
- 当前结论：
  - `GET /api/v1/posts?source=following` 已进入正式合同。
  - 未登录请求返回 `unauthenticated`，不会降级为公开帖子流。
  - 登录后只读取当前用户已关注、active public 社区内的 visible 帖子。
  - 排序、时间范围和 `limit/offset/next_offset/has_more` 复用全站帖子列表合同。
  - 已同步 README、HTTP route 合同和前端缺口台账。

### T45-001：评论效果读取与当前用户积分流水

- 状态：`DONE`
- 优先级：`P2`
- 前置依赖：`T44-002`
- 目标：让前端可以展示评论历史效果和当前用户积分流水，不只在购买成功瞬间拿到一次性返回。
- 当前结论：
  - 评论列表、评论树和用户评论列表的 `comment` 响应新增 `effects[]`。
  - 历史评论效果读取不按 `effects.is_active` 过滤；效果停用后仍可展示历史记录，但不可再次购买。
  - 新增 `GET /api/v1/me/point-transactions?limit=20&offset=0`，返回当前用户积分流水和统一分页元信息。
  - 本切片不包含管理员积分流水、管理员手工调分、全站等级和头衔授予。

### T43-001：注销账号 P0 补齐

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 42 已完成`
- 目标：让账号安全页危险区可以接入真实注销账号合同，而不是只展示禁用按钮。
- 当前结论：
  - 新增 `migrations/000023_add_account_deletion.*.sql`：`users.status` 允许 `deleted`，新增 `deleted_at`，`auth_email_codes.purpose` 允许 `delete_account`。
  - 新增 `POST /api/v1/me/security/email-codes/delete-account`，只允许给当前账号已验证邮箱发送注销确认验证码。
  - 新增 `DELETE /api/v1/me/account`，请求体为 `code?`、`current_password?`、`confirmation`；`confirmation` 必须为 `DELETE`，`code` 和 `current_password` 至少提供一个且任一通过即可。
  - 注销时校验当前密码或注销验证码，随后把用户软删除、清公开资料、清平台 staff 标记、撤销所有 token，并写入 `account_deleted` 安全事件。
  - 注销时将 username 改写为内部墓碑名、email 置空，释放原 username/email 重新注册；历史帖子、评论、举报、审核日志和安全审计继续保留原用户引用。
  - 已同步 README、HTTP route/schema/migration 合同和前端缺口台账。

### T42-001：账号安全 P0 补齐

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 41 已完成`
- 目标：把账号体系补到校园社区 P0 可验收水平，并让前端可以接入邮箱验证码注册、登录、找回密码和账号安全设置页。
- 当前结论：
  - 新增 `migrations/000022_add_auth_security.*.sql`：`users` 增加邮箱、验证状态、最近登录、密码更新时间和 token 失效点；新增 `auth_email_codes` 与 `auth_security_events`。
  - 新增 `internal/auth/authrepository`、`authcode` 和 `mail`，账号安全 usecase 统一编排验证码生成/哈希/发送、邮箱域名校验、频控、审计、邮箱注册、验证码登录、重置密码、改邮箱、改密码和退出所有会话。
  - `MAIL_PROVIDER=log` 本地只把验证码写入 API 日志；`MAIL_PROVIDER=smtp` 支持 `SMTP_*` 配置。配置合同新增 `AUTH_EMAIL_ALLOWED_DOMAINS`、验证码 TTL/重发/次数/IP/长度配置和 SMTP 配置。
  - `POST /api/v1/auth/login` 扩展为 username 或 email + password 登录，失败统一为 `unauthenticated`，并记录登录失败安全事件；旧 `username` 字段继续兼容。
  - 认证中间件在解析 JWT 后会查询数据库确认用户 active 且 token 未早于 `tokens_revoked_after`。
  - 已同步 README、`.env.example`、Docker compose、HTTP route/schema/migration/config 合同和前端缺口台账；P0 账号安全已从前端当前缺口移入已解决记录。
  - 已通过 `go test ./...`、`go build -buildvcs=false ./...`、API route/schema/migration/config 合同校验。
  - Docker prod 栈已重建镜像、执行 migration 到 `22` 并重启 API；真实 HTTP 冒烟覆盖邮箱注册、identifier 邮箱登录、`/me/security`、`logout-all` 旧 token 失效、验证码登录、密码重置、改邮箱和新邮箱登录。

### T41-001：公共搜索增强

- 状态：`DONE`
- 优先级：`P1`
- 前置依赖：`阶段 40 已完成`
- 目标：让公开搜索从基础 communities/posts full-text search 扩展为可供前端搜索页直接接入的三分区搜索合同。
- 当前结论：
  - `searchusecase` 新增 `ScopeUsers` 和 `SearchUsers` 仓储边界，`ScopeAll` 同时编排 communities、posts 和 users 三类结果。
  - `searchhttp.searchResponse` 新增 `users` 数组；用户结果包含 `id`、`username`、`display_name`、`avatar_url`、`headline`、`bio_excerpt`、`status`、`created_at` 和 `updated_at`。
  - `PostgresSearchRepository` 在现有 PostgreSQL full-text search 外增加字面量 `LIKE` 兜底，用于中文连续文本、短词、slug / username 前缀和子串命中。
  - 新增 `migrations/000021_add_search_like_escape_function.*.sql`，提供 `escape_like_query(text)`，避免用户搜索 `%`、`_` 和反斜杠时被当作通配符。
  - 当前不返回后端高亮片段或 `match_reason`，前端可基于响应字段自行高亮。
  - 已同步 README、HTTP route/schema/migration 合同、阶段文档和前端缺口台账；P1 搜索增强已从前端当前缺口移入已解决记录。
  - 已通过 `go test ./internal/search/...`、`go test ./...`、`go build -buildvcs=false ./...`、API 合同校验、API schema 校验和 migration 合同校验。
  - Docker prod 栈已重建镜像、执行 migration 到 `21` 并重启 API；`/healthz` 正常，`GET /api/v1/search?q=11111&scope=users` 可返回 `users` 分区。

### T40-001：白名单嵌入解析补全

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 39 已完成`
- 目标：让用户粘贴常见白名单媒体链接或分享文本后，后端能返回可播放、可持久化、可给前端引用的结构化 embed。
- 当前结论：
  - `POST /api/v1/embeds/resolve` 可从分享文本提取首个公开 URL，并展开 `v.douyin.com` 和 `b23.tv` 短链；重定向只允许落到对应 provider 白名单域名。
  - 抖音支持 `douyin.com/video/:id`、`douyin.com/user/...?...modal_id=:id`、`open.douyin.com/player/video?vid=:id` 和常见 `aweme_id/video_id/item_id` 参数，返回 `open.douyin.com/player/video` 受控 iframe URL。
  - Bilibili 支持 BV 和 av 视频；网易云音乐支持 song / playlist / album；QQ 音乐支持 songmid 和 songid，并统一返回前端可渲染的受控 `embed_url`。
  - 新增 `embeds` 表和 `PostgresContentRefRepository.UpsertEmbed`，按 `provider + provider_ref` 复用同一 `embed.id`，刷新元数据和 `status`。
  - `embed` 响应新增 `id`、`provider_ref`、`title`、`description`、`image_url`、`author_name` 和 `status`；第三方元数据抓取失败不阻断解析。
  - `content_refs.kind=embed` 的 `ref_id` 现在要求 UUID 形态的 `embed.id`，不再接受任意第三方 URL 当 embed 引用。
  - 本阶段不做正文自动扫描、已保存内容额外返回结构化 `embeds` 列表、第三方 API 深度抓取、复杂审核队列或任意 iframe 放开。

### T39-001：当前用户公开资料更新接口

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 38 已完成`
- 目标：补齐前端 `/settings/profile` 页面需要的真实保存合同，并把已暴露资料字段的读模型从占位升级为真实值。
- 当前结论：
  - 新增 `PATCH /api/v1/me/profile`，请求支持 `display_name`、`avatar_url`、`banner_url`、`headline` 和 `bio` 五个可选字段；显式传空字符串时清空对应资料。
  - 新增 `migrations/000017_add_user_profile_fields.*.sql`，把 `display_name`、`avatar_url`、`headline` 和 `bio` 落到 `users` 表，并加基础长度 / URL 约束。
  - `userdomain` 新增资料值对象和 `UpdateProfile` 不变量；`userusecase` 新增资料更新编排，handler 只负责 JSON 解析、当前用户读取和响应映射。
  - 公开用户主页、帖子作者摘要、评论作者摘要和社区成员摘要开始读取真实用户资料列；空 `display_name` 时继续回退 `username`，`badges/roles` 仍保持空数组占位。
  - 本阶段不做头像图片上传归属、勋章系统、复杂角色体系、资料审核流或 `/me` 扩展成完整资料读取接口。
  - 已同步 README、HTTP route/schema/migration 合同、内部架构文档、阶段文档和前端缺口台账。
  - 已通过 `go test ./internal/user/...`、受影响仓储包测试、全量 Go 测试、Go 构建、API 合同校验、API schema 校验和 migration 合同校验。

### T38-001：平台管理扩展接口

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 37 已完成`
- 目标：补齐前端后台管理页需要的最小真实后端合同，替代前端侧占位或手动 SQL 流程。
- 当前结论：
  - 新增 `internal/admin` 模块，按 handler -> usecase -> repository 分层实现平台管理读取、写入和审计。
  - 新增 `admin_settings` 表，种子化 `registration_enabled`、`posting_enabled` 和 `upload_enabled` 为 enabled。
  - 新增 `admin_audit_logs` 表；用户、社区、效果和设置写操作都在事务内写入 before / after 审计日志。
  - 新增 `GET/PATCH /api/v1/admin/users`、`GET/PATCH /api/v1/admin/communities`、`GET/PATCH /api/v1/admin/effects`、`GET/PATCH /api/v1/admin/settings` 和 `GET /api/v1/admin/audit-logs`。
  - 所有 admin 接口均需要 Bearer，并由 usecase 校验当前用户是 active 平台 staff。
  - `registration_enabled=false` 阻止注册，`posting_enabled=false` 阻止发帖和发评论，`upload_enabled=false` 阻止图片上传。
  - 运行时读取缺失设置行时默认 enabled，避免 migration 未就绪时误关站；表内 check 约束仍只允许已知开关。
  - 本阶段不做复杂后台 UI、角色权限矩阵、批量操作、管理端搜索、导出、报表或全局配置表泛化。
  - 已同步 README、HTTP route/schema/migration 合同、内部架构文档、阶段文档和前端缺口台账。
  - 已通过目标包测试、全量 Go 测试、Go 构建、API 合同校验、API schema 校验和 migration 合同校验。

### T37-001：真实缩略图生成策略

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 36 已完成`
- 目标：让图片上传不再只有 `thumbnail_url` 回退原图；对确实需要列表页省流量的大图生成独立缩略图对象。
- 当前结论：
  - 大于 512px 边长的 PNG/JPEG 上传会同步生成最大边 512px、质量 82 的 JPEG 缩略图。
  - 缩略图对象 key 为 `thumbnails/{yyyy}/{mm}/{attachment_id}.jpg`，原图 key 继续保持 `images/{yyyy}/{mm}/{attachment_id}.{ext}`。
  - `thumbnail_url` 由既有 `thumbnail_object_key` 推导；有缩略图时指向独立对象，没有时回退原图 URL。
  - 小图、WebP 或缩略图生成 / 上传失败时不阻断原图上传。
  - 本阶段不新增路由、不新增 migration、不引入第三方图片处理库，不做历史附件补生成、异步图片处理流水线或 WebP 解码缩略图。
  - 已同步 README、HTTP 合同、媒体架构文档、阶段文档和前端缺口台账。
  - 已通过 `go test ./internal/media/...`、`go test ./...`、`go build -buildvcs=false ./...`、API 合同校验、API schema 校验和 migration 合同校验。

### T36-001：结构化 `content_refs` 持久化

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 35 已完成`
- 目标：让帖子和评论的结构化内容引用从响应占位升级为真实持久化读写合同。
- 当前结论：
  - 新增 `post_content_refs` 和 `comment_content_refs` 表，字段为内容 ID、`position`、`kind`、`ref_id` 和 `created_at`，并随帖子 / 评论删除级联清理。
  - 帖子和评论发布 / 编辑请求新增可选 `content_refs`，响应继续返回 `content_refs`，但现在来自持久化记录而不是空数组占位。
  - `kind` 仅允许 `image`、`link_preview` 和 `embed`；`image` 的 `ref_id` 必须匹配已绑定图片附件 ID。
  - 编辑请求省略 `content_refs` 时保留原引用，显式传 `[]` 时清空原引用；有值时按请求顺序整体替换。
  - `link_preview` / `embed` 当前只保存稳定引用字符串，不新增预览或嵌入实体表；真实缩略图生成仍留作后续阶段。
  - 已同步 HTTP schema、migration 合同和目标包测试；全量验证在本轮收口执行。

### T35-001：社区设置和规则管理接口

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 34 已完成`
- 目标：补齐前端社区管理页需要的基础设置读取 / 更新和社区规则维护接口。
- 当前结论：
  - 新增 `community_rules` 表和 `CommunityRule` 领域模型；规则标题不能为空，`position >= 0`。
  - 新增 `GET /api/v1/communities/:slug/manage/settings` 和 `PATCH /api/v1/communities/:slug/manage/settings`；读取沿用社区管理上下文权限，更新仅 owner 可执行。
  - 新增 `GET/POST/PATCH/DELETE /api/v1/communities/:slug/manage/rules`；owner/moderator 可以维护社区规则。
  - 规则列表按 `position ASC, created_at ASC, id ASC` 返回，不增加唯一 position 约束，避免重排 / 交换时制造写入摩擦。
  - 当前 settings 更新只开放 `name` 和 `description`；社区头像、banner 因现有附件 owner 模型只覆盖 post/comment，留到媒体归属扩展后再做。
  - 本阶段已同步 HTTP route/schema/migration 合同和前端缺口台账。

### T34-001：@ 提及通知生产

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 33 已完成`
- 目标：补齐前端后端接口需求记录中 `mentions` 分类只有读取、缺少事件生产的问题。
- 当前结论：
  - `internal/mention` 负责从帖子 / 评论正文中提取合法 `@username`，归一为 `userdomain.Username` 并去重。
  - `notificationusecase.NotifyMentioned` 创建非聚合 `type=mention` 通知，限制 source 为 `post` 或 `comment`，并复用自通知跳过逻辑。
  - `PostUseCase.PublishPost` 和 `CommentUseCase.PublishComment` 会通知正文中被提及的 active 用户。
  - `PostUseCase.UpdatePost` 和 `CommentUseCase.UpdateComment` 只通知编辑后新增的提及。
  - 不存在用户、disabled 用户和自己提及自己不会产生有效通知。
  - 本阶段不新增路由、schema migration 或 response 字段；前端继续使用阶段 33 已有的通知分类读取接口展示 `mentions`。
  - 已通过目标包测试、`go test ./...` 和 `go build -buildvcs=false ./...`。

### T33-001：公开搜索和通知分类合同补齐

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 32 已完成`
- 目标：补齐前端信息架构目标合同中的公开搜索、通知分类筛选、未读分类摘要和全部已读能力，继续保持用户态通知接口 Bearer 边界。
- 当前结论：
  - `GET /api/v1/search` 已移入 `authhttp.OptionalAuth` 公开读取分组；搜索仍只返回 active public 社区和这些社区下的 visible 帖子。
  - 可选 Bearer 仍由 `authhttp.OptionalAuth` 处理：无 token 匿名读取，有格式错误、过期或签名错误 token 时返回 `unauthenticated`。
  - 通知列表新增 `category` 查询参数，支持 `all/replies/mentions/likes/system`，分类基于 `notifications.type` 白名单映射。
  - `GET /api/v1/notifications/unread-summary` 返回当前用户分类未读计数。
  - `POST /api/v1/notifications/read-all` 幂等标记当前用户所有未读通知为已读，并返回 `updated_count/read_at`。
  - `GET /api/v1/posts?source=recommended` 已落地 SQL 可解释推荐查询：匿名 hot+new 混排、社区 rank 去重，登录态按关注/互动社区加权。
  - 推荐流本轮未新增 schema migration；复杂赞通知聚合事件生产和图片后续清理策略继续留作后续阶段。
  - 已通过 Go 测试、Go 构建、合同脚本和真实 HTTP 验证。

### T32-001：保存、关注和评论投票合同补齐

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 31 已完成`
- 目标：为前端公开信息架构补齐保存帖子、关注社区和评论投票的真实后端合同，替换阶段 31 中的占位 viewer 字段。
- 当前结论：
  - 保存、关注和评论投票分别落在 post、community、comment 模块，不新增泛化 engagement 模块。
  - `save_count`、帖子/评论投票计数、社区 `member_count/post_count` 继续作为公开计数字段返回。
  - `is_saved`、`viewer_is_following`、`my_vote` 是 viewer 字段；匿名读取返回 false/0，有效 Bearer 返回当前用户状态，无效 Bearer 不降级。
  - `community_follows` 不计入 `member_count`；关注和成员关系保持分离。
  - 当前不实现保存/关注通知聚合、不实现推荐流或复杂排序。
  - 已通过 Go 测试、Go 构建、合同脚本、migration 执行和真实 HTTP 冒烟。

### T31-001：公开用户主页和读模型摘要字段补齐

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 30 已完成`
- 目标：补齐前端信息架构下一步需要的公开读取能力和摘要字段，同时继续把写操作和权限操作留在 Bearer 保护下。
- 当前结论：
  - `viewer_permissions` 采用 bool 对象结构，前端用于入口显隐和禁用态，权限最终裁决仍以后端写接口为准。
  - 当前用户主页字段都可以公开；数据库尚无 `display_name/avatar_url/headline/bio/badges/roles` 真实列时，后端只返回 username fallback、空字符串或空数组，不伪造资料能力。
  - `save_count=0` 和 `is_saved=false` 是当前没有保存表时的真实状态；后续保存能力单独阶段实现。
  - `nexus_markdown` 替换旧 `body_format=markdown` 合同；`content_refs` 先返回空数组。
  - `best/top/rising/t=...`、推荐流、保存、关注、评论投票、通知分类、链接解析、嵌入解析、积分和效果能力先记录合同，不在本阶段实现。
  - 已通过 Go 测试、Go 构建、合同脚本和真实 HTTP 冒烟。

### T30-001：未登录首页公开信息流合同补齐

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 29 已完成`
- 目标：让首页公开 feed 和帖子详情可以匿名读取，同时保留登录用户的 `my_vote` 视角和写操作认证边界。
- 完成结论：
  - 新增 `authhttp.OptionalAuth`：无 `Authorization` 时匿名通过；有合法 Bearer 时写入当前用户；有格式错误、过期或签名错误的 Bearer 时返回 `unauthenticated`。
  - `GET /api/v1/posts`、`GET /api/v1/posts/:id` 和 `GET /api/v1/communities/:slug/posts` 已注册到公开可选认证分组。
  - `posthttp` 读取 handler 允许空 viewer；usecase 空 `ViewerID` 时只返回投票统计，不查当前用户投票，`my_vote=0`。
  - `POST /api/v1/communities/:slug/posts`、编辑和删除帖子继续注册在 `RequireAuth` 分组；评论、投票、举报、审核和上传仍保持 Bearer。
  - 本阶段不新增 schema migration，不改变 response schema，不改变评论读取认证边界，不处理图片缩略图、附件 TTL 或孤儿附件清理策略。

### 2026-06-07：图片公开 URL 合同修复

- 状态：`DONE`
- 范围：修复本地开发图片链路和 R2 public base URL 防误配校验；不新增业务接口、不新增 schema migration、不迁移历史附件 URL。
- 结论：
  - 本地 `.env` 已切回 `OBJECT_STORAGE_PROVIDER=local`，新上传图片会写入 `var/uploads` 并通过 `/uploads/...` 静态路由读取。
  - 已创建本地 `var/uploads` 目录。
  - `OBJECT_STORAGE_PUBLIC_BASE_URL` 在 `provider=r2` 时不能使用 `*.r2.cloudflarestorage.com` S3 API endpoint；配置校验会在启动阶段拒绝该误配。
  - Stage 15 R2 smoke 和前端 `check:v2-path` 已补上传后真实读取 `attachment.url` 的验收要求，必须返回 HTTP 200 和 `image/*`。
  - 若后续继续使用 R2，需要先在 Cloudflare 开启 R2 public development URL 或绑定自定义域名，再把 `OBJECT_STORAGE_PUBLIC_BASE_URL` 改成该公开读取 base URL。

### 2026-06-03：GHCR 镜像包私有可见性收口

- 状态：`DONE`
- 范围：只修改 deploy GitHub Actions 和部署文档，不改变业务接口、schema migration 或运行时业务语义。
- 结论：
  - `.github/workflows/deploy.yml` 已在 build/push 后校验 GHCR package visibility 必须为 `private`；如果 package 被手动改成 public，发布流水线会失败。
  - Docker image 已加 `org.opencontainers.image.source`、`revision` 和 `version` labels，用于稳定关联源码和版本。
  - 手动 SSH 部署时会先在服务器上用当前 workflow token 登录 GHCR，再拉取 private image。
  - `docs/deployment.md` 已同步 private package 约束和服务器拉取私有镜像的凭据边界。

### T29-001：前端 V2 社区申请审核入口合同补齐

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 28 已完成`
- 目标：让前端审核台不再需要手动输入申请 ID 才能操作 approve/reject，并能可靠判断当前用户是否为平台 staff。
- 完成结论：
  - `CommunityApplicationUseCase` 已新增申请列表和详情读取能力，staff 判断仍在 usecase 层。
  - `PostgresApplicationRepository` 已支持按申请状态分页读取，按 `created_at DESC, id DESC` 排序。
  - `communityhttp` 已新增 `GET /api/v1/community-applications` 和 `GET /api/v1/community-applications/:id`。
  - `GET /api/v1/me` 已新增 `is_platform_staff`；当前用户查询仍先验证用户存在且可登录，再读取平台 staff 标记。
  - 合同文档已记录新增路由、查询参数和响应字段。
  - 已通过 `go test ./...`、`go build -buildvcs=false ./...`、`verify-api-contract-doc.ps1` 和 `verify-api-schema-doc.ps1`。
  - 聚合基线 `verify-current-baseline.ps1 -SkipHttpSmoke -R2Mode Skip` 的合同/config/migration-contract 步骤已通过，但本机 PostgreSQL 未监听 `127.0.0.1:5432`，在 `go run ./cmd/migrate up` 步骤中断；本轮未新增 schema migration。

### T28-001：部署骨架和 CI/CD 基础

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 27 已完成`
- 目标：补齐项目发布前的最小部署骨架。
- 完成结论：
  - 已新增 Dockerfile、生产 compose、生产 env 示例、CI workflow 和手动 deploy workflow。
  - `docs/deployment.md` 已记录本地模拟生产、CI、GHCR 镜像、SSH 部署和 migration 回滚边界。
  - `scripts/verify-current-baseline.ps1` 已改为自动选择 `pwsh` 或 `powershell`，并读取 `docs/contracts/` 下的可提交合同文档。

### 2026-06-03：评论树 warning 收口与内容生命周期合同复核

- 状态：`DONE`
- 范围：只复核阶段 13 评论树合同和阶段 14 内容生命周期合同，不进入新产品阶段。
- 结论：
  - 已先用前端 `npm run check:main-path` 复现评论树 warning：旧本地 API 进程返回子评论先于根评论。
  - 旧 API 进程启动于 2026-06-02；用当前源码重新启动 API 后，前端严格 `npm run check:main-path` 通过且无 warning。
  - 当前源码的 `view=tree` 路径仍由 `commentusecase.buildCommentTree` 输出前序遍历扁平数组，父评论先于子评论。
  - `PATCH/DELETE /api/v1/posts/:id` 和 `PATCH/DELETE /api/v1/comments/:id` 合同已足够前端实现；请求体、成功响应、`204 No Content`、软删除语义和错误码已在 README 与 `docs/internal/architecture/content-lifecycle.md` 收口。
  - 本次未修改业务代码、接口、错误码、响应格式或 schema migration。
- 验证：
  - 前端 `npm run check:main-path` 严格模式通过。
  - `go test ./...` 通过。
  - `go build -buildvcs=false ./...` 通过。
  - `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-current-baseline.ps1 -SkipHttpSmoke -R2Mode Skip` 通过。

## 阶段 28 工单

### T28-001：部署骨架和 CI/CD 基础

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 27 已完成`
- 目标：补齐项目发布前的最小部署骨架，确保后端可以被 Docker 构建、用生产 compose 本机模拟、通过 CI 校验，并为后续服务器部署留出手动 deploy workflow。
- 交付物：
  - `Dockerfile`
  - `.dockerignore`
  - `docker-compose.prod.yml`
  - `.env.production.example`
  - `.github/workflows/ci.yml`
  - `.github/workflows/deploy.yml`
  - `docs/deployment.md`
  - `scripts/verify-current-baseline.ps1`
  - `tasks.md`
  - `docs/internal/README.md`
  - `.ai/slices/stage-28-deployment-skeleton/README.md`
  - `.ai/slices/stage-28-deployment-skeleton/01-deployment-skeleton.md`
- 完成标准：
  - Dockerfile 构建出包含 API binary、migrate binary 和 migrations 的运行镜像。
  - 生产 compose 使用独立 migration job，在 API 启动前执行 `migrate up`。
  - `.env.production.example` 不包含真实密钥，并能作为本机生产模拟模板。
  - CI workflow 在 push/PR 时启动 PostgreSQL、运行 `-SkipHttpSmoke -R2Mode Skip` 快速基线并构建 Docker image。
  - deploy workflow 在 tag/manual 时构建并推送 GHCR image，SSH 部署只在手动输入打开时执行。
  - 文档明确服务器只部署 main tag / Docker image tag，不跟随开发分支。
  - 不新增业务接口，不改变业务运行时语义，不新增 schema migration。
- 结论：
  - 部署骨架已补齐。
  - 后续可以先本机 Docker 模拟生产，再接真实服务器、域名、HTTPS 和 R2 凭据。

## 阶段 28 复盘摘要

阶段 28 已完成：

- 新增 Docker 构建和生产 compose 骨架。
- 新增生产环境变量示例。
- 新增 GitHub Actions CI 和 build/deploy workflow。
- 部署文档已从版本切换规则扩展为可执行部署骨架说明。
- 生产 compose 已使用独立 project name，避免和本地开发 PostgreSQL volume 混用。
- 可被脚本校验的合同文档已迁到 `docs/contracts/` 并提交，CI 不再依赖 ignored 的 `docs/internal/`。
- 本机已验证 `docker build -t cumt-nexus-api:local .`、生产 compose 启动、migration version 9 dirty=false、`/healthz`、`docker compose --env-file .env.production.example -f docker-compose.prod.yml config`、`go test ./...`、`go build -buildvcs=false ./...` 和 `git diff --check`。

阶段 28 遗留限制：

- 本阶段不实际配置服务器、域名、HTTPS、GHCR 权限、SSH secrets 或生产 R2 凭据。
- 本阶段不新增业务接口、schema migration 或产品语义。
- 本阶段不引入 Kubernetes、Terraform 或复杂平台化部署。
- 生产 compose 当前只验证 local object storage 模式；真实 R2 上传和真实服务器部署仍未验证。

阶段 28 review packet：

1. 完成能力：项目具备 Docker image 构建、生产 compose 模拟、CI 快速基线和 GHCR/SSH deploy workflow 骨架。
2. 修改文件：`Dockerfile`、`.dockerignore`、`docker-compose.prod.yml`、`.env.production.example`、`.github/workflows/ci.yml`、`.github/workflows/deploy.yml`、`docs/deployment.md`、`scripts/verify-current-baseline.ps1`、`.gitignore`、`tasks.md`、`docs/internal/README.md`、`.ai/slices/stage-28-deployment-skeleton/*`。
3. 新增或修改接口：无业务接口变更。
4. 完整调用链：Git tag/workflow dispatch -> deploy workflow -> Docker buildx build/push -> GHCR image -> optional SSH -> server `docker compose pull/up`; local simulation -> `docker compose.prod` -> postgres -> migrate job -> api -> `/healthz`。
5. 错误码映射：业务错误码无变化。
6. 测试覆盖和未覆盖内容：已验证本地/CI 快速基线 `-SkipHttpSmoke -R2Mode Skip`、Go 测试、Go 构建、compose config、Docker build、生产 compose 启动、migration version 和 `/healthz`；Stage 13/14 HTTP smoke 和 Stage 15 R2 smoke 在本轮按参数显式跳过；未覆盖真实服务器部署、域名 HTTPS、GHCR 权限、SSH secrets 和真实 R2 上传。
7. 绕过的非主链路能力：Kubernetes、Terraform、蓝绿发布、自动回滚、数据库备份自动化和生产监控告警。
8. 下一阶段建议：补真实服务器部署说明、反向代理和 HTTPS；接入 GHCR 权限与 SSH secrets 后做一次真实服务器 smoke。

## 阶段 27 工单

### T27-001：API 请求必填字段契约校验和文档收口

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 26 已完成`
- 目标：补强阶段 22 API schema 契约，让 `docs/contracts/http-api-schema.md` 的“请求必填字段清单”也能被脚本校验，避免 handler request struct 中的 `binding:"required"` 和文档之间漂移。
- 交付物：
  - `scripts/verify-api-schema-doc.ps1`
  - `scripts/verify-current-baseline.ps1`
  - `README.md`
  - `tasks.md`
  - `docs/contracts/http-api-schema.md`
  - `docs/internal/README.md`
  - `docs/internal/engineering/workflow.md`
  - `docs/internal/architecture/community-v1.md`
  - `.ai/slices/stage-27-api-required-fields-contract/README.md`
  - `.ai/slices/stage-27-api-required-fields-contract/01-api-required-fields-contract.md`
- 完成标准：
  - 校验脚本读取 delivery 层 handler JSON struct 的 package、Go type、JSON 字段和 `binding:"required"` 字段。
  - `http-api-schema.md` 中的“请求必填字段清单”缺少带 required tag 的 request schema 时失败。
  - 文档保留不再有 required tag 的过期 schema 时失败。
  - 文档必填字段集合与 handler request struct 实际 required 字段集合不一致时失败。
  - 继续校验 handler JSON 字段清单和接口 schema 映射。
  - 不新增业务接口，不改变请求校验语义，不改变成功或错误响应格式。
  - 不新增第三方依赖，不新增 schema migration，不进入新产品语义。
- 结论：
  - `scripts/verify-api-schema-doc.ps1` 已补齐 required field contract 校验。
  - 当前覆盖 46 个 handler JSON schema、34 条接口 schema 映射和 10 个带 `binding:"required"` 的 request schema。
  - `verify-current-baseline.ps1` 已通过 API schema fields/routes/required inventory。

## 阶段 27 复盘摘要

阶段 27 已完成：

- API schema 契约校验脚本已补强请求必填字段清单校验。
- `http-api-schema.md` 已记录该脚本同时校验 handler JSON 字段清单、接口 schema 映射和请求必填字段清单。
- README、`tasks.md`、`docs/internal/README.md`、`docs/internal/engineering/workflow.md`、`docs/internal/architecture/community-v1.md` 和 `.ai/slices/stage-27-api-required-fields-contract/` 已同步。

阶段 27 验证结果：

- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-api-schema-doc.ps1` 通过，覆盖 46 个 handler JSON schema、34 条接口 schema 映射和 10 个 required 字段 request schema。
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-current-baseline.ps1 -SkipHttpSmoke -R2Mode Skip` 通过。

阶段 27 遗留限制：

- 本阶段只校验 Gin struct tag 中显式存在的 `binding:"required"`。
- 本阶段不校验请求字段完整业务规则，例如 slug 格式、字符串长度、投票值枚举、分页范围或跨字段约束。
- 本阶段不新增业务接口、请求校验语义、错误码、响应格式、第三方依赖或 schema migration。
- 本阶段不运行真实 R2 上传；`.env` 中本地 R2 凭据不提交。
- 本阶段不进入新的 feed、vote、moderation、notification 或 search 产品语义。

阶段 27 review packet：

1. 完成能力：补强 `scripts/verify-api-schema-doc.ps1`，让 API schema 文档的请求必填字段清单与 handler request struct 的 `binding:"required"` 同步校验。
2. 修改文件：`scripts/verify-api-schema-doc.ps1`、`scripts/verify-current-baseline.ps1`、`README.md`、`tasks.md`、`docs/contracts/http-api-schema.md`、`docs/internal/README.md`、`docs/internal/engineering/workflow.md`、`docs/internal/architecture/community-v1.md`、`.ai/slices/stage-27-api-required-fields-contract/*`。
3. 新增或修改接口：无业务接口变更；无请求校验语义或响应格式变更。
4. 完整调用链：`verify-current-baseline.ps1` -> `verify-api-schema-doc.ps1` -> scan delivery handler JSON structs -> extract JSON fields and `binding:"required"` fields -> read `http-api-schema.md` -> compare schema fields, route mappings and required-field rows。
5. 错误码映射：业务错误码无变化；JSON bind 失败仍按现有 handler 逻辑映射为 `invalid_argument`。
6. 测试覆盖和未覆盖内容：API schema 字段清单、route mapping 和 required field contract 已校验；快速基线已覆盖 API contract、API schema 契约、HTTP 错误契约、配置契约、migration 契约、测试、构建和 migration；未覆盖业务枚举、数值范围、跨字段约束、Stage 13/14 HTTP smoke 和真实 R2 上传。
7. 绕过的非主链路能力：完整 OpenAPI 生成、请求业务规则矩阵、示例请求/响应快照、前端 SDK 生成、真实 R2 上传和新产品语义。
8. 下一阶段建议：如继续工程契约，可补请求业务规则矩阵或示例响应快照；如果进入新的 feed、vote、moderation、notification 或 search 产品语义，需要先确认边界。

## 阶段 26 工单

### T26-001：API 查询参数契约校验和文档收口

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 25 已完成`
- 目标：补强阶段 17 API contract 契约，让 `docs/contracts/http-api-contract.md` 的“查询参数约定”表也能被脚本校验，避免 handler query key 和文档之间漂移。
- 交付物：
  - `scripts/verify-api-contract-doc.ps1`
  - `scripts/verify-current-baseline.ps1`
  - `README.md`
  - `tasks.md`
  - `docs/contracts/http-api-contract.md`
  - `docs/internal/README.md`
  - `docs/internal/engineering/workflow.md`
  - `docs/internal/architecture/community-v1.md`
  - `.ai/slices/stage-26-api-query-contract/README.md`
  - `.ai/slices/stage-26-api-query-contract/01-api-query-contract.md`
- 完成标准：
  - 校验脚本读取源码注册路由、Auth 边界和 handler query key。
  - handler 中 `c.Query(...)`、`c.DefaultQuery(...)`、`c.GetQuery(...)` 和 `parseOptionalIntQuery(c, "...")` 等 query key 被纳入实际清单。
  - `http-api-contract.md` 中的“查询参数约定”表缺少带 query 的路由时失败。
  - 文档保留不再读取 query 的过期路由时失败。
  - 文档参数集合与 handler 实际读取集合不一致时失败。
  - 不新增业务接口，不改变查询参数语义，不改变成功或错误响应格式。
  - 不新增第三方依赖，不新增 schema migration，不进入新产品语义。
- 结论：
  - `scripts/verify-api-contract-doc.ps1` 已补齐 query param contract 校验。
  - 当前覆盖 34 条路由、Auth 边界和 6 条带查询参数的路由。
  - `verify-current-baseline.ps1` 已通过 API contract route/auth/query inventory。

## 阶段 26 复盘摘要

阶段 26 已完成：

- API contract 契约校验脚本已补强查询参数清单校验。
- `http-api-contract.md` 已记录该脚本同时校验 method/path、Auth 列和查询参数表。
- README、`tasks.md`、`docs/internal/README.md`、`docs/internal/engineering/workflow.md`、`docs/internal/architecture/community-v1.md` 和 `.ai/slices/stage-26-api-query-contract/` 已同步。

阶段 26 验证结果：

- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-api-contract-doc.ps1` 通过，覆盖 34 条路由、Auth 边界和 6 条 query 参数路由。
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-current-baseline.ps1 -SkipHttpSmoke -R2Mode Skip` 通过。

阶段 26 遗留限制：

- 本阶段不校验查询参数枚举值、数值范围或默认值语义。
- 本阶段不校验完整 request/response schema；字段和 route mapping 仍由 `verify-api-schema-doc.ps1` 覆盖。
- 本阶段不校验每个业务权限场景的 staff、作者或资源可见性判断。
- 本阶段不新增业务接口、查询参数语义、错误码、响应格式、第三方依赖或 schema migration。
- 本阶段不运行真实 R2 上传；`.env` 中本地 R2 凭据不提交。
- 本阶段不进入新的 feed、vote、moderation、notification 或 search 产品语义。

阶段 26 review packet：

1. 完成能力：补强 `scripts/verify-api-contract-doc.ps1`，让 API contract 文档的查询参数表与 handler query key 读取同步校验。
2. 修改文件：`scripts/verify-api-contract-doc.ps1`、`scripts/verify-current-baseline.ps1`、`README.md`、`tasks.md`、`docs/contracts/http-api-contract.md`、`docs/internal/README.md`、`docs/internal/engineering/workflow.md`、`docs/internal/architecture/community-v1.md`、`.ai/slices/stage-26-api-query-contract/*`。
3. 新增或修改接口：无业务接口变更；无查询参数语义或响应格式变更。
4. 完整调用链：`verify-current-baseline.ps1` -> `verify-api-contract-doc.ps1` -> read route/auth facts -> read handler function bodies -> extract query keys -> compare `http-api-contract.md` query parameter table。
5. 错误码映射：业务错误码无变化；非法查询参数仍按现有 handler/usecase 逻辑映射为 `invalid_argument`。
6. 测试覆盖和未覆盖内容：API route/auth/query contract 已校验；快速基线已覆盖 API contract、API schema 契约、HTTP 错误契约、配置契约、migration 契约、测试、构建和 migration；未覆盖 query 参数枚举/范围/默认值语义、Stage 13/14 HTTP smoke 和真实 R2 上传。
7. 绕过的非主链路能力：查询参数语义矩阵、示例响应快照、OpenAPI 生成、前端 SDK 生成、真实 R2 上传和新产品语义。
8. 下一阶段建议：如继续工程契约，可补查询参数语义矩阵或示例响应快照；如果进入新的 feed、vote、moderation、notification 或 search 产品语义，需要先确认边界。

## 阶段 25 工单

### T25-001：API 认证边界契约校验和文档收口

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 24 已完成`
- 目标：补强阶段 17 API contract 契约，让 `docs/contracts/http-api-contract.md` 路由表中的 Auth 列也能被脚本校验，避免认证边界在源码和文档之间漂移。
- 交付物：
  - `scripts/verify-api-contract-doc.ps1`
  - `scripts/verify-current-baseline.ps1`
  - `README.md`
  - `tasks.md`
  - `docs/contracts/http-api-contract.md`
  - `docs/internal/README.md`
  - `docs/internal/engineering/workflow.md`
  - `docs/internal/architecture/community-v1.md`
  - `.ai/slices/stage-25-api-auth-boundary-contract/README.md`
  - `.ai/slices/stage-25-api-auth-boundary-contract/01-api-auth-boundary-contract.md`
- 完成标准：
  - 校验脚本读取源码注册路由并为每条路由形成 Auth 边界。
  - `/healthz` 映射为 `public`。
  - `GET /uploads/*filepath` 映射为 `public, local only`。
  - `POST /api/v1/auth/register` 和 `POST /api/v1/auth/login` 映射为 `public`。
  - 通过 `authhttp.RequireAuth` 保护分组注册的 `/api/v1` 业务接口映射为 `Bearer`。
  - 文档缺失路由、保留过期路由或 Auth 列与源码边界不一致时失败。
  - 不新增业务接口，不改变认证中间件语义，不改变成功或错误响应格式。
  - 不新增第三方依赖，不新增 schema migration，不进入新产品语义。
- 结论：
  - `scripts/verify-api-contract-doc.ps1` 已补齐 Auth 边界校验。
  - 当前覆盖 34 条路由和 Auth 边界。
  - `verify-current-baseline.ps1` 已通过 API contract route/auth inventory。

## 阶段 25 复盘摘要

阶段 25 已完成：

- API contract 契约校验脚本已补强 Auth 边界校验。
- `http-api-contract.md` 已记录该脚本同时校验 method/path 和 Auth 列。
- README、`tasks.md`、`docs/internal/README.md`、`docs/internal/engineering/workflow.md`、`docs/internal/architecture/community-v1.md` 和 `.ai/slices/stage-25-api-auth-boundary-contract/` 已同步。

阶段 25 验证结果：

- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-api-contract-doc.ps1` 通过，覆盖 34 条路由和 Auth 边界。
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-current-baseline.ps1 -SkipHttpSmoke -R2Mode Skip` 通过。

阶段 25 遗留限制：

- 本阶段不校验完整 request/response schema；字段和 route mapping 仍由 `verify-api-schema-doc.ps1` 覆盖。
- 本阶段不校验每个业务权限场景的 staff、作者或资源可见性判断。
- 本阶段不新增业务接口、认证语义、错误码、响应格式、第三方依赖或 schema migration。
- 本阶段不运行真实 R2 上传；`.env` 中本地 R2 凭据不提交。
- 本阶段不进入新的 feed、vote、moderation、notification 或 search 产品语义。

阶段 25 review packet：

1. 完成能力：补强 `scripts/verify-api-contract-doc.ps1`，让 API contract 文档的 Auth 列与源码路由边界同步校验。
2. 修改文件：`scripts/verify-api-contract-doc.ps1`、`scripts/verify-current-baseline.ps1`、`README.md`、`tasks.md`、`docs/contracts/http-api-contract.md`、`docs/internal/README.md`、`docs/internal/engineering/workflow.md`、`docs/internal/architecture/community-v1.md`、`.ai/slices/stage-25-api-auth-boundary-contract/*`。
3. 新增或修改接口：无业务接口变更；无认证中间件语义或响应格式变更。
4. 完整调用链：`verify-current-baseline.ps1` -> `verify-api-contract-doc.ps1` -> read health/static route facts -> read auth public route file -> read protected route files -> compare `http-api-contract.md` method/path/Auth table。
5. 错误码映射：业务错误码无变化；认证失败仍统一映射为 `unauthenticated`。
6. 测试覆盖和未覆盖内容：API route/auth contract 已校验；快速基线已覆盖 API 路由与认证边界契约、API schema 契约、HTTP 错误契约、配置契约、migration 契约、测试、构建和 migration；未覆盖业务权限矩阵、Stage 13/14 HTTP smoke 和真实 R2 上传。
7. 绕过的非主链路能力：业务权限矩阵、示例响应快照、OpenAPI 生成、前端 SDK 生成、真实 R2 上传和新产品语义。
8. 下一阶段建议：如继续工程契约，可补业务权限矩阵或示例响应快照；如果进入新的 feed、vote、moderation、notification 或 search 产品语义，需要先确认边界。

## 阶段 24 工单

### T24-001：API schema 路由映射校验和文档收口

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 23 已完成`
- 目标：补强阶段 22 API schema 契约，让 `docs/contracts/http-api-schema.md` 的“接口 Schema 映射”表也能被脚本校验，避免路由契约和 schema 文档之间漂移。
- 交付物：
  - `scripts/verify-api-schema-doc.ps1`
  - `README.md`
  - `tasks.md`
  - `docs/contracts/http-api-schema.md`
  - `docs/internal/README.md`
  - `docs/internal/engineering/workflow.md`
  - `docs/internal/architecture/community-v1.md`
  - `.ai/slices/stage-24-api-schema-route-map/README.md`
  - `.ai/slices/stage-24-api-schema-route-map/01-api-schema-route-map.md`
- 完成标准：
  - 校验脚本读取 `docs/contracts/http-api-contract.md` 的路由表。
  - 校验脚本读取 `docs/contracts/http-api-schema.md` 的“接口 Schema 映射”表。
  - 路由缺失映射、过期映射、schema 引用不存在、成功状态码不在 `200/201/204` 或 `204` 带 success schema 时失败。
  - 继续校验 delivery 层 handler JSON struct 和 schema 文档字段清单一致。
  - 不新增业务接口，不生成 OpenAPI，不改变成功或错误响应格式。
  - 不新增第三方依赖，不新增 schema migration，不进入新产品语义。
- 结论：
  - `scripts/verify-api-schema-doc.ps1` 已补齐 route mapping 校验。
  - 当前覆盖 34 条路由映射和 46 个 handler JSON schema。
  - `verify-current-baseline.ps1` 已通过既有 API schema 契约入口间接覆盖本阶段新增校验。

## 阶段 24 复盘摘要

阶段 24 已完成：

- API schema 契约校验脚本已补强接口 schema 映射校验。
- `http-api-schema.md` 已记录该脚本同时校验 handler JSON 字段清单和路由映射。
- README、`tasks.md`、`docs/internal/README.md`、`docs/internal/engineering/workflow.md`、`docs/internal/architecture/community-v1.md` 和 `.ai/slices/stage-24-api-schema-route-map/` 已同步。

阶段 24 验证结果：

- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-api-schema-doc.ps1` 通过，覆盖 34 条路由映射和 46 个 handler JSON schema。
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-current-baseline.ps1 -SkipHttpSmoke -R2Mode Skip` 通过。

阶段 24 遗留限制：

- 本阶段不生成 OpenAPI、Swagger 或前端 SDK。
- 本阶段不校验完整业务枚举、数值范围、嵌套类型语义、完整示例或错误消息全文。
- 本阶段不新增业务接口、错误码、响应格式、第三方依赖或 schema migration。
- 本阶段不运行真实 R2 上传；`.env` 中本地 R2 凭据不提交。
- 本阶段不进入新的 feed、vote、moderation、notification 或 search 产品语义。

阶段 24 review packet：

1. 完成能力：补强 `scripts/verify-api-schema-doc.ps1`，让 schema 文档的接口映射与路由契约、schema 清单同步校验。
2. 修改文件：`scripts/verify-api-schema-doc.ps1`、`README.md`、`tasks.md`、`docs/contracts/http-api-schema.md`、`docs/internal/README.md`、`docs/internal/engineering/workflow.md`、`docs/internal/architecture/community-v1.md`、`.ai/slices/stage-24-api-schema-route-map/*`。
3. 新增或修改接口：无业务接口变更；无成功或错误响应格式变更。
4. 完整调用链：`verify-current-baseline.ps1` -> `verify-api-schema-doc.ps1` -> read `http-api-contract.md` routes -> read `http-api-schema.md` route mappings -> scan delivery handler JSON structs -> compare route mappings and schema refs。
5. 错误码映射：业务错误码无变化；本阶段只在脚本层发现文档漂移时失败。
6. 测试覆盖和未覆盖内容：API schema 字段清单和 route mapping 已校验；快速基线已覆盖 API 路由契约、API schema 契约、HTTP 错误契约、配置契约、migration 契约、测试、构建和 migration；未覆盖完整 OpenAPI、业务枚举、数值范围、错误消息全文、Stage 13/14 HTTP smoke 和真实 R2 上传。
7. 绕过的非主链路能力：OpenAPI 生成、前端 SDK 生成、示例响应快照、业务规则矩阵、真实 R2 上传和新产品语义。
8. 下一阶段建议：如继续工程契约，可补示例响应快照或业务错误场景矩阵；如果进入新的 feed、vote、moderation、notification 或 search 产品语义，需要先确认边界。

## 阶段 23 工单

### T23-001：HTTP 错误契约校验和文档收口

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 22 已完成`
- 目标：让 `apperr` 错误码、HTTP 状态码映射和统一错误响应形状可以被脚本校验，避免错误契约在代码和文档之间漂移。
- 交付物：
  - `scripts/verify-http-error-contract-doc.ps1`
  - `scripts/verify-current-baseline.ps1`
  - `docs/contracts/http-error-handling.md`
  - `README.md`
  - `tasks.md`
  - `docs/internal/README.md`
  - `docs/internal/engineering/workflow.md`
  - `docs/internal/architecture/community-v1.md`
  - `.ai/slices/stage-23-http-error-contract/README.md`
  - `.ai/slices/stage-23-http-error-contract/01-http-error-contract-check.md`
- 完成标准：
  - 校验脚本读取 `internal/apperr/apperr.go` 的错误码常量集合。
  - 校验脚本读取 `internal/platform/httpserver/error.go` 的 app error 到 HTTP status 映射。
  - 校验脚本确认 `internal` 和未知错误兜底为 `500 internal server error`。
  - 校验脚本确认 `internal/platform/httpserver/response.go` 的错误响应 JSON 字段包含 `error`、`code` 和 `message`。
  - 校验脚本读取 `docs/contracts/http-error-handling.md` 的错误码表，并在文档缺失、过期或状态码不一致时失败。
  - 当前基线脚本纳入 HTTP 错误契约校验。
  - 不新增错误码，不改变错误响应格式，不改变认证错误语义。
  - 不新增第三方依赖，不新增 schema migration。
  - 不进入新的产品语义阶段。
- 结论：
  - `scripts/verify-http-error-contract-doc.ps1` 已新增并已通过，当前覆盖 6 个错误码。
  - `scripts/verify-current-baseline.ps1` 已加入 HTTP 错误契约校验。

## 阶段 23 复盘摘要

阶段 23 已完成：

- HTTP 错误契约校验脚本 `scripts/verify-http-error-contract-doc.ps1`。
- 当前基线脚本已加入 HTTP 错误契约校验。
- `docs/contracts/http-error-handling.md` 已记录校验入口和校验边界。
- README、`tasks.md`、`docs/internal/README.md`、`docs/internal/engineering/workflow.md`、`docs/internal/architecture/community-v1.md` 和 `.ai/slices/stage-23-http-error-contract/` 已同步。

阶段 23 验证结果：

- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-http-error-contract-doc.ps1` 通过，覆盖 6 个错误码和 `error/code/message` 响应形状。
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-current-baseline.ps1 -SkipHttpSmoke -R2Mode Skip` 通过，覆盖 API 路由契约、API schema 契约、HTTP 错误契约、配置清单契约、配置语义契约、migration 契约、`go test ./...`、`go build -buildvcs=false ./...` 和 `go run ./cmd/migrate up`。

阶段 23 遗留限制：

- 本阶段不新增错误码。
- 本阶段不改变错误响应格式或认证错误语义。
- 本阶段不校验每个业务场景的错误消息全文。
- 本阶段不运行 Stage 13/14 HTTP smoke。
- 本阶段不验证真实 R2 上传。
- 本阶段不进入新的 feed、vote、moderation、notification 或 search 产品语义。

阶段 23 review packet：

1. 完成能力：新增 HTTP 错误契约校验，校验错误码集合、HTTP 状态码映射和统一错误响应形状，并纳入当前基线脚本。
2. 修改文件：`scripts/verify-http-error-contract-doc.ps1`、`scripts/verify-current-baseline.ps1`、`docs/contracts/http-error-handling.md`、`README.md`、`tasks.md`、`docs/internal/README.md`、`docs/internal/engineering/workflow.md`、`docs/internal/architecture/community-v1.md`、`.ai/slices/stage-23-http-error-contract/*`。
3. 新增或修改接口：无业务接口变更；无错误码或错误响应格式变更。
4. 完整调用链：`verify-current-baseline.ps1` -> `verify-http-error-contract-doc.ps1` -> read `apperr.go` constants -> read `httpserver/error.go` mappings -> read `httpserver/response.go` JSON tags -> compare `http-error-handling.md` table。
5. 错误码映射：`invalid_argument=400`、`unauthenticated=401`、`forbidden=403`、`not_found=404`、`conflict=409`、`internal=500`。
6. 测试覆盖和未覆盖内容：HTTP 错误契约已校验；快速基线已覆盖 API 路由契约、API schema 契约、HTTP 错误契约、配置契约、migration 契约、测试、构建和 migration；未覆盖每个业务场景错误消息全文、Stage 13/14 HTTP smoke 和真实 R2 上传。
7. 绕过的非主链路能力：错误消息全文快照、业务场景级错误矩阵、真实 R2 上传、新产品语义。
8. 下一阶段建议：可以补业务场景级错误矩阵或示例响应快照；若要进入新的 feed、vote、moderation、notification 或 search 产品语义，需要先确认边界。

## 阶段 22 工单

### T22-001：HTTP API schema 契约快照和字段清单校验

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 21 已完成`
- 目标：让当前 HTTP API 的成功响应和 JSON 请求体有可读的轻量 schema 快照，并通过脚本校验文档中的 handler JSON 字段名没有漂移。
- 交付物：
  - `docs/contracts/http-api-schema.md`
  - `scripts/verify-api-schema-doc.ps1`
  - `scripts/verify-current-baseline.ps1`
  - `README.md`
  - `tasks.md`
  - `docs/internal/README.md`
  - `docs/internal/engineering/workflow.md`
  - `docs/internal/architecture/community-v1.md`
  - `.ai/slices/stage-22-api-schema-contract/README.md`
  - `.ai/slices/stage-22-api-schema-contract/01-api-schema-contract.md`
- 完成标准：
  - 文档记录每个当前接口的 request 类型、success response 类型和成功 HTTP status。
  - 文档记录 delivery 层 handler JSON struct 的 package、Go type 和 JSON 字段名。
  - 校验脚本扫描 `internal/*/delivery/*http` 下非测试 Go 文件中的 JSON struct。
  - 校验脚本在文档缺少 schema、文档存在过期 schema 或字段顺序/名称不一致时失败。
  - 当前基线脚本纳入 API schema 契约校验。
  - 不新增或修改业务接口，不生成 OpenAPI，不生成前端 SDK。
  - 不改变错误码、错误响应格式、成功响应字段或运行时行为。
  - 不新增第三方依赖，不新增 schema migration。
  - 不进入新的产品语义阶段。
- 结论：
  - `docs/contracts/http-api-schema.md` 已新增。
  - `scripts/verify-api-schema-doc.ps1` 已新增并已通过，当前覆盖 46 个 handler JSON struct。
  - `scripts/verify-current-baseline.ps1` 已加入 API schema 契约校验。

## 阶段 22 复盘摘要

阶段 22 已完成：

- HTTP API schema 契约快照 `docs/contracts/http-api-schema.md`。
- API schema 字段清单校验脚本 `scripts/verify-api-schema-doc.ps1`。
- 当前基线脚本已加入 API schema 契约校验。
- README、`tasks.md`、`docs/internal/README.md`、`docs/internal/engineering/workflow.md`、`docs/internal/architecture/community-v1.md`、`docs/contracts/http-api-contract.md` 和 `.ai/slices/stage-22-api-schema-contract/` 已同步。

阶段 22 验证结果：

- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-api-schema-doc.ps1` 通过，覆盖 46 个 handler JSON struct。
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-current-baseline.ps1 -SkipHttpSmoke -R2Mode Skip` 通过，覆盖 API 路由契约、API schema 契约、配置清单契约、配置语义契约、migration 契约、`go test ./...`、`go build -buildvcs=false ./...` 和 `go run ./cmd/migrate up`。

阶段 22 遗留限制：

- 本阶段不生成 OpenAPI、Swagger 或前端 SDK。
- 本阶段不校验完整业务枚举、数值范围、错误消息全文或示例响应全文。
- 本阶段不运行 Stage 13/14 HTTP smoke。
- 本阶段不验证真实 R2 上传。
- 本阶段不进入新的 feed、vote、moderation、notification 或 search 产品语义。

阶段 22 review packet：

1. 完成能力：新增 HTTP API schema 契约快照，新增 handler JSON 字段清单校验，并纳入当前基线脚本。
2. 修改文件：`docs/contracts/http-api-schema.md`、`scripts/verify-api-schema-doc.ps1`、`scripts/verify-current-baseline.ps1`、`docs/contracts/http-api-contract.md`、`README.md`、`tasks.md`、`docs/internal/README.md`、`docs/internal/engineering/workflow.md`、`docs/internal/architecture/community-v1.md`、`.ai/slices/stage-22-api-schema-contract/*`。
3. 新增或修改接口：无业务接口变更。
4. 完整调用链：`verify-current-baseline.ps1` -> `verify-api-contract-doc.ps1` -> `verify-api-schema-doc.ps1` -> scan delivery handler JSON structs -> compare `http-api-schema.md` schema inventory。
5. 错误码映射：业务错误码无变化；错误响应格式继续以 `docs/contracts/http-error-handling.md` 为准。
6. 测试覆盖和未覆盖内容：API schema 字段清单已校验；快速基线已覆盖 API 路由契约、API schema 契约、配置契约、migration 契约、测试、构建和 migration；未覆盖完整业务枚举、数值范围、错误消息全文、Stage 13/14 HTTP smoke 和真实 R2 上传。
7. 绕过的非主链路能力：OpenAPI 生成、前端 SDK 生成、错误消息全文快照、业务枚举/数值范围自动校验、真实 R2 上传、新产品语义。
8. 下一阶段建议：可以继续补错误消息全文快照或 schema 示例快照；若要进入新的 feed、vote、moderation、notification 或 search 产品语义，需要先确认边界。

## 阶段 21 工单

### T21-001：配置加载运行时契约测试和文档收口

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 20 已完成`
- 目标：让配置加载代码的默认值和 R2/local 条件必需行为有直接 Go 测试覆盖，不只依赖文档和静态脚本。
- 交付物：
  - `internal/platform/config/load_test.go`
  - `README.md`
  - `tasks.md`
  - `docs/internal/README.md`
  - `docs/internal/engineering/workflow.md`
  - `docs/contracts/configuration.md`
  - `docs/internal/architecture/community-v1.md`
  - `.ai/slices/stage-21-config-load-contract/README.md`
  - `.ai/slices/stage-21-config-load-contract/01-config-load-runtime-tests.md`
- 完成标准：
  - 测试清理并恢复当前进程内配置相关环境变量，避免本机环境污染。
  - 测试使用临时工作目录，避免仓库根目录 `.env` 干扰。
  - local 默认值测试覆盖 `APP_ENV`、timeout、PostgreSQL、HTTP、Log、Auth、Storage 和 Upload 默认值。
  - R2 成功加载测试覆盖 provider、endpoint、bucket、access key、secret key、public base URL、path-style 和 CORS list。
  - R2 缺凭据测试覆盖 bucket、access key 和 secret key 缺失错误。
  - primitive 解析失败测试覆盖 int、duration 和 bool。
  - 不新增配置变量，不改变运行时配置语义。
  - 不写入真实 R2 凭据。
  - 不进入新的产品语义阶段。
- 结论：
  - `internal/platform/config/load_test.go` 已新增并通过。
  - 测试使用 `t.Chdir(t.TempDir())` 和显式环境变量集合，避免本地 `.env` 或 shell 环境影响结果。
  - 本阶段未修改 `load.go` 或 `validate.go`，未改变运行时配置语义。

## 阶段 21 复盘摘要

阶段 21 已完成：

- 配置加载运行时契约测试 `internal/platform/config/load_test.go`。
- README、`tasks.md`、`docs/internal/README.md`、`docs/internal/engineering/workflow.md`、`docs/contracts/configuration.md`、`docs/internal/architecture/community-v1.md` 和 `.ai/slices/stage-21-config-load-contract/` 已同步。

阶段 21 验证结果：

- `go test ./internal/platform/config -v` 通过，覆盖新增 `TestLoad*` 和既有 `TestValidate*`。
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-current-baseline.ps1 -SkipHttpSmoke -R2Mode Skip` 通过，覆盖 API 契约校验、配置清单契约校验、配置语义契约校验、migration 契约校验、`go test ./...`、`go build -buildvcs=false ./...` 和 `go run ./cmd/migrate up`。

阶段 21 遗留限制：

- 本阶段不验证真实 R2 上传。
- 本阶段不做错误消息全文快照。
- 本阶段不配置真实 R2 凭据。
- 本阶段不进入新的 feed、vote、moderation、notification 或 search 产品语义。

阶段 21 review packet：

1. 完成能力：新增配置加载运行时契约测试，覆盖 local 默认值、R2 成功加载、R2 缺凭据失败和基础解析失败。
2. 修改文件：`internal/platform/config/load_test.go`、`README.md`、`tasks.md`、`docs/internal/README.md`、`docs/internal/engineering/workflow.md`、`docs/contracts/configuration.md`、`docs/internal/architecture/community-v1.md`、`.ai/slices/stage-21-config-load-contract/*`。
3. 新增或修改接口：无业务接口变更。
4. 完整调用链：TestLoad* -> clear config env -> temp cwd -> set required env -> Load -> loadDotEnvIfPresent -> default/parse helpers -> validate。
5. 错误码映射：业务错误码无变化；测试覆盖配置加载错误文本中包含对应配置项名称。
6. 测试覆盖和未覆盖内容：`go test ./internal/platform/config -v` 已覆盖配置加载运行时契约；快速基线已覆盖 API 契约、配置清单契约、配置语义契约、migration 契约、测试、构建和 migration；未覆盖真实 R2 上传和错误消息全文快照；本阶段未重跑 Stage 13/14 HTTP smoke。
7. 绕过的非主链路能力：真实 R2 上传、错误消息全文快照、真实 R2 凭据配置、新产品语义。
8. 下一阶段建议：运行快速基线；提供 R2 dev bucket 凭据后运行 `.\scripts\verify-current-baseline.ps1 -R2Mode Require`；也可以继续补配置错误消息快照。

## 阶段 20 工单

### T20-001：配置语义契约校验和文档收口

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 19 已完成`
- 目标：让配置文档中的必需性、默认值和枚举说明可以被脚本校验，避免 R2/local 等配置语义在代码和文档之间漂移。
- 交付物：
  - `scripts/verify-config-semantics-doc.ps1`
  - `scripts/verify-current-baseline.ps1`
  - `docs/contracts/configuration.md`
  - `docs/internal/architecture/media-storage.md`
  - `README.md`
  - `tasks.md`
  - `docs/internal/README.md`
  - `docs/internal/engineering/workflow.md`
  - `docs/internal/architecture/community-v1.md`
  - `.ai/slices/stage-20-config-semantics/README.md`
  - `.ai/slices/stage-20-config-semantics/01-config-semantics-check.md`
- 完成标准：
  - 文档缺少配置加载代码中的变量时失败。
  - 文档出现配置加载代码未加载的变量时失败。
  - 文档必需性与 `requiredString`、默认值和 R2/local 条件必需语义不一致时失败。
  - 文档默认值与 `load.go` 中的默认值不一致时失败。
  - 文档说明缺少 `validate.go` 中确认的枚举值时失败。
  - 不新增配置变量。
  - 不改变运行时配置语义。
  - 不写入真实 R2 凭据。
  - 不进入新的产品语义阶段。
- 结论：
  - `scripts/verify-config-semantics-doc.ps1` 已新增并通过。
  - `docs/contracts/configuration.md` 已补齐枚举说明和 `OBJECT_STORAGE_LOCAL_ROOT` 的默认值语义。
  - `docs/internal/architecture/media-storage.md` 已对齐 local fallback 默认值语义。
  - `scripts/verify-current-baseline.ps1` 已包含配置语义契约校验。
  - 本阶段未新增配置变量，未改变运行时配置语义。

## 阶段 20 复盘摘要

阶段 20 已完成：

- 配置语义契约校验脚本 `scripts/verify-config-semantics-doc.ps1`。
- 当前基线脚本已加入配置语义契约校验。
- 配置文档已补齐 `POSTGRES_SSL_MODE`、`LOG_LEVEL`、`LOG_FORMAT` 等枚举说明。
- `OBJECT_STORAGE_LOCAL_ROOT` 已明确为有默认值的非必需环境变量；最终配置仍不能为空。
- README、`tasks.md`、`docs/internal/README.md`、`docs/contracts/configuration.md`、`docs/internal/engineering/workflow.md`、`docs/internal/architecture/community-v1.md`、`docs/internal/architecture/media-storage.md` 和 `.ai/slices/stage-20-config-semantics/` 已同步。

阶段 20 验证结果：

- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-config-semantics-doc.ps1` 通过。
- 配置语义契约校验覆盖 33 个配置项和 5 个枚举配置项。
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-config-contract-doc.ps1` 通过，当前配置变量名清单仍为 33 个。
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-current-baseline.ps1 -SkipHttpSmoke -R2Mode Skip` 通过，覆盖 API 契约校验、配置清单契约校验、配置语义契约校验、migration 契约校验、`go test ./...`、`go build -buildvcs=false ./...` 和 `go run ./cmd/migrate up`。

阶段 20 遗留限制：

- 配置语义契约校验不覆盖完整数值范围、跨字段约束或错误消息文本。
- 本阶段不配置真实 R2 凭据。
- 本阶段不进入新的 feed、vote、moderation、notification 或 search 产品语义。

阶段 20 review packet：

1. 完成能力：新增配置语义契约校验，并纳入当前基线脚本。
2. 修改文件：`scripts/verify-config-semantics-doc.ps1`、`scripts/verify-current-baseline.ps1`、`docs/contracts/configuration.md`、`docs/internal/architecture/media-storage.md`、`README.md`、`tasks.md`、`docs/internal/README.md`、`docs/internal/engineering/workflow.md`、`docs/internal/architecture/community-v1.md`、`.ai/slices/stage-20-config-semantics/*`。
3. 新增或修改接口：无业务接口变更。
4. 完整调用链：verify-config-semantics-doc -> read `load.go` -> derive required/default semantics -> read `validate.go` -> derive enum/provider semantics -> read `configuration.md` -> compare documented semantics；baseline script -> API contract check -> config inventory check -> config semantic check -> migration contract check -> existing baseline checks。
5. 错误码映射：业务错误码无变化；脚本层发现配置文档语义漂移时失败。
6. 测试覆盖和未覆盖内容：配置必需性、默认值和枚举说明已校验；快速基线已覆盖 API 契约、配置清单契约、配置语义契约、migration 契约、测试、构建和 migration；未覆盖完整数值范围、跨字段约束或错误消息文本；本阶段未重跑 Stage 13/14 HTTP smoke。
7. 绕过的非主链路能力：数值范围自动校验、跨字段约束自动校验、错误消息快照、真实 R2 凭据配置、新产品语义。
8. 下一阶段建议：提供 R2 dev bucket 凭据后运行 `.\scripts\verify-current-baseline.ps1 -R2Mode Require`；也可以继续补配置数值范围/错误消息快照。

## 阶段 19 工单

### T19-001：migration 契约清单校验和文档收口

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 18 已完成`
- 目标：让 migration 文件和内部 migration 文档清单之间可以被脚本校验，避免后续 schema 变更出现编号、配对或文档漂移。
- 交付物：
  - `scripts/verify-migration-contract.ps1`
  - `scripts/verify-current-baseline.ps1`
  - `docs/contracts/migrations.md`
  - `README.md`
  - `tasks.md`
  - `docs/internal/README.md`
  - `docs/internal/engineering/workflow.md`
  - `docs/internal/architecture/community-v1.md`
  - `.ai/slices/stage-19-migration-contract/README.md`
  - `.ai/slices/stage-19-migration-contract/01-migration-contract-check.md`
- 完成标准：
  - `migrations/` 中出现不符合 `000001_name.up.sql` / `000001_name.down.sql` 格式的文件时失败。
  - migration 版本不是从 `000001` 起连续时失败。
  - 任一版本缺少 up 或 down 文件时失败。
  - 任一版本存在多个 up 或多个 down 文件时失败。
  - 同一版本 up/down 名称不一致时失败。
  - `docs/contracts/migrations.md` 中的版本/名称清单缺失、过期或不一致时失败。
  - 不新增或修改 schema migration。
  - 不改变 migration runner 行为。
  - 不进入新的产品语义阶段。
- 结论：
  - `scripts/verify-migration-contract.ps1` 已新增并通过。
  - `docs/contracts/migrations.md` 已新增并记录当前 9 个 migration。
  - `scripts/verify-current-baseline.ps1` 已包含 migration 契约校验。
  - 本阶段未新增或修改 SQL migration，未改变 migration runner 行为。

## 阶段 19 复盘摘要

阶段 19 已完成：

- migration 契约清单校验脚本 `scripts/verify-migration-contract.ps1`。
- migration 工程文档 `docs/contracts/migrations.md`。
- 当前基线脚本已加入 migration 契约清单校验。
- README、`tasks.md`、`docs/internal/README.md`、`docs/internal/engineering/workflow.md`、`docs/internal/architecture/community-v1.md` 和 `.ai/slices/stage-19-migration-contract/` 已同步。

阶段 19 验证结果：

- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-migration-contract.ps1` 通过。
- 当前 `migrations/` 中共有 9 个 migration，编号 `000001` 到 `000009` 连续，up/down 成对，文档清单一致。
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-current-baseline.ps1 -SkipHttpSmoke -R2Mode Skip` 通过，覆盖 API 契约校验、配置契约校验、migration 契约校验、`go test ./...`、`go build -buildvcs=false ./...` 和 `go run ./cmd/migrate up`。

阶段 19 遗留限制：

- migration 契约校验只覆盖文件命名、编号、配对和文档清单，不校验 SQL 语义。
- down migration 的实际可逆性仍需要人工审查。
- 本阶段不配置真实 R2 凭据。
- 本阶段不进入新的 feed、vote、moderation、notification 或 search 产品语义。

阶段 19 review packet：

1. 完成能力：新增 migration 契约清单校验，并纳入当前基线脚本。
2. 修改文件：`scripts/verify-migration-contract.ps1`、`scripts/verify-current-baseline.ps1`、`docs/contracts/migrations.md`、`README.md`、`tasks.md`、`docs/internal/README.md`、`docs/internal/engineering/workflow.md`、`docs/internal/architecture/community-v1.md`、`.ai/slices/stage-19-migration-contract/*`。
3. 新增或修改接口：无业务接口变更。
4. 完整调用链：verify-migration-contract -> read `migrations/` -> validate file naming/version pairing -> read `docs/contracts/migrations.md` -> compare version/name inventory；baseline script -> API contract check -> config contract check -> migration contract check -> existing baseline checks。
5. 错误码映射：业务错误码无变化；脚本层发现命名、配对、编号或文档清单漂移时失败。
6. 测试覆盖和未覆盖内容：migration 文件契约和文档清单已校验；快速基线已覆盖 API 契约、配置契约、migration 契约、测试、构建和 migration；未覆盖 SQL 语义和 down migration 可逆性；本阶段未重跑 Stage 13/14 HTTP smoke。
7. 绕过的非主链路能力：SQL 语义静态分析、down migration 可逆性自动校验、真实 R2 凭据配置、新产品语义。
8. 下一阶段建议：提供 R2 dev bucket 凭据后运行 `.\scripts\verify-current-baseline.ps1 -R2Mode Require`；后续新增 schema 时追加下一个 migration 版本并同步 `docs/contracts/migrations.md`；若要进入新的 feed、vote、moderation、notification 或 search 产品语义，需要先确认边界。

## 阶段 18 工单

### T18-001：配置契约清单校验和文档收口

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 17 已完成`
- 目标：让环境变量清单在配置加载代码、`.env.example` 和配置文档之间可以被脚本校验。
- 交付物：
  - `scripts/verify-config-contract-doc.ps1`
  - `scripts/verify-current-baseline.ps1`
  - `README.md`
  - `tasks.md`
  - `docs/internal/README.md`
  - `docs/contracts/configuration.md`
  - `docs/internal/engineering/workflow.md`
  - `docs/internal/architecture/community-v1.md`
  - `.ai/slices/stage-18-config-contract/README.md`
  - `.ai/slices/stage-18-config-contract/01-config-contract-check.md`
- 完成标准：
  - 脚本缺少文档、`.env.example` 或配置加载文件时失败。
  - 配置加载代码中出现但 `.env.example` 或配置文档缺失的变量时失败。
  - `.env.example` 或配置文档出现代码未加载的变量时失败。
  - 当前 33 个配置变量全部一致。
  - 不改变运行时配置加载语义。
  - 不进入新的产品语义阶段。
- 结论：
  - `scripts/verify-config-contract-doc.ps1` 已新增并通过。
  - `scripts/verify-current-baseline.ps1` 已包含配置契约校验。
  - `docs/contracts/configuration.md` 已写入校验命令和 local/R2 public base URL 边界。
  - 本阶段未新增配置变量，未改变运行时配置加载语义。

## 阶段 18 复盘摘要

阶段 18 已完成：

- 配置契约清单校验脚本 `scripts/verify-config-contract-doc.ps1`。
- 当前基线脚本已加入配置契约清单校验。
- 配置文档已补校验命令和校验边界。
- `OBJECT_STORAGE_PUBLIC_BASE_URL` 的文档说明已修正为：R2 必需；local 空值时由代码补 `http://localhost:8080/uploads`。
- README、`tasks.md`、`docs/internal/README.md`、`docs/contracts/configuration.md`、`docs/internal/engineering/workflow.md`、`docs/internal/architecture/community-v1.md` 和 `.ai/slices/stage-18-config-contract/` 已同步。

阶段 18 验证结果：

- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-config-contract-doc.ps1` 通过。
- 当前加载代码、`.env.example` 和配置文档一致，共 33 个环境变量。
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-current-baseline.ps1 -SkipHttpSmoke -R2Mode Skip` 通过，覆盖 API 契约校验、配置契约校验、`go test ./...`、`go build -buildvcs=false ./...` 和 `go run ./cmd/migrate up`。

阶段 18 遗留限制：

- 配置契约校验只覆盖环境变量名集合，不校验完整默认值和枚举语义。
- 本阶段不配置真实 R2 凭据。
- 本阶段不进入新的 feed、vote、moderation、notification 或 search 产品语义。

阶段 18 review packet：

1. 完成能力：新增配置契约清单校验，并纳入当前基线脚本。
2. 修改文件：`scripts/verify-config-contract-doc.ps1`、`scripts/verify-current-baseline.ps1`、`docs/contracts/configuration.md`、`README.md`、`tasks.md`、`docs/internal/README.md`、`docs/internal/engineering/workflow.md`、`docs/internal/architecture/community-v1.md`、`.ai/slices/stage-18-config-contract/*`。
3. 新增或修改接口：无业务接口变更。
4. 完整调用链：verify-config-contract-doc -> read `load.go` env keys -> read `.env.example` keys -> read configuration doc keys -> compare key sets；baseline script -> API contract check -> config contract check -> existing baseline checks。
5. 错误码映射：业务错误码无变化；脚本层发现缺失或过期配置项时失败。
6. 测试覆盖和未覆盖内容：配置变量名集合已校验；快速基线已覆盖 API 契约、配置契约、测试、构建和 migration；未覆盖完整默认值/枚举语义；本阶段未重跑 Stage 13/14 HTTP smoke。
7. 绕过的非主链路能力：真实 R2 凭据配置、配置 schema 生成、新配置变量、新产品语义。
8. 下一阶段建议：提供 R2 dev bucket 凭据后运行 `.\scripts\verify-current-baseline.ps1 -R2Mode Require`；若要进入新的 feed、vote、moderation、notification 或 search 产品语义，需要先确认边界。

## 阶段 17 工单

### T17-001：HTTP API 契约快照和路由清单校验

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 16 已完成`
- 目标：让当前 API 路由和认证边界有一份可校验的内部契约快照，供后续前后端协作和目标模式阶段切换参考。
- 交付物：
  - `docs/contracts/http-api-contract.md`
  - `scripts/verify-api-contract-doc.ps1`
  - `README.md`
  - `tasks.md`
  - `docs/internal/README.md`
  - `docs/internal/engineering/workflow.md`
  - `docs/internal/architecture/community-v1.md`
  - `.ai/slices/stage-17-api-contract-snapshot/README.md`
  - `.ai/slices/stage-17-api-contract-snapshot/01-http-api-contract.md`
- 完成标准：
  - 文档中的路由清单与源码 `RegisterRoutes` 和 local static route 保持一致。
  - 文档明确 public/auth/protected/local-only 边界。
  - 文档明确这是契约快照，不是完整 OpenAPI schema。
  - 校验脚本缺失路由或存在过期路由时失败。
  - 不改变任何业务接口、错误码或响应格式。
  - 不进入新的产品语义阶段。
- 结论：
  - `docs/contracts/http-api-contract.md` 已新增。
  - `scripts/verify-api-contract-doc.ps1` 已新增并通过。
  - `scripts/verify-current-baseline.ps1` 已包含 API 契约路由清单校验。
  - 本阶段未新增 Go 业务代码、schema migration、第三方依赖、业务接口、错误码或响应格式。

## 阶段 17 复盘摘要

阶段 17 已完成：

- 当前 HTTP API 契约快照 `docs/contracts/http-api-contract.md`。
- 路由清单同步校验脚本 `scripts/verify-api-contract-doc.ps1`。
- 当前基线脚本已加入 API 契约路由清单校验。
- README、`tasks.md`、`docs/internal/README.md`、`docs/internal/engineering/workflow.md`、`docs/internal/architecture/community-v1.md` 和 `.ai/slices/stage-17-api-contract-snapshot/` 已同步。

阶段 17 验证结果：

- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-api-contract-doc.ps1` 通过。
- 当前文档和源码注册路由一致，共 34 条 route。
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-current-baseline.ps1 -SkipHttpSmoke -R2Mode Skip` 通过，覆盖 API 契约校验、`go test ./...`、`go build -buildvcs=false ./...` 和 `go run ./cmd/migrate up`。

阶段 17 遗留限制：

- 契约快照只校验 HTTP method 和 path，不校验完整 request/response schema。
- 本阶段不生成 OpenAPI。
- 本阶段不进入新的 feed、vote、moderation、notification 或 search 产品语义。

阶段 17 review packet：

1. 完成能力：新增 HTTP API 契约快照，新增路由清单同步校验脚本，并将该校验纳入当前基线脚本。
2. 修改文件：`docs/contracts/http-api-contract.md`、`scripts/verify-api-contract-doc.ps1`、`scripts/verify-current-baseline.ps1`、`README.md`、`tasks.md`、`docs/internal/README.md`、`docs/internal/engineering/workflow.md`、`docs/internal/architecture/community-v1.md`、`.ai/slices/stage-17-api-contract-snapshot/*`。
3. 新增或修改接口：无业务接口变更。
4. 完整调用链：verify-api-contract-doc -> read route registration sources -> read contract doc route table -> compare method/path set；baseline script -> API contract check -> existing baseline checks。
5. 错误码映射：业务错误码无变化；脚本层发现缺失或过期路由时失败。
6. 测试覆盖和未覆盖内容：路由 method/path 清单已校验；快速基线已覆盖测试、构建和 migration；未覆盖完整 request/response schema；本阶段未重跑 Stage 13/14 HTTP smoke。
7. 绕过的非主链路能力：OpenAPI 生成、前端 SDK 生成、新业务接口、新产品语义。
8. 下一阶段建议：提供 R2 dev bucket 凭据后运行 `.\scripts\verify-current-baseline.ps1 -R2Mode Require`；若要进入新的 feed、vote、moderation、notification 或 search 产品语义，需要先确认边界。

## 阶段 16 工单

### T16-001：当前基线验收脚本和文档收口

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 15 已完成`
- 目标：提供一个当前基线验收脚本，作为后续目标模式长跑前后的统一检查入口。
- 交付物：
  - `scripts/verify-current-baseline.ps1`
  - `README.md`
  - `tasks.md`
  - `docs/internal/README.md`
  - `docs/internal/engineering/workflow.md`
  - `docs/internal/architecture/community-v1.md`
  - `.ai/slices/stage-16-baseline-verification/README.md`
  - `.ai/slices/stage-16-baseline-verification/01-current-baseline-script.md`
- 完成标准：
  - 脚本默认覆盖测试、构建、migration、Stage 13 smoke、Stage 14 smoke 和 Stage 15 R2 凭据门禁。
  - 脚本支持跳过 HTTP smoke，用于快速本地检查。
  - 脚本支持 R2 `Skip`、`SkipWhenMissing` 和 `Require` 三种模式。
  - 文档明确默认 `SkipWhenMissing` 模式在存在 R2 dev bucket 凭据时会执行真实上传并留下测试对象。
  - 文档明确当前没有 R2 凭据时不能声称真实 R2 上传已验证。
  - 不进入新的产品语义阶段，不改业务接口。
- 结论：
  - `scripts/verify-current-baseline.ps1` 已新增。
  - 默认命令已通过；Stage 15 R2 分支因当前缺凭据输出 skipped JSON。
  - 本阶段未新增 Go 业务代码、schema migration、第三方依赖或业务接口。

## 阶段 16 复盘摘要

阶段 16 已完成：

- 当前基线验收脚本 `scripts/verify-current-baseline.ps1`。
- 默认完整基线命令：
  - `go test ./...`
  - `go build -buildvcs=false ./...`
  - `go run ./cmd/migrate up`
  - `scripts/smoke-stage-13-content-system.ps1 -SkipMigration`
  - `scripts/smoke-stage-14-content-lifecycle.ps1 -SkipMigration`
  - `scripts/smoke-stage-15-r2-upload.ps1 -SkipMigration -SkipWhenMissingCredentials`
- 快速检查模式：`powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-current-baseline.ps1 -SkipHttpSmoke -R2Mode Skip`。
- R2 控制模式：`-R2Mode Skip|SkipWhenMissing|Require`。
- README、`tasks.md`、`docs/internal/README.md`、`docs/internal/engineering/workflow.md`、`docs/internal/architecture/community-v1.md` 和 `.ai/slices/stage-16-baseline-verification/` 已同步。

阶段 16 验证结果：

- 快速模式通过。
- 默认完整基线通过。
- Stage 13 smoke 通过：user `s13_smoke_260603094539297`，post `4872ac6d-e23f-4803-908a-08318b212778`，root comment `053c5405-a95a-4610-8a3b-f410f7e920e3`，child comment `c90300f6-4301-4c63-a27d-8dd57aac8e3c`。
- Stage 14 smoke 通过：author `s14_author_260603094711380`，intruder `s14_intruder_260603094711380`，post `18bf9639-d672-4d60-84cb-00a7bfc6b05d`，comment `5ddaa5c7-6320-45f3-8cc5-7d590df3f68f`。
- Stage 15 R2 分支输出 skipped JSON，缺少 `OBJECT_STORAGE_ENDPOINT`、`OBJECT_STORAGE_BUCKET`、`OBJECT_STORAGE_ACCESS_KEY_ID`、`OBJECT_STORAGE_SECRET_ACCESS_KEY`、`OBJECT_STORAGE_PUBLIC_BASE_URL`。

阶段 16 遗留限制：

- 当前本机未提供 R2 dev bucket 凭据，真实 R2 上传仍未验证。
- 默认 `SkipWhenMissing` 模式如果检测到 R2 dev bucket 凭据，会执行真实上传并留下测试对象。
- 本阶段不进入新的 feed、vote、moderation、notification 或 search 产品语义。

阶段 16 review packet：

1. 完成能力：新增当前基线验收入口，统一执行测试、构建、migration、Stage 13/14 HTTP smoke 和 Stage 15 R2 凭据门禁。
2. 修改文件：`scripts/verify-current-baseline.ps1`、`README.md`、`tasks.md`、`docs/internal/README.md`、`docs/internal/engineering/workflow.md`、`docs/internal/architecture/community-v1.md`、`.ai/slices/stage-16-baseline-verification/*`。
3. 新增或修改接口：无业务接口变更。
4. 完整调用链：baseline script -> Go tests/build/migration -> Stage 13 smoke -> Stage 14 smoke -> Stage 15 R2 smoke or credential gate。
5. 错误码映射：业务错误码无变化；脚本层任一必跑步骤失败则整体失败，默认 R2 缺凭据路径是 skipped JSON。
6. 测试覆盖和未覆盖内容：快速模式和默认完整基线已通过；真实 R2 上传未覆盖，因为当前没有 R2 dev bucket 凭据。
7. 绕过的非主链路能力：新产品语义、R2 对象清理、生产密钥配置、业务接口扩展。
8. 下一阶段建议：提供 R2 dev bucket 凭据后运行 `.\scripts\verify-current-baseline.ps1 -R2Mode Require`；若要进入新的 feed、vote、moderation、notification 或 search 产品语义，需要先确认边界。

## 阶段 15 工单

### T15-001：R2 smoke 工具、文档和凭据门禁

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 14 已完成`
- 目标：补齐真实 R2 dev bucket 上传验证入口，并明确没有凭据时不能声称 R2 已验证。
- 交付物：
  - `scripts/smoke-stage-15-r2-upload.ps1`
  - `README.md`
  - `tasks.md`
  - `docs/internal/README.md`
  - `docs/internal/architecture/media-storage.md`
  - `docs/internal/engineering/workflow.md`
  - `.ai/slices/stage-15-r2-verification/README.md`
  - `.ai/slices/stage-15-r2-verification/01-r2-smoke-tooling.md`
- 完成标准：
  - 脚本缺少 R2 凭据时默认失败；显式 `-SkipWhenMissingCredentials` 时输出 skipped JSON。
- 脚本真实运行时强制 `OBJECT_STORAGE_PROVIDER=r2`。
- 脚本真实运行时验证上传返回 URL 使用 `OBJECT_STORAGE_PUBLIC_BASE_URL`。
- 脚本真实运行时验证上传返回 URL 可被 HTTP 读取，返回 200 且 `Content-Type` 为 `image/*`。
- 脚本真实运行时验证 R2 attachment 可绑定到帖子。
  - 文档明确本阶段不做 R2 对象清理。
  - 本机没有 R2 凭据时，最终报告必须把真实上传标为未验证。
- 结论：
  - Stage 15 工具和文档门禁已完成。
  - 缺凭据 skipped 分支已通过；真实 R2 上传等待 R2 dev bucket 凭据后运行脚本验证。
  - 本阶段未新增 Go 业务代码、schema migration、第三方依赖、对象删除或 R2 清理任务。

## 阶段 15 复盘摘要

阶段 15 已完成：

- `scripts/smoke-stage-15-r2-upload.ps1`。
- 真实运行路径强制 `OBJECT_STORAGE_PROVIDER=r2`，并要求 R2 endpoint、bucket、access key、secret key、public base URL。
- smoke 成功路径会注册用户、上传 1x1 PNG、验证 attachment `ready`、验证 URL 使用 `OBJECT_STORAGE_PUBLIC_BASE_URL`、真实读取该 URL 返回 200 和 `image/*`，并创建绑定该 attachment 的 `public` 帖子。
- 缺少 R2 凭据时，默认运行失败；显式 `-SkipWhenMissingCredentials` 会输出 skipped JSON 并退出 0。
- README、`tasks.md`、`docs/internal/README.md`、`docs/internal/architecture/media-storage.md`、`docs/internal/engineering/workflow.md`、`docs/internal/architecture/community-v1.md` 和 `.ai/slices/stage-15-r2-verification/` 已同步。
- `go test ./...`、`go build -buildvcs=false ./...`、`go run ./cmd/migrate up` 通过。
- 不带 `-SkipWhenMissingCredentials` 且缺少 R2 凭据时，脚本预检按预期失败，未启动 API。
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\smoke-stage-15-r2-upload.ps1 -SkipWhenMissingCredentials` 通过，输出缺失 R2 凭据的 skipped JSON。

阶段 15 遗留限制：

- 当前本机未提供 R2 dev bucket 凭据，真实 R2 上传未验证。
- smoke 成功运行会在 dev bucket 留下测试对象；本阶段不实现 R2 对象清理。
- 不配置生产密钥，不新增对象删除能力，不把 local fallback 或 skipped 分支当作真实 R2 验证。

阶段 15 review packet：

1. 完成能力：新增真实 Cloudflare R2 dev bucket 上传 smoke 脚本；缺凭据时默认失败，显式 skip 时输出 skipped JSON；真实路径会验证图片上传、URL 前缀、公开 URL 可读性和帖子附件绑定。
2. 修改文件：`scripts/smoke-stage-15-r2-upload.ps1`、`README.md`、`tasks.md`、`docs/internal/README.md`、`docs/internal/architecture/media-storage.md`、`docs/internal/engineering/workflow.md`、`docs/internal/architecture/community-v1.md`、`.ai/slices/stage-15-r2-verification/*`。
3. 新增或修改接口：无业务接口变更；复用 `POST /api/v1/uploads/images` 和 `POST /api/v1/communities/public/posts` 做 smoke。
4. 完整调用链：smoke script -> temporary API with `OBJECT_STORAGE_PROVIDER=r2` -> media HTTP handler -> media usecase -> `internal/storage` R2 client -> Cloudflare R2 -> media repository -> post usecase binding。
5. 错误码映射：业务错误码无变化；脚本层缺少 R2 配置时默认失败，显式 `-SkipWhenMissingCredentials` 时输出 skipped JSON，不映射为 HTTP 业务错误。
6. 测试覆盖和未覆盖内容：`go test ./...`、`go build -buildvcs=false ./...`、`go run ./cmd/migrate up`、缺凭据默认失败预检和 skipped 分支已通过；真实 R2 上传未覆盖，因为当前本机无 R2 dev bucket 凭据。
7. 绕过的非主链路能力：生产密钥配置、R2 对象清理、对象物理删除、图片缩略图、图片处理流水线、前端上传 UI。
8. 下一阶段建议：不进入新产品语义前，先补一个当前基线验收入口，统一执行测试、构建、migration、Stage 13/14 smoke 和 Stage 15 R2 凭据门禁。

## 阶段 14 复盘摘要

阶段 14 已完成：

- `PATCH /api/v1/posts/:id`。
- `DELETE /api/v1/posts/:id`。
- `PATCH /api/v1/comments/:id`。
- `DELETE /api/v1/comments/:id`。
- 用户主动删除使用 `deleted`；平台审核移除继续使用 `removed`。
- `go test ./...`、`go build -buildvcs=false ./...`、`go run ./cmd/migrate up` 通过。
- `scripts/smoke-stage-14-content-lifecycle.ps1` 通过：user `s14_author_260603091420648`，intruder `s14_intruder_260603091420648`，post `b52d21de-fdc1-447c-902e-0ddb1918d6ba`，comment `eb017625-3f96-4dc4-8017-ba58ffdf0ecc`。

阶段 14 遗留限制：

- 不做编辑历史、草稿、恢复、附件重新绑定、R2 对象物理删除、搜索索引刷新、通知扩展或审核动作扩展。

## 阶段 14 工单

### T14-001：阶段 14 产品边界、架构文档与工单切换

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 13 已完成`
- 目标：定义作者编辑和软删除的最小闭环，明确状态、权限、接口和不做事项。
- 交付物：
  - `docs/internal/architecture/content-lifecycle.md`
  - `docs/internal/architecture/community-v1.md`
  - `docs/internal/engineering/workflow.md`
  - `README.md`
  - `docs/internal/README.md`
  - `tasks.md`
  - `.ai/slices/stage-14-content-lifecycle/README.md`
  - `.ai/slices/stage-14-content-lifecycle/01-stage-14-product-boundary.md`
- 完成标准：
  - 帖子和评论编辑/删除接口契约写清楚。
  - `deleted` 与 `removed` 的语义边界写清楚。
  - 附件/R2 对象不物理删除的边界写清楚。
  - 暂不实现事项写清楚。
- 结论：
  - 已新增 `docs/internal/architecture/content-lifecycle.md`。
  - 已同步 README、内部文档索引、`community-v1.md`、workflow 和 `.ai/slices/stage-14-content-lifecycle/`。
  - 阶段 14 不新增 schema migration，不引入新依赖。

### T14-002：帖子编辑和软删除

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T14-001`
- 目标：让作者可以编辑和软删除自己的 visible 帖子。
- 交付物：
  - `PATCH /api/v1/posts/:id`
  - `DELETE /api/v1/posts/:id`
  - post domain / usecase / repository / HTTP 测试
- 完成标准：
  - 未认证返回 `unauthenticated`。
  - 非作者返回 `forbidden`。
  - 非法 ID、title 或 body 返回 `invalid_argument`。
  - 不存在或非 visible 帖子返回 `not_found`。
  - 编辑后帖子详情、社区列表和全站列表读取新内容。
  - 删除后帖子详情、社区列表和全站列表不再返回该帖子。
- 结论：
  - `PATCH /api/v1/posts/:id` 已接入受保护路由。
  - `DELETE /api/v1/posts/:id` 已接入受保护路由。
  - post domain 新增编辑和标记 deleted 的状态转换。
  - post usecase 负责认证、visible 读取、作者校验和业务错误。
  - PostgreSQL repository 使用 `UPDATE ... WHERE status='visible'`，0 行更新映射为 `not_found`。
  - 已通过 `go test ./internal/post/...`。

### T14-003：评论编辑和软删除

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T14-002`
- 目标：让作者可以编辑和软删除自己的 visible 评论。
- 交付物：
  - `PATCH /api/v1/comments/:id`
  - `DELETE /api/v1/comments/:id`
  - comment domain / usecase / repository / HTTP 测试
- 完成标准：
  - 未认证返回 `unauthenticated`。
  - 非作者返回 `forbidden`。
  - 非法 ID 或 body 返回 `invalid_argument`。
  - 不存在或非 visible 评论返回 `not_found`。
  - 编辑后评论列表和 tree view 读取新正文。
  - 删除后评论列表和 tree view 不再返回该评论。
- 结论：
  - `PATCH /api/v1/comments/:id` 已接入受保护路由。
  - `DELETE /api/v1/comments/:id` 已接入受保护路由。
  - comment domain 新增编辑正文和标记 deleted 的状态转换。
  - comment usecase 负责认证、visible 评论和 visible 所属帖子确认、作者校验和业务错误。
  - PostgreSQL repository 使用 `UPDATE ... WHERE status='visible'`，0 行更新映射为 `not_found`。
  - 已通过 `go test ./internal/comment/...`。

### T14-004：阶段 14 冒烟、文档收口与 review packet

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T14-003`
- 目标：验证阶段 14 主链路并收口文档。
- 交付物：
  - `scripts/smoke-stage-14-content-lifecycle.ps1`
  - README、tasks、docs/internal、`.ai/slices` 收口
  - 阶段 14 review packet
- 完成标准：
  - `go test ./...` 通过。
  - `go build -buildvcs=false ./...` 通过。
  - `go test ./internal/post/... ./internal/comment/...` 通过。
  - `go run ./cmd/migrate up` 通过。
  - 真实 HTTP 冒烟覆盖帖子编辑、帖子删除、评论编辑、评论删除和非作者 forbidden 路径。
- 结论：
  - `scripts/smoke-stage-14-content-lifecycle.ps1` 已新增并通过。
  - smoke 覆盖作者编辑帖子、作者编辑评论、非作者编辑帖子 forbidden、非作者编辑评论 forbidden、非作者删除评论 forbidden、非作者删除帖子 forbidden、作者删除评论、作者删除帖子、删除后帖子详情 `not_found`。
  - 阶段 14 review packet 已输出。

## 阶段 13 复盘摘要

阶段 13 已完成：

- Reddit-style 评论树读取。
- 历史阶段 13 曾使用 `body_format=markdown` 帖子/评论响应契约；阶段 31 已替换为 `format=nexus_markdown`。
- Cloudflare R2 storage 配置和校验。
- `media_attachments` migration、domain、usecase、repository。
- `POST /api/v1/uploads/images` 图片上传接口。
- 发帖 `attachment_ids` 绑定与帖子读取 attachments。
- 评论 `attachment_ids` 绑定与评论 flat/tree 读取 attachments。
- `go test ./...`、`go build -buildvcs=false ./...`、`go run ./cmd/migrate up` 通过。
- `scripts/smoke-stage-13-content-system.ps1` 通过：图片上传、帖子图片绑定、根评论/子评论图片绑定、评论树读取和非法 `view` 失败路径。

阶段 13 遗留限制：

- 真实 R2 上传未使用生产凭据验证。
- 不做前端 UI、富文本 HTML 编辑器、任意 HTML、任意 iframe、embed 播放器、评论投票、通知扩展、搜索扩展、生产真实密钥配置或对象物理删除任务。

## 阶段 13 工单

### T13-001：阶段 13 产品边界、架构文档与工单切换

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 12 已完成`
- 目标：先写文档，确认评论树、Markdown-like 正文、图片附件和 R2 存储边界。
- 交付物：
  - `docs/internal/architecture/content-system.md`
  - `docs/internal/architecture/media-storage.md`
  - `tasks.md`
  - `README.md`
  - `docs/internal/README.md`
  - `docs/internal/engineering/workflow.md`
  - `.ai/slices/stage-13-content-system/README.md`
  - `.ai/slices/stage-13-content-system/01-stage-13-product-boundary.md`
- 完成标准：
  - 评论树接口契约写清楚。
  - Markdown-like 正文边界写清楚。
  - 图片附件数据模型写清楚。
  - Cloudflare R2 配置、上传流程和安全限制写清楚。
  - 暂不实现事项和后续 embed 预留方向写清楚。
- 结论：
  - 阶段 13 文档边界已确认。
  - 生产/主方案直接使用 Cloudflare R2，local filesystem 仅作为本地无凭据验证 fallback。
  - 新增 AWS SDK for Go v2 的用途、替代方案和影响范围已写入 `media-storage.md`。

### T13-002：评论树读取契约

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T13-001`
- 目标：让帖子评论读取支持 Reddit-style tree view。
- 交付物：
  - `GET /api/v1/posts/:id/comments?view=tree&sort=new&limit=20&offset=0&max_depth=6`
  - 评论 DTO 新增历史字段 `body_format`、`depth`、`reply_count`、`has_more_replies`；阶段 31 已将正文格式字段替换为 `format`
  - 根评论 `parent_id` 返回 `null`
  - comment usecase / repository / HTTP 测试
- 完成标准：
  - `view=tree` 返回扁平前序遍历数组。
  - 父评论总在子评论之前。
  - 根评论按 `sort=new` 排序，子评论在父评论下按同一规则排序。
  - `limit/offset` 作用于根评论，并在文档说明局限。
  - 非法 view/sort/max_depth 返回 `invalid_argument`。
- 结论：
  - `GET /api/v1/posts/:id/comments?view=tree&sort=new&max_depth=...` 已实现。
  - 根评论 `parent_id` 返回 `null`，响应当时包含 `body_format`、`depth`、`reply_count` 和 `has_more_replies`；阶段 31 已将正文格式字段替换为 `format`
  - 已通过 `go test ./internal/comment/...`。

### T13-003：Markdown-like 正文格式契约

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T13-002`
- 目标：帖子和评论响应明确正文格式，不渲染也不存储 HTML。
- 交付物：
  - posts/comments 响应当时新增 `body_format=markdown`；阶段 31 已替换为 `format=nexus_markdown`
  - 如需多格式持久化，再新增格式列和 check 约束
  - README 和架构文档同步
- 完成标准：
  - 创建接口继续保存用户原始正文。
  - 后端不渲染 HTML。
  - 后端不保存用户 HTML。
  - 文档明确禁止任意 HTML 和 iframe。
- 结论：
  - 帖子和评论响应当时固定返回 `body_format=markdown`；阶段 31 已替换为 `format=nexus_markdown`。
  - 本阶段未新增正文格式数据库列；未来多格式再引入 schema 约束。
  - 已通过 `go test ./internal/post/... ./internal/comment/...`。

### T13-004：R2 storage 配置和校验

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T13-003`
- 目标：新增 Cloudflare R2 对象存储配置边界。
- 交付物：
  - `internal/platform/config` storage/upload 配置
  - `.env.example`
  - README
  - config 测试
- 完成标准：
  - `OBJECT_STORAGE_PROVIDER=r2|local`。
  - `r2` 必须校验 endpoint、bucket、access key、secret key、public base URL。
  - `local` fallback 只用于本地开发。
  - 上传大小和数量配置必须大于 0。
- 结论：
  - 已新增 storage/upload 配置、`.env.example` 和配置校验测试。
  - 生产/主方案直接使用 Cloudflare R2；local 仅作为本地 fallback。
  - 已通过 `go test ./internal/platform/config/...`。

### T13-005：media attachment 数据模型

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T13-004`
- 目标：新增图片附件元数据事实表和业务边界。
- 交付物：
  - `media_attachments` migration
  - media domain / usecase / repository
  - repository 测试
- 完成标准：
  - 支持 `pending|ready|blocked|failed` 状态。
  - 支持 owner `none|post|comment`。
  - 支持校验 uploader、状态、数量和绑定目标。
  - repository 错误统一映射为业务错误。
- 结论：
  - 已新增 `media_attachments` migration、media domain/usecase/repository。
  - owner 支持 `none|post|comment`，图片 MIME 白名单为 JPEG/PNG/WebP。
  - 已通过 `go test ./internal/media/...`。

### T13-006：图片上传接口

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T13-005`
- 目标：新增登录用户图片上传能力。
- 交付物：
  - `POST /api/v1/uploads/images`
  - R2 storage client 和 local fallback
  - media HTTP handler / usecase / tests
  - `cmd/api/main.go` wiring
- 完成标准：
  - multipart/form-data 支持 `file` 和可选 `alt_text`。
  - 只允许 JPEG、PNG、WebP。
  - 文件大小受配置限制。
  - 上传成功保存 `media_attachments(status=ready)`。
  - 失败时不把未验证能力说成通过。
- 结论：
  - `POST /api/v1/uploads/images` 已接入受保护路由。
  - R2 client 封装在 `internal/storage`，handler/usecase 不依赖 AWS SDK。
  - 已通过 `go test ./cmd/api ./internal/media/... ./internal/storage/...`。

### T13-007：帖子图片绑定

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T13-006`
- 目标：发帖支持绑定已上传图片，帖子读取返回 attachments。
- 交付物：
  - `POST /api/v1/communities/:slug/posts` 请求支持 `attachment_ids`
  - 帖子详情、社区帖子列表、全站帖子流返回 attachments
  - usecase/repository/HTTP 测试
- 完成标准：
  - 附件必须属于当前用户。
  - 附件必须 `ready`。
  - 附件未绑定或已绑定当前帖子。
  - 单帖图片数量受配置限制。
- 结论：
  - 发帖请求已支持 `attachment_ids`。
  - 帖子详情、社区帖子列表和全站帖子流已返回 attachments。
  - 已通过 `go test ./internal/post/... ./internal/media/... ./cmd/api`。

### T13-008：评论图片绑定

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T13-007`
- 目标：评论支持绑定已上传图片，评论树返回 attachments。
- 交付物：
  - `POST /api/v1/posts/:id/comments` 请求支持 `attachment_ids`
  - 评论列表和 tree view 返回 attachments
  - usecase/repository/HTTP 测试
- 完成标准：
  - 根评论和子评论都可返回图片附件。
  - 单评论图片数量受配置限制。
  - 评论树响应顺序保持稳定。
- 结论：
  - 评论发布请求已支持 `attachment_ids`。
  - 评论 flat list 和 `view=tree` 均返回 attachments。
  - 已通过 `go test ./internal/comment/... ./internal/media/... ./cmd/api`。

### T13-009：阶段 13 收口验证和 review packet

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T13-008`
- 目标：验证阶段 13 主链路并收口文档。
- 交付物：
  - README、tasks、docs/internal、`.ai/slices` 收口
  - 真实冒烟记录
  - 阶段 13 review packet
- 完成标准：
  - `go test ./...` 通过。
  - `go build -buildvcs=false ./...` 通过。
  - `go test ./internal/comment/...` 通过。
  - `go test ./internal/post/...` 通过。
  - `go test ./internal/media/...` 通过。
  - `go test ./internal/platform/config/...` 通过。
  - `go run ./cmd/migrate up` 通过。
  - 真实冒烟覆盖评论树、图片上传、帖子图片绑定、评论图片绑定和关键失败路径。
- 结论：
  - `go test ./...` 通过。
  - `go build -buildvcs=false ./...` 通过。
  - `go run ./cmd/migrate up` 通过。
  - `scripts/smoke-stage-13-content-system.ps1` 通过：上传图片、发帖绑定图片、根评论绑定图片、子评论绑定图片、`view=tree` 读取 comment attachments、非法 `view` 返回 `invalid_argument`。
  - 真实 R2 上传未使用生产凭据验证；本阶段不配置生产真实密钥。
  - 阶段 13 主链路已完成。

## 阶段 12 工单

### T12-001：阶段 12 产品边界与文档切换

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 11 已完成`
- 目标：确认阶段 12 先做站内通知中心，不进入 WebSocket、邮件、推送或通知设置。
- 交付物：
  - `tasks.md`
  - `README.md`
  - `docs/internal/README.md`
  - `docs/internal/architecture/community-v1.md`
  - `.ai/slices/stage-12-notifications/README.md`
  - `.ai/slices/stage-12-notifications/01-stage-12-product-boundary.md`
- 完成标准：
  - 阶段 12 产品形态明确为站内通知中心。
  - 通知源生成暂不接入跨模块事件，本阶段先保证通知事实和读取/已读入口。
  - 不做 WebSocket、邮件、移动推送、通知设置、批量已读或删除通知。
- 结论：
  - 已切换到 `stage/12-notifications`。
  - 阶段 12 产品形态确认为站内通知中心。
  - 跨模块自动事件源暂缓，先保证通知事实和读取/已读入口。

### T12-002：通知 schema / usecase / repository / HTTP

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T12-001`
- 目标：新增站内通知事实和读取/已读接口。
- 交付物：
  - `notifications` migration
  - notification domain / usecase / repository
  - `GET /api/v1/notifications`
  - `POST /api/v1/notifications/:id/read`
  - HTTP handler 和测试
- 完成标准：
  - 列表默认返回当前用户 unread 通知。
  - `status=read|all` 可切换过滤。
  - 标记已读只允许当前用户自己的通知。
  - 重复标记已读幂等。
  - handler 不直接访问数据库。
- 结论：
  - `notifications` migration 已新增。
  - `GET /api/v1/notifications` 已接入受保护路由。
  - `POST /api/v1/notifications/:id/read` 已接入受保护路由。
  - repository 按 `recipient_id` 过滤，跨用户访问返回 `not_found`。

### T12-003：阶段 12 冒烟、文档收口与 review packet

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T12-002`
- 目标：验证阶段 12 主链路并收口文档。
- 交付物：
  - 阶段 12 真实冒烟记录。
  - README 和内部文档收口。
  - 阶段 12 review packet。
- 完成标准：
  - `go test ./...` 通过。
  - `go build -buildvcs=false ./...` 通过。
  - 真实冒烟覆盖列表、标记已读、status 过滤和非法输入。
  - 文档已直接读取确认。
  - 阶段 12 review packet 已输出。
- 结论：
  - `go test ./internal/notification/...` 通过。
  - `go test ./...` 通过。
  - `go build -buildvcs=false ./...` 通过。
  - 真实 HTTP 冒烟通过：user `s12_user_1780378362`，notification `eeca7b14-d053-42e2-9b53-c4e0e8748e3d`。

## 阶段 11 工单

### T11-001：阶段 11 产品边界与文档切换

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 10 已完成`
- 目标：确认阶段 11 先做 PostgreSQL 基础搜索，不引入全文索引或外部搜索引擎。
- 交付物：
  - `tasks.md`
  - `README.md`
  - `docs/internal/README.md`
  - `docs/internal/architecture/community-v1.md`
  - `.ai/slices/stage-11-search/README.md`
  - `.ai/slices/stage-11-search/01-stage-11-product-boundary.md`
- 完成标准：
  - 阶段 11 产品形态明确为 `GET /api/v1/search` 基础搜索。
  - 搜索范围限定为社区名称、帖子标题和帖子正文。
  - 后续目标顺序保持为：阶段 12 通知。
  - 不做全文索引、外部搜索引擎、标签搜索、评论搜索、高亮、搜索分析、个性化排序或通知。
- 结论：
  - 已切换到 `stage/11-search`。
  - 阶段 11 产品形态确认为 `GET /api/v1/search` 基础搜索。
  - 搜索范围限定为公开可读社区名称、帖子标题和帖子正文。

### T11-002：搜索 usecase / repository / HTTP

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T11-001`
- 目标：新增基础搜索接口。
- 交付物：
  - `GET /api/v1/search`
  - search usecase 参数校验和编排
  - PostgreSQL 搜索查询
  - HTTP handler 和测试
- 完成标准：
  - `q` 必填，trim 后为空或超过 `100` 返回 `invalid_argument`。
  - `scope` 默认 `all`，非法 scope 返回 `invalid_argument`。
  - `scope=communities` 只返回 communities。
  - `scope=posts` 只返回 posts。
  - `scope=all` 同时返回 communities 和 posts。
  - 搜索结果只包含公开可读社区和 visible 帖子。
  - handler 不直接访问数据库。
- 结论：
  - `GET /api/v1/search` 已接入受保护路由。
  - search usecase 负责认证、query、scope 和分页校验。
  - PostgreSQL repository 使用 `ILIKE` 搜索 active public communities 和 visible posts。
  - handler 只处理认证上下文、query 参数和响应映射。

### T11-003：阶段 11 冒烟、文档收口与 review packet

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T11-002`
- 目标：验证阶段 11 主链路并收口文档。
- 交付物：
  - 阶段 11 真实冒烟记录。
  - README 和内部文档收口。
  - 阶段 11 review packet。
- 完成标准：
  - `go test ./...` 通过。
  - `go build -buildvcs=false ./...` 通过。
  - 真实冒烟覆盖社区名称、帖子标题、帖子正文、scope 和非法输入。
  - 文档已直接读取确认。
  - 阶段 11 review packet 已输出。
- 结论：
  - `go test ./internal/search/...` 通过。
  - `go test ./...` 通过。
  - `go build -buildvcs=false ./...` 通过。
  - 真实 HTTP 冒烟通过：community `s11-search-1780377530`，post `25cc3944-3fea-42be-8cf9-3538163df792`。

## 阶段 10 工单

### T10-001：阶段 10 产品边界与文档切换

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 9 已完成`
- 目标：确认阶段 10 先做审核台 target 内容预览增强，不进入搜索或通知实现。
- 交付物：
  - `tasks.md`
  - `README.md`
  - `docs/internal/README.md`
  - `docs/internal/architecture/community-v1.md`
  - `.ai/slices/stage-10-moderation-console-enhancement/README.md`
  - `.ai/slices/stage-10-moderation-console-enhancement/01-stage-10-product-boundary.md`
- 完成标准：
  - 阶段 10 产品形态明确为审核台 `target_preview` 增强。
  - 不新增数据库表，不改变阶段 8 权限和事务边界。
  - 后续目标顺序保持为：阶段 11 搜索、阶段 12 通知。
  - 不做审核后台 UI、社区 moderator、批量处理、自动审核、申诉、搜索或通知。
- 结论：
  - 已切换到 `stage/10-moderation-console-enhancement`。
  - 阶段 10 产品形态确认为审核台 `target_preview` 增强。
  - 不新增数据库表，不改变阶段 8 权限和事务边界。

### T10-002：举报列表和详情 target preview

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T10-001`
- 目标：让平台 staff 查看举报列表和详情时直接看到被举报目标预览。
- 交付物：
  - `GET /api/v1/moderation/reports` 响应新增 `target_preview`
  - `GET /api/v1/moderation/reports/:id` 响应新增 `target_preview`
  - moderation console usecase DTO
  - PostgreSQL 查询 join posts/comments
  - HTTP handler 和测试
- 完成标准：
  - post preview 包含 `target_type`、`post_id`、`author_id`、`status`、`title`、`body_excerpt`、`created_at`、`updated_at`。
  - comment preview 包含 `target_type`、`comment_id`、`post_id`、`author_id`、`status`、`body_excerpt`、`created_at`、`updated_at`。
  - preview 读取目标当前状态，可显示 `removed`。
  - 孤立举报不导致列表整体失败。
  - handler 不直接访问数据库。
- 结论：
  - `GET /api/v1/moderation/reports` 响应已新增 `target_preview`。
  - `GET /api/v1/moderation/reports/:id` 响应已新增 `target_preview`。
  - PostgreSQL repository 使用 `LEFT JOIN posts/comments` 读取目标当前状态和摘要。
  - usecase 通过读取 DTO 传递预览；domain 不持有预览字段。

### T10-003：阶段 10 冒烟、文档收口与 review packet

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T10-002`
- 目标：验证阶段 10 主链路并收口文档。
- 交付物：
  - 阶段 10 真实冒烟记录。
  - README 和内部文档收口。
  - 阶段 10 review packet。
- 完成标准：
  - `go test ./...` 通过。
  - `go build -buildvcs=false ./...` 通过。
  - 真实冒烟覆盖列表 preview、详情 preview、post/comment target 和 removed 状态。
  - 文档已直接读取确认。
  - 阶段 10 review packet 已输出。
- 结论：
  - `go test ./internal/moderation/...` 通过。
  - `go test ./...` 通过。
  - `go build -buildvcs=false ./...` 通过。
  - 真实 HTTP 冒烟通过：staff `s10_staff_1780376484`，post report `607e7a4e-576e-4b3b-9c12-cad7b701fce4`，comment report `8dad6346-c003-4d55-b89d-0cf6724d18c9`。

## 阶段 9 工单

### T9-001：阶段 9 产品边界与文档切换

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 8 已完成`
- 目标：确认阶段 9 先做 hot feed / 内容分发最小闭环，不进入审核台增强、搜索或通知实现。
- 交付物：
  - `tasks.md`
  - `README.md`
  - `docs/internal/README.md`
  - `docs/internal/architecture/community-v1.md`
  - `.ai/slices/stage-09-content-distribution/README.md`
  - `.ai/slices/stage-09-content-distribution/01-stage-9-product-boundary.md`
- 完成标准：
  - 阶段 9 产品形态明确为 `sort=new|hot` 的帖子流排序能力。
  - 默认 `new` 契约保持兼容。
  - hot 排序只基于现有投票事实和创建时间，不引入推荐系统。
  - 后续目标顺序记录为：阶段 10 审核台增强、阶段 11 搜索、阶段 12 通知。
  - 不做个性化推荐、预计算时间线、推荐系统、反作弊、评论投票、通知或审核台增强。
- 结论：
  - 已切换到 `stage/9-content-distribution`。
  - 阶段 9 产品形态确认为 `sort=new|hot` 的帖子流排序能力。
  - 后续目标顺序记录为：阶段 10 审核台增强、阶段 11 搜索、阶段 12 通知。

### T9-002：帖子流 hot 排序 usecase / repository / HTTP

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T9-001`
- 目标：让全站帖子流和社区帖子列表支持 `sort=hot`。
- 交付物：
  - `GET /api/v1/posts?sort=hot`
  - `GET /api/v1/communities/:slug/posts?sort=hot`
  - post usecase 排序参数与校验
  - PostgreSQL 查询排序
  - HTTP handler 和测试
- 完成标准：
  - 不传 `sort` 时仍按 `created_at DESC, id DESC`。
  - `sort=new` 等价于默认排序。
  - `sort=hot` 按 `score DESC, upvote_count DESC, created_at DESC, id DESC`。
  - 非法 `sort` 返回 `invalid_argument`。
  - 响应继续包含投票统计和当前用户投票视角。
  - handler 不计算热度，不直接访问数据库。
- 结论：
  - `GET /api/v1/posts` 已支持 `sort=new|hot`。
  - `GET /api/v1/communities/:slug/posts` 已支持 `sort=new|hot`。
  - `sort=hot` 由 PostgreSQL repository 按 `score DESC, upvote_count DESC, created_at DESC, id DESC` 排序。
  - usecase 负责排序参数归一化和 `invalid_argument` 映射，handler 只透传 query。

### T9-003：阶段 9 冒烟、文档收口与 review packet

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T9-002`
- 目标：验证阶段 9 主链路并收口文档。
- 交付物：
  - 阶段 9 真实冒烟记录。
  - README 和内部文档收口。
  - 阶段 9 review packet。
- 完成标准：
  - `go test ./...` 通过。
  - `go build -buildvcs=false ./...` 通过。
  - 真实冒烟覆盖全站 hot、社区 hot、默认 new 和非法 sort。
  - 文档已直接读取确认。
  - 阶段 9 review packet 已输出。
- 结论：
  - `go test ./...` 通过。
  - `go build -buildvcs=false ./...` 通过。
  - 真实 HTTP 冒烟通过：作者 `s9_author_1780375514`，hot 帖 `9cbe0470-3d0d-449d-b782-a2a9dae4b554`，warm 帖 `89996fbd-a2d4-411a-ac89-f94c500f7411`，cold 帖 `f2f529e7-1580-44f0-bbb4-3ece9524410d`。

## 阶段 8 工单

### T8-001：阶段 8 产品边界与文档切换

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 7 已完成`
- 目标：确认阶段 8 先做审核台最小闭环，不进入审核后台 UI、社区 moderator、通知、防刷、自动审核、申诉或 hot feed。
- 交付物：
  - `tasks.md`
  - `README.md`
  - `docs/internal/README.md`
  - `docs/internal/architecture/community-v1.md`
  - `.ai/slices/stage-08-moderation-console/README.md`
  - `.ai/slices/stage-08-moderation-console/01-stage-8-product-boundary.md`
- 完成标准：
  - 阶段 8 产品形态明确为平台 staff 的举报处理入口。
  - dismiss 举报最小留痕使用 `content_reports.status/reviewed_by/reviewed_at/updated_at`。
  - remove-target 复用阶段 7 的内容移除事务。
  - 不扩展 `moderation_actions.action` 枚举。
  - 不做审核后台 UI、社区 moderator、通知、防刷、自动审核、申诉或 hot feed。
- 结论：
  - 阶段 8 目标已确认：举报列表、举报详情、dismiss、基于举报移除目标内容。
  - 已切换到 `stage/8-moderation-console`。
  - 长期边界已沉淀到 `docs/internal/architecture/community-v1.md`。

### T8-002：举报列表与举报详情

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T8-001`
- 目标：让平台 staff 可以查看举报列表和单个举报详情。
- 交付物：
  - `GET /api/v1/moderation/reports`
  - `GET /api/v1/moderation/reports/:id`
  - moderation console usecase / repository query
  - HTTP handler 和测试
- 完成标准：
  - 未登录返回 `unauthenticated`。
  - 非 staff 返回 `forbidden`。
  - 列表默认只返回 `pending` 举报。
  - 支持 `status`、`limit`、`offset` 查询参数。
  - `limit` 默认 `20`，最大 `50`。
  - 详情不存在返回 `not_found`。
  - handler 不直接访问数据库。
- 结论：
  - `GET /api/v1/moderation/reports` 已接入受保护路由。
  - `GET /api/v1/moderation/reports/:id` 已接入受保护路由。
  - `ConsoleUseCase` 会先检查 `users.is_platform_staff`，非 staff 返回 `forbidden`。
  - 列表 status 默认 `pending`，分页默认 `20`，最大 `50`。
  - PostgreSQL repository 已支持按 status 查询举报列表和按 ID 查询详情。
  - HTTP handler 只处理认证上下文、query/path 参数和响应映射。
  - `go test ./internal/moderation/...`、`go test ./...` 和 `go build -buildvcs=false ./...` 已通过。

### T8-003：dismiss 举报

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T8-002`
- 目标：让平台 staff 可以 dismiss 不需要移除内容的 pending 举报。
- 交付物：
  - `POST /api/v1/moderation/reports/:id/dismiss`
  - dismiss usecase / repository
  - HTTP handler 和测试
- 完成标准：
  - 未登录返回 `unauthenticated`。
  - 非 staff 返回 `forbidden`。
  - 举报不存在返回 `not_found`。
  - 非 `pending` 举报返回 `conflict`。
  - 成功后 `content_reports.status=dismissed`，写入 `reviewed_by`、`reviewed_at`、`updated_at`。
  - 不新增 `moderation_actions.action=dismiss`。
- 结论：
  - `POST /api/v1/moderation/reports/:id/dismiss` 已接入受保护路由。
  - `ConsoleUseCase.DismissReport` 会先确认 staff、举报存在且状态为 `pending`。
  - PostgreSQL repository 使用 `UPDATE ... WHERE status='pending' RETURNING ...` 写入 `dismissed`、`reviewed_by`、`reviewed_at`、`updated_at`。
  - 非 pending 举报返回 `conflict`。
  - dismiss 不新增 `moderation_actions.action=dismiss`。
  - `go test ./internal/moderation/...`、`go test ./...` 和 `go build -buildvcs=false ./...` 已通过。

### T8-004：基于举报移除目标内容

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T8-003`
- 目标：让平台 staff 可以从举报详情直接移除被举报的帖子或评论。
- 交付物：
  - `POST /api/v1/moderation/reports/:id/remove-target`
  - remove reported target usecase / repository
  - HTTP handler 和测试
- 完成标准：
  - 未登录返回 `unauthenticated`。
  - 非 staff 返回 `forbidden`。
  - 举报不存在返回 `not_found`。
  - 非 `pending` 举报返回 `conflict`。
  - reason 为空返回 `invalid_argument`。
  - 移除内容、写入审核动作、解决 pending 举报仍在同一事务内完成。
  - 移除后现有 visible 读取接口不再返回目标内容。
- 结论：
  - `POST /api/v1/moderation/reports/:id/remove-target` 已接入受保护路由。
  - `ConsoleUseCase.RemoveReportedTarget` 会先确认 staff、校验举报 ID 和 reason、确认举报存在且为 `pending`。
  - PostgreSQL repository 在同一事务内锁定举报行、校验 pending、移除目标内容、写入 `moderation_actions`、将同 target 的 pending 举报标记为 `resolved`。
  - 非 pending 举报返回 `conflict`，目标不存在或已不可见返回 `not_found`。
  - repository、usecase 和 HTTP 测试已覆盖 remove-target 主链路。
  - `go test ./internal/moderation/...` 已通过。

### T8-005：阶段 8 冒烟、文档收口与 review packet

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T8-004`
- 目标：验证阶段 8 主链路并收口文档。
- 交付物：
  - 阶段 8 真实冒烟记录。
  - README 和内部文档收口。
  - 阶段 8 review packet。
- 完成标准：
  - `go test ./...` 通过。
  - `go build -buildvcs=false ./...` 通过。
  - 真实冒烟覆盖举报列表、举报详情、dismiss、remove-target 和移除后读取不可见。
  - 文档已直接读取确认。
  - 阶段 8 review packet 已输出。
- 结论：
  - `go test ./...` 已通过。
  - `go build -buildvcs=false ./...` 已通过。
  - 真实本地冒烟已通过：注册普通用户、注册 staff 用户、SQL 设置 `is_platform_staff=true`、公共总版发帖、发布评论、举报帖子、举报评论、staff 列表查看、详情查看、dismiss 举报、基于举报移除评论、验证移除后评论列表不可见。
  - 冒烟结果包含 `UnauthListCode=unauthenticated`、`NonStaffListCode=forbidden`、`PendingListDefaultLimit=20`、`PendingListDefaultOffset=0`、`DismissedReportStatus=dismissed`、`RemoveTargetType=comment`、`CommentListContainsRemovedComment=False`。
  - README、`.ai/slices/stage-08-moderation-console/`、`docs/internal/README.md` 和 `docs/internal/architecture/community-v1.md` 已同步阶段 8 收口结论。

## 阶段 8 复盘摘要

阶段 8 已完成：

- 平台 staff 可以查看举报列表，默认 `pending`，分页默认 `20`，最大 `50`，按 `created_at DESC`。
- 平台 staff 可以查看单个举报详情。
- 平台 staff 可以 dismiss `pending` 举报，并写入 `dismissed`、`reviewed_by`、`reviewed_at` 和 `updated_at`。
- 平台 staff 可以基于 `pending` 举报移除目标帖子或评论。
- remove-target 复用阶段 7 内容移除事务，移除内容、写入 `moderation_actions(action=remove)`、解决 pending 举报在同一 PostgreSQL 事务内完成。
- 未登录访问审核台接口返回 `unauthenticated`。
- 非 staff 访问审核台接口返回 `forbidden`。
- `go test ./...` 通过。
- `go build -buildvcs=false ./...` 通过。
- 真实本地冒烟通过。

阶段 8 遗留限制：

- 不做审核后台 UI。
- 不做社区 moderator 权限。
- 不做通知。
- 不做防刷策略。
- 不做自动审核。
- 不做申诉流程。
- 不做批量处理。
- 不做 target 内容预览增强。
- 不做 hot feed。

阶段 8 review packet：

1. 完成能力：举报列表、举报详情、dismiss 举报、基于举报 remove-target、staff 权限校验、真实冒烟和文档收口。
2. 修改文件：moderation usecase/repository/http、`cmd/api`、README、`tasks.md`、`.ai/slices/stage-08-moderation-console/`、内部文档索引和长期架构文档。
3. 新增或修改接口：新增 `GET /api/v1/moderation/reports`、`GET /api/v1/moderation/reports/:id`、`POST /api/v1/moderation/reports/:id/dismiss`、`POST /api/v1/moderation/reports/:id/remove-target`。
4. 完整调用链：HTTP handler -> auth middleware -> moderation console usecase -> platform staff repository / moderation repository -> PostgreSQL；remove-target 继续进入 moderation repository 事务，移除目标内容、写审核动作、解决 pending 举报。
5. 错误码映射：`unauthenticated`、`forbidden`、`invalid_argument`、`not_found`、`conflict`、`internal`。
6. 测试覆盖：usecase、repository、HTTP、`go test ./...`、`go build -buildvcs=false ./...`、真实 HTTP 冒烟；未覆盖 CI 冒烟、并发批量处理、审核台 UI、社区 moderator、通知、防刷、自动审核和申诉。
7. 本阶段绕过：审核后台 UI、社区 moderator 权限、通知、防刷、自动审核、申诉、批量处理、target 内容预览增强、hot feed。
8. 下一阶段建议：暂停在阶段 8 里程碑；阶段 9 需要先选择产品边界，候选为 hot feed / 内容分发、审核台增强、搜索或通知。

## 阶段 7 工单

### T7-001：阶段 7 产品边界与文档切换

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 6 已完成`
- 目标：确认阶段 7 先做轻量举报与平台 staff 移除内容闭环，不进入审核台、通知、防刷或 hot feed。
- 交付物：
  - `tasks.md`
  - `docs/internal/architecture/community-v1.md`
  - `.ai/slices/stage-07-moderation/README.md`
  - `.ai/slices/stage-07-moderation/01-stage-7-product-boundary.md`
- 完成标准：
  - 阶段 7 产品形态明确为举报 + staff 移除。
  - 内容移除使用既有 `removed` 内容状态。
  - 审核动作必须留痕。
  - 不做审核台列表、社区 moderator、通知、防刷和申诉。

### T7-002：举报与审核 schema / domain

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T7-001`
- 目标：建立举报事实和审核动作事实。
- 交付物：
  - `migrations/000007_create_moderation_schema.*.sql`
  - `internal/moderation/moderationdomain`
  - domain 测试
- 完成标准：
  - 举报支持 post/comment 两种 target。
  - 举报状态至少支持 `pending`、`resolved`、`dismissed`。
  - 审核动作至少支持 `remove`。
  - reason 必须非空。
  - 同一用户对同一 target 只能有一个 pending 举报。
  - schema 不做 destructive migration。
- 结论：
  - `migrations/000007_create_moderation_schema.*.sql` 已新增。
  - `content_reports` 支持 post/comment target，并保留外键约束。
  - `moderation_actions` 支持 `remove` 审核动作。
  - domain 和 domain 测试已新增。
  - `go test ./internal/moderation/...` 已通过。

### T7-003：举报提交 usecase / repository / HTTP

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T7-002`
- 目标：让已登录用户可以举报 visible 帖子和 visible 评论。
- 交付物：
  - `POST /api/v1/posts/:id/reports`
  - `POST /api/v1/comments/:id/reports`
  - moderation repository
  - moderation usecase
  - HTTP handler 和测试
- 完成标准：
  - 未登录返回 `unauthenticated`。
  - 非法 target id 返回 `invalid_argument`。
  - target 不存在或不可见返回 `not_found`。
  - reason 为空返回 `invalid_argument`。
  - 重复 pending 举报返回 `conflict`。
  - handler 不直接访问数据库。
- 结论：
  - `POST /api/v1/posts/:id/reports` 已接入。
  - `POST /api/v1/comments/:id/reports` 已接入。
  - usecase 依赖 post/comment repository interface 来确认 target visible。
  - repository 测试覆盖举报创建、评论举报和重复 pending 举报冲突。
  - HTTP 测试覆盖成功、未登录、非法请求和 usecase 错误传播。

### T7-004：staff 移除帖子和评论

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T7-003`
- 目标：让平台 staff 可以移除 visible 帖子和 visible 评论，并写入审核动作。
- 交付物：
  - `POST /api/v1/posts/:id/moderation/remove`
  - `POST /api/v1/comments/:id/moderation/remove`
  - remove usecase / repository 事务
  - HTTP handler 和测试
- 完成标准：
  - 未登录返回 `unauthenticated`。
  - 非 staff 返回 `forbidden`。
  - target 不存在或不可见返回 `not_found`。
  - 移除内容和审核动作写入必须在同一数据库事务中完成。
  - 移除后现有 visible 读取接口不再返回该内容。
- 结论：
  - `POST /api/v1/posts/:id/moderation/remove` 已接入受保护路由。
  - `POST /api/v1/comments/:id/moderation/remove` 已接入受保护路由。
  - usecase 通过 `users.is_platform_staff` 判断平台 staff，非 staff 返回 `forbidden`。
  - PostgreSQL repository 在同一事务内完成内容 `status=removed`、写入 `moderation_actions`、将 pending 举报标记为 `resolved`。
  - target 不存在或非 `visible` 时返回 `not_found`。
  - usecase、repository 和 HTTP 测试已覆盖移除主链路。
  - `go test ./internal/moderation/...`、`go test ./...` 和 `go build -buildvcs=false ./...` 已通过。

### T7-005：阶段 7 冒烟、文档收口与 review packet

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T7-004`
- 目标：验证阶段 7 主链路并收口文档。
- 交付物：
  - 阶段 7 真实冒烟记录。
  - README 和内部文档收口。
  - 阶段 7 review packet。
- 完成标准：
  - `go test ./...` 通过。
  - `go build -buildvcs=false ./...` 通过。
  - 真实冒烟覆盖举报和移除闭环。
  - 文档已直接读取确认。
  - 阶段 7 review packet 已输出。
- 结论：
  - `go run ./cmd/migrate up` 已执行。
  - 真实本地冒烟已通过：注册普通用户、注册 staff 用户、SQL 设置 `is_platform_staff=true`、公共总版发帖、发评论、举报帖子、举报评论、staff 移除评论、staff 移除帖子、验证移除后 visible 读取不可见。
  - 冒烟结果包含 `RemovedPostDetailStatus=404`、`CommentListContainsRemovedComment=False`、`CommunityListContainsRemovedPost=False`、`LatestListContainsRemovedPost=False`。
  - `go test ./...` 和 `go build -buildvcs=false ./...` 已通过。
  - README、`.ai/slices/stage-07-moderation/`、`docs/internal/README.md` 和 `docs/internal/architecture/community-v1.md` 已同步阶段 7 收口结论。

## 阶段 7 复盘摘要

阶段 7 已完成：

- `content_reports` 举报事实表、`moderation_actions` 审核动作事实表、moderation domain 和 repository。
- 已登录用户可以举报 visible 帖子。
- 已登录用户可以举报 visible 评论。
- 平台 staff 可以移除 visible 帖子。
- 平台 staff 可以移除 visible 评论。
- 移除内容和审核动作写入在同一 PostgreSQL 事务内完成。
- 同一 target 的 pending 举报会在移除时标记为 `resolved`。
- 被移除的帖子不再出现在帖子详情、社区帖子列表和全站最新流中。
- 被移除的评论不再出现在帖子评论列表中。
- `go test ./...` 通过。
- `go build -buildvcs=false ./...` 通过。
- 真实本地冒烟通过。

阶段 7 遗留限制：

- 不做审核台列表。
- 不做社区 moderator 权限。
- 不做通知。
- 不做防刷策略。
- 不做自动审核。
- 不做申诉流程。
- 不做 hot feed。

阶段 7 review packet：

1. 完成能力：帖子/评论举报、平台 staff 移除帖子/评论、审核动作留痕、pending 举报自动 resolved、真实冒烟和文档收口。
2. 修改文件：moderation domain/usecase/repository/http、`cmd/api`、migration、README、`tasks.md`、`.ai/slices/stage-07-moderation/`、内部文档索引。
3. 新增或修改接口：新增 `POST /api/v1/posts/:id/reports`、`POST /api/v1/comments/:id/reports`、`POST /api/v1/posts/:id/moderation/remove`、`POST /api/v1/comments/:id/moderation/remove`。
4. 完整调用链：HTTP handler -> auth middleware -> moderation usecase -> post/comment visible repository 或 platform staff repository -> moderation repository -> PostgreSQL。
5. 错误码映射：`unauthenticated`、`forbidden`、`invalid_argument`、`not_found`、`conflict`、`internal`。
6. 测试覆盖：domain、usecase、repository、HTTP、`go test ./...`、`go build -buildvcs=false ./...`、真实 HTTP 冒烟；未覆盖 CI 冒烟、并发压力、审核台列表、社区 moderator、通知、防刷、自动审核和申诉。
7. 本阶段绕过：审核台列表、社区 moderator 权限、通知、防刷、自动审核、申诉、hot feed。
8. 下一阶段建议：暂停在阶段 7 里程碑；下一阶段先讨论 hot feed、审核台增强、搜索或通知的产品边界。

## 阶段 6 复盘摘要

阶段 6 已完成：

- 已登录用户可以对 visible 帖子 upvote、downvote、切换投票和取消投票。
- 同一用户对同一帖子只有一个有效投票状态。
- 社区帖子列表、帖子详情、全站最新帖子流返回 `upvote_count`、`downvote_count`、`score`、`my_vote`。
- 全站最新帖子流只返回 visible 帖子，默认按 `created_at DESC`，分页仍使用 `limit/offset`，默认 `20`，最大 `50`。
- 投票事实独立建模，不把投票只做成帖子表上的计数字段。
- 不把 vote/downvote 直接接入 hot 排序；阶段 6 只做投票状态和读取视角。
- `go test ./...` 通过。
- `go build -buildvcs=false ./...` 通过。
- 真实冒烟覆盖：发帖、投票、切换投票、取消投票、社区帖子列表投票统计、帖子详情投票统计、全站最新流。
- README、`tasks.md`、`.ai/slices/stage-06-feed-vote/` 和长期架构文档已直接读取确认。
- 阶段 6 review packet 已输出。

## 阶段 6 工单

### T6-001：阶段 6 产品边界与文档切换

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 5 已完成`
- 目标：确认阶段 6 不再绕过 vote/downvote 和全站最新流，并把工单、架构文档和 `.ai/slices/` 切换到阶段 6。
- 交付物：
  - `tasks.md`
  - `README.md`
  - `docs/internal/README.md`
  - `docs/internal/architecture/community-v1.md`
  - `docs/internal/engineering/workflow.md`
  - `.ai/slices/stage-06-feed-vote/README.md`
  - `.ai/slices/stage-06-feed-vote/01-stage-6-product-boundary.md`
- 完成标准：
  - 阶段 6 产品形态明确为全站最新流 + 帖子 upvote/downvote。
  - 明确 `score = upvote_count - downvote_count`。
  - 明确 `my_vote = 1 | -1 | 0`。
  - 明确本阶段不做 hot feed、评论投票、审核台、通知和防刷策略。
  - 新阶段分支为 `stage/6-feed-vote`。
- 结论：
  - 用户已明确要求阶段 6 直接支持 vote/downvote，不先退化为 only like。
  - 阶段 6 先做投票事实、当前用户投票视角和全站最新流，不做 hot 排序。

### T6-002：帖子投票 schema 与领域模型

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T6-001`
- 目标：建立帖子投票事实表和领域对象。
- 交付物：
  - `migrations/000006_create_post_votes.*.sql`
  - `internal/vote/votedomain`
  - domain 测试
- 完成标准：
  - `post_votes` 记录用户对帖子的当前投票状态。
  - 投票值只允许 `1` 和 `-1`。
  - 取消投票通过删除记录表达。
  - 同一用户同一帖子只能有一条记录。
- 结论：
  - `migrations/000006_create_post_votes.*.sql` 已新增。
  - `post_votes` 使用 `(post_id, user_id)` 复合主键。
  - `post_votes.value` 使用 check 约束限制为 `-1` 或 `1`。
  - `votedomain.PostVote` 和 `VoteValue` 已落地，domain 不依赖 Gin、pgx、SQL 或 JWT。
  - `go test ./internal/vote/votedomain`、`go test ./...` 和 `go build -buildvcs=false ./...` 已通过。

### T6-003：帖子投票 repository

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T6-002`
- 目标：提供帖子投票写入、删除、读取和统计查询边界。
- 交付物：
  - `internal/vote/voteusecase/repository.go`
  - `internal/vote/voterepository`
  - repository 测试
- 完成标准：
  - 支持 upsert 用户投票。
  - 支持幂等删除用户投票。
  - 支持查询当前用户对单帖或多帖的投票。
  - 支持查询单帖或多帖的 upvote/downvote 统计。
  - 数据库错误统一映射为 `internal/apperr`。
- 结论：
  - `voteusecase.PostVoteRepository` 已定义在业务侧。
  - `PostgresPostVoteRepository` 已支持 upsert、幂等删除、单帖当前用户投票、多帖当前用户投票和多帖投票统计。
  - repository 只处理 `post_votes` 持久化和数据库错误映射，不判断帖子是否 visible。
  - repository 测试覆盖创建、切换、统计、删除幂等和外键缺失错误映射。
  - `go test ./internal/vote/...`、`go test ./...` 和 `go build -buildvcs=false ./...` 已通过。

### T6-004：帖子投票 usecase 与 HTTP 接口

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T6-003`
- 目标：让已登录用户可以对 visible 帖子 upvote、downvote、切换投票和取消投票。
- 交付物：
  - `PUT /api/v1/posts/:id/vote`
  - `DELETE /api/v1/posts/:id/vote`
  - vote usecase
  - HTTP handler 和测试
- 完成标准：
  - 未登录返回 `unauthenticated`。
  - 帖子不存在或不可见返回 `not_found`。
  - `value` 不是 `1` 或 `-1` 返回 `invalid_argument`。
  - 重复投同一值幂等成功。
  - upvote/downvote 可相互切换。
  - 重复取消投票幂等成功。
- 结论：
  - `PUT /api/v1/posts/:id/vote` 已接入受保护路由。
  - `DELETE /api/v1/posts/:id/vote` 已接入受保护路由。
  - vote usecase 会先确认帖子 visible，再 upsert 或删除投票。
  - handler 只解析 JSON、当前用户和 path param，不直接访问数据库。
  - usecase 和 HTTP 测试已覆盖成功、未登录、非法参数、帖子不存在和取消投票。
  - `go test ./internal/vote/...`、`go test ./...` 和 `go build -buildvcs=false ./...` 已通过。

### T6-005：帖子读取返回投票统计和当前用户视角

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T6-004`
- 目标：让社区帖子列表和帖子详情返回投票统计与当前用户投票状态。
- 交付物：
  - `GET /api/v1/communities/:slug/posts` 响应增加投票字段。
  - `GET /api/v1/posts/:id` 响应增加投票字段。
  - post 读取 usecase / query 边界调整。
  - HTTP 和 usecase 测试。
- 完成标准：
  - 每个帖子返回 `upvote_count`、`downvote_count`、`score`、`my_vote`。
  - `score = upvote_count - downvote_count`。
  - 未投票用户 `my_vote = 0`。
  - 不改变错误响应格式和认证错误语义。
- 结论：
  - 社区帖子列表和帖子详情已返回 `upvote_count`、`downvote_count`、`score`、`my_vote`。
  - post usecase 负责聚合 vote summary 和当前用户投票。
  - post handler 只读取当前用户并传入 usecase，不直接查询 vote。
  - 缺失投票统计时补零，未投票用户 `my_vote=0`。
  - 顺手补充了 HTTP CORS 基础配置：`HTTP_CORS_ALLOWED_ORIGINS`，默认空值不启用。
  - `go test ./internal/post/...`、`go test ./internal/vote/...` 和 `go test ./internal/platform/...` 已通过。

### T6-006：全站最新帖子流

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T6-005`
- 目标：提供全站最新帖子流，不把公共总版伪装成全站 Feed。
- 交付物：
  - `GET /api/v1/posts`
  - feed 或 post query 边界
  - HTTP、usecase/query 测试
- 完成标准：
  - 只返回 visible 帖子。
  - 只返回可公开读取社区内的帖子。
  - 默认按 `created_at DESC` 排序。
  - 分页使用 `limit/offset`，默认 `20`，最大 `50`。
  - 返回投票统计和 `my_vote`。
- 结论：
  - `GET /api/v1/posts` 已接入受保护路由。
  - post repository 新增 `ListVisibleInPublicCommunities`，通过 `posts` join `communities`，只读取 `posts.status='visible'` 且 `communities.status='active'`、`communities.visibility='public'` 的帖子。
  - post usecase 复用阶段 6 的 vote summary 聚合，返回 `upvote_count`、`downvote_count`、`score`、`my_vote`。
  - HTTP handler 只负责读取当前用户、分页参数和响应映射，不直接访问数据库或计算业务规则。
  - repository、usecase 和 HTTP 测试已覆盖全站最新流。

### T6-007：阶段 6 冒烟、文档收口与 review packet

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T6-006`
- 目标：验证阶段 6 主链路并收口文档。
- 交付物：
  - 阶段 6 真实冒烟记录。
  - README 和内部文档收口。
  - 阶段 6 review packet。
- 完成标准：
  - `go test ./...` 通过。
  - `go build -buildvcs=false ./...` 通过。
  - 真实冒烟覆盖 upvote、downvote、切换、取消、帖子读取投票视角、全站最新流。
  - 文档已直接读取确认。
  - 阶段 6 review packet 已输出。
- 结论：
  - `go test ./...` 已通过。
  - `go build -buildvcs=false ./...` 已通过。
  - `go run ./cmd/migrate up` 已执行到最新 migration。
  - 真实本地冒烟通过，临时端口 `:18086`，冒烟用户 `smoke_vote_260602022623491`，冒烟帖子 `2b74a80d-6d57-4d61-a3ae-db5e006766b6`。
  - 冒烟覆盖：注册、公共总版发帖、upvote、帖子详情投票统计、社区帖子列表投票统计、downvote 切换、全站最新流投票视角、取消投票、CORS preflight。
  - README、`tasks.md`、`.ai/slices/stage-06-feed-vote/`、`docs/internal/README.md`、`docs/internal/architecture/community-v1.md`、`docs/contracts/configuration.md` 已直接读取确认。

## 阶段 6 复盘摘要

阶段 6 已完成：

- `post_votes` 投票事实表、vote domain、vote repository 和 vote usecase。
- `PUT /api/v1/posts/:id/vote` 和 `DELETE /api/v1/posts/:id/vote`。
- 社区帖子列表、帖子详情和全站最新流返回 `upvote_count`、`downvote_count`、`score`、`my_vote`。
- `GET /api/v1/posts` 全站最新帖子流，读取 active public 社区中的 visible 帖子，按 `created_at DESC`。
- HTTP CORS 基础配置：`HTTP_CORS_ALLOWED_ORIGINS`。
- `go test ./...` 通过。
- `go build -buildvcs=false ./...` 通过。
- 真实本地冒烟通过。

阶段 6 遗留限制：

- 不做 hot feed、推荐排序或个性化首页。
- 不做评论投票。
- 不做审核台联动、通知或防刷策略。
- 不做投票计数物化和大 offset 性能优化。

阶段 6 review packet：

1. 完成能力：帖子 upvote/downvote/切换/取消、帖子读取投票视角、全站最新帖子流、CORS 基础配置、真实冒烟和文档收口。
2. 修改文件：vote domain/usecase/repository/http、post usecase/repository/http、`cmd/api`、migration、platform config/httpserver、README、`tasks.md`、`.ai/slices/stage-06-feed-vote/`、内部架构与配置文档。
3. 新增或修改接口：新增 `PUT /api/v1/posts/:id/vote`、`DELETE /api/v1/posts/:id/vote`、`GET /api/v1/posts`；社区帖子列表和帖子详情响应新增投票字段。
4. 完整调用链：HTTP handler -> auth middleware -> usecase -> repository interface -> PostgreSQL implementation；post usecase 聚合 vote summary 和当前用户投票。
5. 错误码映射：`unauthenticated`、`invalid_argument`、`not_found`、`conflict`、`internal`；认证失败仍统一为 `unauthenticated`。
6. 测试覆盖：domain、usecase、repository、HTTP、CORS、`go test ./...`、`go build -buildvcs=false ./...`、真实 HTTP 冒烟；未覆盖 CI 环境真实冒烟和大 offset 性能。
7. 本阶段绕过：hot feed、推荐排序、评论投票、审核台联动、通知、防刷策略、投票计数物化。
8. 下一阶段建议：先讨论阶段 7 产品边界，再决定进入 hot feed、审核台、搜索或通知。

## 阶段 5 复盘摘要

阶段 5 已完成：

- 完整真实冒烟覆盖阶段 2 到阶段 4 主链路。
- `go test ./...` 通过。
- `go build -buildvcs=false ./...` 通过。
- README、`tasks.md`、`docs/internal/README.md`、`docs/internal/architecture/community-v1.md`、`docs/internal/engineering/workflow.md` 和 `.ai/slices/` 已收口并直接读取确认。

阶段 5 review packet：

1. 完成能力：完整真实冒烟、文档收口、最终 review packet。
2. 修改文件：README、`tasks.md`、内部文档和 `.ai/slices/stage-05-acceptance/`。
3. 接口：验证既有社区申请、帖子、评论接口；未新增接口。
4. 调用链：注册/登录 -> 社区申请审批 -> 发帖 -> 帖子读取 -> 评论发布 -> 评论读取。
5. 错误码：`unauthenticated`、`invalid_argument`、`forbidden`、`not_found`、`conflict`、`internal`。
6. 测试覆盖：`go test ./...`、`go build -buildvcs=false ./...`、完整真实冒烟。
7. 绕过能力：vote、feed、moderation、搜索、图片上传、通知、私信等后续阶段。
8. 下一阶段：阶段 6 内容分发与投票基础。

## 阶段 5 工单

### T5-001：完整主链路真实冒烟与最终收口

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 4 已完成`
- 目标：验证阶段 2-4 主链路，收口所有项目文档并输出最终 review packet。
- 交付物：
  - 完整真实冒烟结果。
  - 最终 README 和内部文档收口。
  - 阶段 5 review packet。
- 完成标准：
  - `go test ./...` 通过。
  - `go build -buildvcs=false ./...` 通过。
  - 完整真实冒烟覆盖社区申请审批、发帖、帖子读取、评论发布和评论读取。
  - 文档已直接读取确认。
  - 最终 review packet 已输出。
- 注意点：
  - 阶段 5 不进入 feed、vote、moderation 等新产品语义阶段。
  - 非主链路能力继续记录为后续阶段。
- 结论：
  - 完整真实冒烟已通过。
  - `go test ./...` 和 `go build -buildvcs=false ./...` 已通过。
  - 最终 review packet 已输出。

## 阶段 4 复盘摘要

阶段 4 已完成：

- `comments` 表、comment domain、comment repository 和 usecase repository contract。
- 已登录用户可以对 visible 帖子发布评论。
- `parent_id` 可为空或指向同一帖子下的 visible 父评论。
- 已登录用户可以读取帖子评论扁平列表。
- 列表使用 `limit/offset`，默认 `20`，最大 `50`，按 `created_at DESC`。
- 列表只返回 `visible` 评论。
- `go test ./...` 通过。
- `go build -buildvcs=false ./...` 通过。
- 真实本地冒烟已覆盖评论发布和帖子评论列表。

阶段 4 遗留限制：

- 不做评论树优化。
- 不做评论编辑、删除。
- 不做 vote / like / dislike。
- 不做审核台和通知。

阶段 4 review packet：

1. 完成能力：评论 schema/domain/repository、评论发布、帖子评论列表。
2. 修改文件：comment domain/usecase/repository/http、`cmd/api`、migration、测试、README、`tasks.md`、`.ai/slices/stage-04-comment/`、`docs/internal/architecture/community-v1.md`。
3. 接口：新增 `POST /api/v1/posts/:id/comments`、`GET /api/v1/posts/:id/comments`。
4. 调用链：comment HTTP handler -> comment usecase -> post repository / comment repository -> PostgreSQL。
5. 错误码：`unauthenticated`、`invalid_argument`、`not_found`、`conflict`。
6. 测试覆盖：domain、usecase、HTTP、repository；未覆盖评论树优化和非 visible 审核状态流转。
7. 绕过能力：评论树优化、评论编辑、删除、投票、审核台、通知。
8. 下一阶段：阶段 5 完整真实冒烟、文档收口、最终 review packet。

## 阶段 4 工单

### T4-001：评论 schema、领域模型与仓储边界

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 3 已完成`
- 目标：建立评论主链路的最小写入和读取数据边界。
- 交付物：
  - `migrations/000005_create_comments.*.sql`
  - `internal/comment/commentdomain`
  - `internal/comment/commentusecase/repository.go`
  - `internal/comment/commentrepository`
  - domain 和 repository 测试
- 完成标准：
  - `comments.post_id` 关联 `posts.id`。
  - `comments.author_id` 关联 `users.id`。
  - `comments.parent_id` 可为空或关联 `comments.id`。
  - `comments.status` 至少支持 `visible`。
  - body 校验进入 domain。
  - repository 支持创建评论、按帖子分页列出 visible 评论。
  - repository 错误统一映射为 `internal/apperr`。
- 注意点：
  - 评论读取先返回扁平列表。
  - 不做评论树优化、投票和审核台。
- 结论：
  - `comments` schema 已落地到 `migrations/000005_create_comments.*.sql`。
  - `commentdomain` 已覆盖评论 ID、正文、状态、父评论和发布时间不变量。
  - comment usecase 侧 repository contract 已落地。
  - PostgreSQL repository 已支持创建评论、按 ID 查询 visible 评论、按帖子分页查询 visible 评论。
  - repository 集成测试已覆盖创建、详情、帖子评论列表和外键缺失错误映射。

### T4-002：评论发布接口

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T4-001`
- 目标：让已登录用户可以对 visible 帖子发表评论。
- 交付物：
  - `POST /api/v1/posts/:id/comments`
  - 评论发布 usecase
  - HTTP handler 和测试
- 完成标准：
  - 未登录返回 `unauthenticated`。
  - 帖子不存在或不可见返回 `not_found`。
  - 无效 body 返回 `invalid_argument`。
  - `parent_id` 为空或指向同一帖子下的父评论。
  - 成功发布后评论状态为 `visible`。
- 注意点：
  - 不做评论树优化。
  - 不做评论编辑、删除和投票。
- 结论：
  - `POST /api/v1/posts/:id/comments` 已接入受保护路由。
  - 评论发布前先确认帖子 visible。
  - `parent_id` 为空时发布顶层评论；不为空时必须指向同一帖子下的 visible 父评论。
  - 成功发布的评论状态为 `visible`。
  - handler 只做请求解析、当前用户读取和响应映射。

### T4-003：帖子评论列表

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T4-002`
- 目标：让已登录用户可以读取某个帖子的评论扁平列表。
- 交付物：
  - `GET /api/v1/posts/:id/comments`
  - 评论读取 usecase
  - HTTP handler 和测试
- 完成标准：
  - 帖子不存在或不可见返回 `not_found`。
  - 列表只返回 `visible` 评论。
  - 列表使用 `limit/offset`，默认 `20`，最大 `50`。
  - 默认按 `created_at DESC` 排序。
- 注意点：
  - 暂不返回树形结构。
  - 暂不返回投票状态。
- 结论：
  - `GET /api/v1/posts/:id/comments` 已接入受保护路由。
  - 列表先确认帖子 visible，再按帖子 ID 查询 visible 评论。
  - 列表分页由 usecase 收口：默认 `20`，最大 `50`，负数返回 `invalid_argument`。
  - 读取先返回扁平列表。

### T4-004：阶段 4 冒烟检查与文档收口

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T4-003`
- 目标：验证评论发布和帖子评论列表闭环，并收口阶段 4 文档。
- 交付物：
  - 阶段 4 本地冒烟步骤。
  - README 当前状态更新。
  - 阶段 4 遗留限制记录。
  - 阶段 5 收口建议。
- 完成标准：
  - `go test ./...` 通过。
  - `go build -buildvcs=false ./...` 通过。
  - 本地真实冒烟覆盖评论发布和帖子评论列表。
  - 阶段 4 review packet 已输出。
- 结论：
  - `go test ./...` 已通过。
  - `go build -buildvcs=false ./...` 已通过。
  - 真实本地冒烟已通过：注册用户、发布帖子、发布评论、帖子评论列表包含新评论。
  - 冒烟结果包含 `PublishedCommentStatus=visible`、`PublishedCommentID=3d96a5fe-761a-4405-b5db-e7676c451578`、`ListContainsComment=True`。
  - README、`.ai/slices/stage-04-comment/` 和 `docs/internal/architecture/community-v1.md` 已同步阶段 4 收口结论。

## 阶段 3 复盘摘要

阶段 3 已完成：

- `posts` 表、post domain、post repository 和 usecase repository contract。
- 已登录用户可以在 `active + public` 社区发布帖子。
- 已登录用户可以读取社区帖子列表。
- 已登录用户可以读取帖子详情。
- 列表使用 `limit/offset`，默认 `20`，最大 `50`，按 `created_at DESC`。
- 列表和详情只返回 `visible` 帖子。
- `go test ./...` 通过。
- `go build -buildvcs=false ./...` 通过。
- 真实本地冒烟已覆盖发布帖子、社区帖子列表和帖子详情。

阶段 3 遗留限制：

- 不做图片上传、标签、编辑、删除、草稿。
- 不做 vote / like / dislike。
- 不做 hot feed / 全站 feed。
- 不做搜索和审核台。

阶段 3 review packet：

1. 完成能力：帖子 schema/domain/repository、帖子发布、社区帖子列表、帖子详情。
2. 修改文件：post domain/usecase/repository/http、community `CanPostInCommunity`、`cmd/api`、migration、测试、README、`tasks.md`、`.ai/slices/stage-03-post/`、`docs/internal/architecture/community-v1.md`。
3. 接口：新增 `POST /api/v1/communities/:slug/posts`、`GET /api/v1/communities/:slug/posts`、`GET /api/v1/posts/:id`。
4. 调用链：post HTTP handler -> post usecase -> community usecase / post repository -> PostgreSQL。
5. 错误码：`unauthenticated`、`invalid_argument`、`forbidden`、`not_found`、`conflict`。
6. 测试覆盖：domain、usecase、HTTP、repository；未覆盖大 offset 性能和非 visible 审核状态流转。
7. 绕过能力：图片上传、标签、编辑、删除、草稿、vote、hot feed、全站 feed、搜索、审核台。
8. 下一阶段：阶段 4 评论发布、帖子评论列表。

## 阶段 3 工单

### T3-001：帖子 schema、领域模型与仓储边界

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`阶段 2 已完成`
- 目标：建立帖子主链路的最小写入和读取数据边界。
- 交付物：
  - `migrations/000004_create_posts.*.sql`
  - `internal/post/postdomain`
  - `internal/post/postusecase/repository.go`
  - `internal/post/postrepository`
  - domain 和 repository 测试
- 完成标准：
  - `posts.community_id` 关联 `communities.id`。
  - `posts.author_id` 关联 `users.id`。
  - `posts.status` 至少支持 `visible`。
  - title/body 校验进入 domain。
  - repository 支持创建帖子、按 ID 查询 visible 帖子、按社区分页列出 visible 帖子。
  - repository 错误统一映射为 `internal/apperr`。
- 注意点：
  - 不做帖子编辑、删除、草稿。
  - 不做 vote、hot feed 或审核台。
- 结论：
  - `posts` schema 已落地到 `migrations/000004_create_posts.*.sql`。
  - `postdomain` 已覆盖帖子 ID、标题、正文、状态和发布时间不变量。
  - post usecase 侧 repository contract 已落地。
  - PostgreSQL repository 已支持创建帖子、按 ID 查询 visible 帖子、按社区分页查询 visible 帖子。
  - repository 集成测试已覆盖创建、详情、社区列表和外键缺失错误映射。

### T3-002：帖子发布接口

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T3-001`
- 目标：让已登录用户可以在 `active + public` 社区发布帖子。
- 交付物：
  - `POST /api/v1/communities/:slug/posts`
  - 发布帖子 usecase
  - community 发帖权限判断能力
  - HTTP handler 和测试
- 完成标准：
  - 未登录返回 `unauthenticated`。
  - 社区不存在或不可公开读取返回 `not_found`。
  - 社区非 `active + public` 时不能发帖。
  - 无效 title/body 返回 `invalid_argument`。
  - 成功发布后帖子状态为 `visible`。
  - handler 不直接访问数据库，不承载发帖权限判断。
- 注意点：
  - 发帖权限先按 `active + public + logged in` 判断。
  - 不做图片上传、标签、草稿、编辑和删除。
- 结论：
  - `POST /api/v1/communities/:slug/posts` 已接入受保护路由。
  - 发帖 usecase 先解析社区 slug，再调用 community usecase 的 `CanPostInCommunity`。
  - `CanPostInCommunity` 使用 `active + public + logged in` 最小策略。
  - 成功发布的帖子状态为 `visible`。
  - handler 只做请求解析、当前用户读取和响应映射。

### T3-003：社区帖子列表与帖子详情

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T3-002`
- 目标：让已登录用户可以读取社区帖子列表和帖子详情。
- 交付物：
  - `GET /api/v1/communities/:slug/posts`
  - `GET /api/v1/posts/:id`
  - 帖子读取 usecase
  - HTTP handler 和测试
- 完成标准：
  - 列表只返回目标社区内 `visible` 帖子。
  - 列表使用 `limit/offset`，默认 `20`，最大 `50`。
  - 默认按 `created_at DESC` 排序。
  - 帖子详情只返回 `visible` 帖子。
  - 社区不存在或不可公开读取返回 `not_found`。
  - 帖子不存在返回 `not_found`。
- 注意点：
  - 不做全站 feed。
  - 不做 hot 排序。
  - 不做搜索。
- 结论：
  - `GET /api/v1/communities/:slug/posts` 已接入受保护路由。
  - `GET /api/v1/posts/:id` 已接入受保护路由。
  - 列表先确认社区可公开读取，再按社区 ID 查询 visible 帖子。
  - 列表分页由 usecase 收口：默认 `20`，最大 `50`，负数返回 `invalid_argument`。
  - 详情只返回 visible 帖子。

### T3-004：阶段 3 冒烟检查与文档收口

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T3-003`
- 目标：验证帖子发布、社区帖子列表和帖子详情闭环，并收口阶段 3 文档。
- 交付物：
  - 阶段 3 本地冒烟步骤。
  - README 当前状态更新。
  - 阶段 3 遗留限制记录。
  - 阶段 4 工单切换建议。
- 完成标准：
  - `go test ./...` 通过。
  - `go build -buildvcs=false ./...` 通过。
  - 本地真实冒烟覆盖发布帖子、社区帖子列表和帖子详情。
  - 阶段 3 review packet 已输出。
- 结论：
  - `go test ./...` 已通过。
  - `go build -buildvcs=false ./...` 已通过。
  - 真实本地冒烟已通过：注册用户、在 `public` 社区发布帖子、社区帖子列表包含新帖、帖子详情可读取。
  - 冒烟结果包含 `PublishedPostStatus=visible`、`PublishedPostID=805fe8c9-10e4-46a5-9c7c-13141996c97d`、`ListContainsPost=True`。
  - README、`.ai/slices/stage-03-post/` 和 `docs/internal/architecture/community-v1.md` 已同步阶段 3 收口结论。

## 阶段 2 复盘摘要

阶段 2 已完成：

- 系统内置公共总版存在，并且建模为 `Community(kind=system, slug=public)`。
- 社区、社区成员关系、社区申请具备 schema、领域模型和仓储边界。
- 已登录用户可以查看社区列表和社区详情。
- 已登录用户可以提交社区申请。
- 平台审批者可以批准或拒绝社区申请。
- 批准申请会在同一事务里创建社区，并把申请人设为该社区 `owner`。
- `go test ./...` 通过。
- `go build -buildvcs=false ./...` 通过。
- 真实本地冒烟已覆盖公共总版、社区读取、申请提交、批准和拒绝。

阶段 2 遗留限制：

- 不做 staff 管理接口，本地 demo 通过 SQL 设置 `users.is_platform_staff=true`。
- 不做申请列表、申请取消、后台审核台。
- 不做社区头像、背景图、分类目录。
- 不做私密社区、邀请制和复杂成员加入/退出。
- 不做帖子、评论、投票、feed 和审核台。

阶段 2 review packet：

1. 完成能力：公共总版、社区读取、社区申请提交、平台审批批准/拒绝、审批通过事务内创建社区和 owner。
2. 修改文件：community usecase/repository/http、`cmd/api`、repository/usecase/http 测试、README、`tasks.md`、`.ai/slices/stage-02-community/`、`docs/internal/architecture/community-v1.md`。
3. 接口：新增 `POST /api/v1/community-applications`、`POST /api/v1/community-applications/:id/approve`、`POST /api/v1/community-applications/:id/reject`。
4. 调用链：HTTP handler -> community application usecase -> staff repository / transaction manager -> application/community/membership repository -> PostgreSQL。
5. 错误码：`unauthenticated`、`invalid_argument`、`forbidden`、`not_found`、`conflict`。
6. 测试覆盖：usecase、HTTP、repository 事务成功和回滚；未覆盖并发双审批压力测试。
7. 绕过能力：staff 管理接口、审核台、feed、vote、搜索、图片上传、私信、私密社区、复杂成员加入/退出。
8. 下一阶段：阶段 3 帖子发布、社区帖子列表、帖子详情。

## 阶段 2 工单

### T2-001：社区边界设计与阶段 2 工单切换

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T1-007`
- 目标：确认社区/板块建模边界，并把当前工单板切换到阶段 2。
- 交付物：
  - `docs/internal/architecture/community-v1.md`
  - `tasks.md`
  - `.ai/slices/stage-02-community/README.md`
- 结论：
  - 代码和数据库统一使用 `Community` 表达社区/板块。
  - 公共总版是系统内置 `Community`，不是全站聚合 Feed。
  - 用户申请板块使用 `CommunityApplication`，审批通过后再创建 `Community`。
  - 社区成员角色只表达社区内权限；平台审批者是系统级身份。
  - 帖子后续只依赖 `community_id` 和 `CanPostInCommunity`。

### T2-002：社区数据表与平台审批者边界

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T2-001`
- 目标：为社区基础管理建立最小 schema。
- 交付物：
  - `communities` 表。
  - `community_memberships` 表。
  - `community_applications` 表。
  - 平台审批者最小身份字段或关系表。
  - migration 测试或本地验证记录。
- 完成标准：
  - `communities.slug` 全局唯一。
  - `communities.kind` 支持 `system`、`user_created`。
  - `communities.status` 支持 `active`、`suspended`、`archived`。
  - `community_memberships.role` 支持 `owner`、`moderator`、`member`。
  - `community_applications.status` 支持 `pending`、`approved`、`rejected`、`canceled`。
  - 平台审批者不通过 JWT claims 判断。
- 注意点：
  - 不做复杂 RBAC 或权限矩阵。
  - 不把社区成员角色复用成平台审批权限。
  - 不做私密社区、邀请制和复杂成员加入流程。
- 结论：
  - 社区 schema 落在 `migrations/000003_create_community_schema.*.sql`。
  - 平台审批者使用 `users.is_platform_staff` 表达，后续审批 usecase 从数据库读取，不进入 JWT claims。
  - `communities`、`community_memberships`、`community_applications` 已具备最小枚举约束和唯一约束。
  - `community_applications` 对 `pending` 申请的 `requested_slug` 做部分唯一约束，避免待审批 slug 冲突。

### T2-003：社区领域模型与仓储边界

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T2-002`
- 目标：建立社区、成员关系和申请单的领域对象、值对象和仓储接口。
- 交付物：
  - `internal/community/communitydomain`
  - `internal/community/communityrepository`
  - domain 测试
  - repository 测试
- 完成标准：
  - slug、name、status、kind、visibility、role、application status 的校验进入 domain。
  - repository 负责数据库错误映射，例如 slug 冲突映射为 `conflict`。
  - domain 不依赖 Gin、pgx、HTTP 或 JWT。
  - usecase 所需 repository interface 定义在业务侧。
- 注意点：
  - 不引入万能 `BaseRepository`。
  - 不把审批流程写成贫血 CRUD。
- 结论：
  - `internal/community/communitydomain` 已建立社区、成员关系和申请单领域模型。
  - `internal/community/communityusecase/repository.go` 已定义 usecase 侧仓储接口。
  - `internal/community/communityrepository` 已实现 PostgreSQL repository。
  - repository 已覆盖 slug 冲突、pending 申请 slug 冲突、成员关系重复和 not_found 映射。

### T2-004：公共总版与社区读取接口

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T2-003`
- 目标：让系统具备可稳定依赖的公共社区和社区读取能力。
- 交付物：
  - 公共总版初始化方案。
  - `GET /api/v1/communities`
  - `GET /api/v1/communities/:slug`
  - usecase、query/repository、HTTP 测试。
- 完成标准：
  - 公共总版 `slug=public` 存在且不可由普通用户修改。
  - 社区列表默认只返回可公开读取的 `active` 社区。
  - 社区详情可以通过 slug 查询。
  - handler 只做请求解析和响应映射，不承载业务判断。
- 注意点：
  - 全站 Feed 不在本工单实现。
  - 社区头像、背景图、分类目录暂缓。
- 结论：
  - 公共总版初始化放在 API 启动监听前的 `communityusecase.EnsurePublicCommunity`，不放在 handler 或读取接口里临时创建。
  - 公共总版缺失时通过 domain 构造 `Community(kind=system, slug=public, status=active, visibility=public)` 并写入 repository。
  - 已存在的 `slug=public` 如果不是合法系统公共总版，服务启动失败，不静默修复。
  - `GET /api/v1/communities` 和 `GET /api/v1/communities/:slug` 已接入认证保护。
  - 社区列表只返回 active public 社区；详情接口对不可公开读取的社区返回 `not_found`。
  - handler 只做请求解析和响应映射，读取编排留在 usecase。

### T2-005：用户提交社区申请

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T2-003`
- 目标：让已登录用户可以申请创建社区/板块。
- 交付物：
  - `POST /api/v1/community-applications`
  - 提交申请 usecase
  - HTTP handler 和测试
- 完成标准：
  - 未登录返回 `unauthenticated`。
  - 无效 slug/name/reason 返回 `invalid_argument`。
  - 与已有社区 slug 冲突返回 `conflict`。
  - 与其他 `pending` 申请 slug 冲突返回 `conflict`。
  - 成功提交后申请状态为 `pending`。
- 注意点：
  - 提交申请不直接创建社区。
  - 不在 handler 判断重复申请或 slug 冲突。
- 结论：
  - `POST /api/v1/community-applications` 已接入受保护路由。
  - handler 只解析 JSON 和当前用户 ID，不判断社区 slug 或 pending 申请冲突。
  - `CommunityApplicationUseCase.SubmitCommunityApplication` 复用 domain 校验 slug、name、reason。
  - 与已有社区 slug 冲突由 usecase 查询 `CommunityRepository.FindBySlug` 后映射为 `conflict`。
  - 与其他 pending 申请 slug 冲突由 PostgreSQL 部分唯一索引和 repository 错误映射返回 `conflict`。
  - usecase 和 HTTP 测试已覆盖成功、未登录、非法参数和冲突路径。

### T2-006：审批社区申请并创建社区

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T2-005`
- 目标：让平台审批者可以批准或拒绝社区申请。
- 交付物：
  - `POST /api/v1/community-applications/:id/approve`
  - `POST /api/v1/community-applications/:id/reject`
  - 审批 usecase
  - 事务内创建社区和 owner 成员关系
  - usecase、repository、HTTP 测试
- 完成标准：
  - 非平台审批者返回 `forbidden`。
  - 不存在的申请返回 `not_found`。
  - 非 `pending` 申请不能重复审批。
  - 批准申请在同一事务中完成：申请 `approved`、社区 `active`、申请人 `owner`。
  - 拒绝申请记录 `reject_reason`，不创建社区。
- 注意点：
  - 审批者身份不从 JWT claims 读取。
  - 不做复杂后台管理台。
- 结论：
  - `POST /api/v1/community-applications/:id/approve` 已接入受保护路由。
  - `POST /api/v1/community-applications/:id/reject` 已接入受保护路由。
  - 审批者身份通过 `users.is_platform_staff` 从数据库读取，不进入 JWT claims。
  - 非平台审批者返回 `forbidden`。
  - 审批 usecase 通过 `CommunityTransactionManager.WithinTx` 在同一事务内锁定申请、保存审批状态、创建社区、创建 owner 成员关系。
  - 审批通过创建 `Community(kind=user_created, status=active, visibility=public)`，并把申请人写为 `owner`。
  - 拒绝只更新申请状态和 `reject_reason`，不创建社区。
  - repository 集成测试覆盖事务成功路径，以及社区 slug 冲突时审批状态回滚为 `pending`。

### T2-007：阶段 2 冒烟检查与文档收口

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T2-006`
- 目标：验证阶段 2 社区基础管理闭环，并收口 README 与文档。
- 交付物：
  - 阶段 2 本地冒烟步骤。
  - README 当前状态更新。
  - 阶段 2 遗留限制记录。
  - 阶段 3 方向选择建议。
- 完成标准：
  - `go test ./...` 通过。
  - `go build -buildvcs=false ./...` 通过。
  - 本地真实冒烟覆盖公共总版、社区列表、社区详情、申请提交、批准和拒绝。
  - 阶段 2 已完成能力和未完成能力边界清楚。
- 结论：
  - `go test ./...` 已通过。
  - `go build -buildvcs=false ./...` 已通过。
  - 真实本地冒烟已通过：健康检查、注册普通用户、注册审批者、SQL 设置 `is_platform_staff=true`、读取公共总版、提交申请、批准申请、读取新社区、拒绝申请。
  - 冒烟结果包含 `SubmittedApplicationStatus=pending`、`ApprovedApplicationStatus=approved`、`CreatedCommunitySlug=smoke-2225547798`、`RejectedApplicationStatus=rejected`。
  - README、`.ai/slices/stage-02-community/` 和 `docs/internal/architecture/community-v1.md` 已同步阶段 2 收口结论。

## 阶段 1 复盘摘要

阶段 1 已完成：

- 用户可以注册。
- 用户可以登录并获得 Bearer JWT access token。
- 受保护接口可以识别当前用户 ID。
- `GET /api/v1/me` 返回当前用户公开字段。
- 密码不会明文存储。
- 认证错误统一映射为 `unauthenticated`。
- 本地真实冒烟已覆盖注册、登录、带 token 访问 `/me` 和关键失败路径。

阶段 1 遗留限制：

- 不做 refresh token、多端会话、token 主动吊销。
- 不做第三方登录、邮箱验证和找回密码。
- 不做复杂 RBAC。
- `/me` 不做资料编辑、头像、邮箱和权限列表。
- 认证中间件只解析 Bearer access token，不查数据库。

### T1-007：阶段 1 复盘与阶段 2 方向选择

- 状态：`DONE`
- 优先级：`P0`
- 前置依赖：`T1-006`
- 结论：
  - 阶段 1 已满足退出标准。
  - 阶段 2 选择 `社区/板块基础管理`。
  - 阶段 2 先解决社区实体、成员关系、申请单和平台审批者边界。
  - 不提前铺满帖子、评论、投票、Feed 与审核的详细工单。

## 后续阶段方向

阶段 2 完成后，从以下方向选择下一阶段：

- 帖子主链路
- 评论与投票
- Feed 与审核

选择规则：

- 优先选择能形成主链路闭环的阶段。
- 如果基础设施暴露问题，先补基础设施。
- 不提前一次性写满未来阶段工单。

## Stage 49 Community Member Governance Contract

- Status: `DONE`
- Branch: `main`
- Goal: close the frontend P1 community member governance backend gap without introducing a separate platform role hierarchy.

### T49-001 Community moderators and owner transfer

- Status: `DONE`
- Priority: `P1`
- Dependencies: stage 48 user follow/social counters.
- Delivered:
  - Added `POST /api/v1/communities/:slug/manage/moderators` for owner-only moderator appointment by username.
  - Added `DELETE /api/v1/communities/:slug/manage/moderators/:user_id` for owner-only moderator demotion back to member.
  - Added `POST /api/v1/communities/:slug/manage/owner-transfer` and `POST /api/v1/communities/:slug/manage/owner-transfer/:transfer_id/accept` for two-step owner transfer.
  - Added `POST /api/v1/admin/communities/:id/owner` for platform staff owner takeover with audit log.
  - Added `migrations/000026_create_community_owner_transfers.*.sql` for transfer records and the one-active-owner uniqueness guard.
  - Updated HTTP route/schema/migration contracts, README and frontend `backend-api-needs.md`.
- Rules:
  - Community moderator writes require the current viewer to be the active owner.
  - Active member count controls moderator cap: `<500 => 5`, `>=500 => 10`, `>=2000 => 20`.
  - Owner transfer is not effective until the target user accepts it.
  - Platform takeover kept using the current `is_platform_staff` boundary at this stage; precise `platform_role=owner|admin|staff|null` was completed in Stage 50 and tightened in Stage 52.
- Verification:
  - `go test ./internal/community/... ./internal/admin/... ./cmd/api -count=1`
  - `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-api-contract-doc.ps1`
  - `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-api-schema-doc.ps1`
  - `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-migration-contract.ps1`

## Stage 50 Platform Role Hierarchy Contract

- Status: `DONE`
- Branch: `main`
- Goal: close the remaining P1 platform role hierarchy backend gap while preserving the legacy `is_platform_staff` boundary.

### T50-001 Platform owner/admin/staff roles

- Status: `DONE`
- Priority: `P1`
- Dependencies: Stage 49 community member governance.
- Delivered:
  - Added `migrations/000027_add_platform_roles.*.sql` with `users.platform_role`.
  - Backfilled existing `is_platform_staff=true` users to `platform_role='owner'`.
  - Added `PATCH /api/v1/admin/users/:id/platform-role`.
  - Added `platform_role` to admin user responses.
  - Kept old `is_platform_staff` compatibility: precise role writes synchronize the bool flag.
- Rules:
  - Role values are `owner`, `admin`, `staff`, or `null`.
  - Direct role writes now only let owner update non-owner users to `admin`, `staff`, or `null`.
  - Direct role writes cannot create, downgrade, or alter an owner account.
  - Admin and staff cannot change platform roles.
  - Owner creation, transfer and recovery are handled by the Stage 52 owner-transfer/bootstrap/recovery contract.

## Stage 51 Moderation Sanctions and Community Removal

- Status: `DONE`
- Branch: `main`
- Goal: close the remaining P1 moderation sanctions and community-scoped removal backend gap.

### T51-001 Account bans and community-scoped removal

- Status: `DONE`
- Priority: `P1`
- Dependencies: Stage 50 platform role hierarchy.
- Delivered:
  - Added `migrations/000028_create_user_sanctions.*.sql` for durable account sanctions with revocation state.
  - Added `POST /api/v1/admin/users/:id/sanctions`, `GET /api/v1/admin/users/:id/sanctions` and `POST /api/v1/admin/user-sanctions/:sanction_id/revoke`.
  - Added `POST /api/v1/communities/:slug/moderation/posts/:id/remove` and `POST /api/v1/communities/:slug/moderation/comments/:id/remove`.
  - Active account bans now block password login and protected-token validation until revoked or expired.
  - Updated HTTP route/schema/migration contracts, README and frontend `backend-api-needs.md`.
- Rules:
  - Sanction type currently supports `account_ban`.
  - Duration is fixed to `1d`, `3d`, `7d`, `30d` or `permanent`; backend computes `expires_at`.
  - Permanent sanctions use `expires_at=null`; expired temporary sanctions read as `expired`.
  - Owner can sanction non-owner users; admin can sanction users without a platform role; staff can only view sanctions.
  - Revoke writes `revoked_by` and `revoked_at` and preserves the sanction record.
  - Community owner/moderator removal is limited to posts/comments in the path community and does not grant account-ban authority.
- Verification:
  - `go test ./internal/moderation/... ./internal/admin/... ./internal/auth/... ./cmd/api -count=1`
  - `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-api-contract-doc.ps1`
  - `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-api-schema-doc.ps1`
  - `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-migration-contract.ps1`

## Stage 52 Platform Owner Transfer and Recovery

- Status: `DONE`
- Branch: `main`
- Goal: close the platform owner transfer/recovery gap without allowing web-side owner takeover.

### T52-001 Platform owner transfer and recovery

- Status: `DONE`
- Priority: `P1`
- Dependencies: Stage 51 moderation sanctions.
- Delivered:
  - Added `migrations/000029_create_platform_owner_transfers.*.sql` for pending owner transfer records, system audit actor refs and a single active platform owner guard.
  - Tightened `PATCH /api/v1/admin/users/:id/platform-role` so it can only set non-owner users to `admin`, `staff`, or `null`.
  - Tightened `PATCH /api/v1/admin/users/:id` so owners cannot be disabled/restored/deleted through normal user management and admin writes only cover ordinary users.
  - Added platform owner transfer routes: `GET/POST/DELETE /api/v1/admin/owner-transfer`, `GET /api/v1/owner-transfer/:transfer_id` and `POST /api/v1/owner-transfer/:transfer_id/accept`.
  - Added `cmd/admin bootstrap-owner` and `cmd/admin recover-owner` for deployment-side first-owner bootstrap and compromised-owner recovery.
- Rules:
  - Only the current active owner can create a transfer; target must be active and non-owner.
  - Transfer creation and acceptance both require the acting account's current password.
  - Transfers expire after 48 hours and only one pending transfer can exist at a time.
  - Acceptance promotes the target to the only active owner, downgrades the previous owner to the recorded previous role and revokes previous-owner sessions.
  - Recovery is CLI-only and writes system audit logs through `actor_ref`.
- Verification:
  - `go test ./internal/admin/... -count=1`

## Stage 53 Reddit Mod Tools P1 Contract

- Status: `DONE`
- Branch: `main`
- Goal: close the frontend Reddit-style moderation tools P1 backend gap while keeping Modmail, Automod and scheduling as FUTURE scope.

### T53-001 Moderation queues, content actions and mod tooling

- Status: `DONE`
- Priority: `P1`
- Dependencies: Stage 52 platform owner transfer and recovery.
- Delivered:
  - Added `migrations/000030_create_moderation_tools.*.sql`.
  - Added platform and community moderation queues:
    - `GET /api/v1/admin/mod-queues`
    - `POST /api/v1/admin/mod-queues/actions`
    - `GET /api/v1/communities/:slug/mod-queues`
    - `POST /api/v1/communities/:slug/mod-queues/actions`
  - Added community content actions: approve, spam, ignore report, lock, pin, mark NSFW, mark spoiler and flair.
  - Added community removal reasons and saved responses CRUD.
  - Added community banned / muted / approved users, moderation user profile and mod notes.
  - Added community Mod Log with action, actor and target filters.
  - Updated HTTP route/schema contracts, migration contract and frontend `backend-api-needs.md`.
- Rules:
  - Platform queue routes require platform staff.
  - Community mod tools require active community owner or moderator.
  - Batch actions return per-target success or failure.
  - Content actions write `moderation_actions` and `community_moderation_logs`.
  - Visible post flags use independent columns for locked, pinned, NSFW, spoiler and flair instead of hiding the post through `status=locked`.
  - `spam` is a content status and remains hidden from ordinary public reads because public reads only return `visible`.
  - Removal reasons and saved responses are soft-deactivated, not physically deleted.
  - P2 Modmail, Automod, flair templates, scheduled posts, guides, digest and insights remain FUTURE.
- Verification:
  - `go test ./internal/moderation/...`
  - `go test ./internal/post/... ./internal/comment/... ./internal/moderation/... -count=1`
  - `go test ./... -count=1`
  - `go build -buildvcs=false ./...`
  - `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-api-contract-doc.ps1`
  - `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-api-schema-doc.ps1`
  - `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-migration-contract.ps1`
  - `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-http-error-contract-doc.ps1`

## Stage 54 Platform Owner Community Override

- Status: `DONE`
- Branch: `main`
- Goal: close the frontend platform owner all-community management override gap without writing platform owners into every community membership.

### T54-001 Platform owner override for community management and mod tools

- Status: `DONE`
- Priority: `P1`
- Dependencies: Stage 53 Reddit Mod Tools P1 contract.
- Delivered:
  - Added active platform owner checks to community management and Reddit Mod Tools community scope.
  - Added `viewer_permissions.platform_owner_override` to community permission responses.
  - Allowed platform owner override for community manage reads, settings update, rule CRUD, moderator appoint/remove and community-scoped Mod Tools.
  - Kept `viewer_role` as the real community membership role.
  - Kept community owner transfer restricted to the real community owner; platform owner community takeover remains the platform admin owner route.
  - Updated HTTP route/schema contracts, README and frontend `backend-api-needs.md`.
- Rules:
  - Only `platform_role=owner` with active user status receives the override.
  - Platform `admin` and `staff` do not receive all-community owner permissions.
  - Override does not write `community_memberships`.
  - Community Mod Tools operations still scope by the path community ID.
- Verification:
  - `go test ./internal/community/... ./internal/moderation/... -count=1`
  - `go test ./... -count=1`
  - `go build -buildvcs=false ./...`
  - `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-api-contract-doc.ps1`
  - `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-api-schema-doc.ps1`
  - `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-migration-contract.ps1`
  - `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-http-error-contract-doc.ps1`

## Stage 56 Notification Interaction Context Contract

- Status: `DONE`
- Branch: `main`
- Goal: close the remaining frontend backend gap for the interaction/system notification center and comment notification context.

### T56-001 Notification interactions, actors and content context

- Status: `DONE`
- Priority: `P1`
- Dependencies: Stage 55 community profile media and owner transfer usability.
- Delivered:
  - Added `category=interactions` to `GET /api/v1/notifications`.
  - Changed missing notification `status` default from `unread` to `all` for the rebuilt message center.
  - Kept explicit `status=all|unread|read` support for compatibility.
  - Extended notification responses with `actor`, `last_actor`, `aggregate_count` and `context`.
  - Joined notification context from existing users, posts, comments and communities without a new migration.
  - Added comment notification permalinks with `#comment-<comment_id>` anchors.
  - Documented that private messages/conversations are still not a live backend capability.
- Rules:
  - `category=interactions` contains replies, mentions and likes only.
  - `category=system` remains separate and must not be mixed into interactions.
  - Frontend should not expose a working private-message entrypoint until a later formal contract lands.
- Verification:
  - `go test ./internal/notification/...`
  - `go test ./...`
  - `go build -buildvcs=false ./...`
  - `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-api-contract-doc.ps1`
  - `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-api-schema-doc.ps1`
  - `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-migration-contract.ps1`
  - `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-http-error-contract-doc.ps1`
  - `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-config-contract-doc.ps1`
  - `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-config-semantics-doc.ps1`
  - `docker compose --env-file .env -f docker-compose.prod.yml build`
  - `docker compose --env-file .env -f docker-compose.prod.yml run --rm migrate version`
  - Local HTTP probes against `http://localhost:8080/healthz`, notification category routes and community owner-transfer/settings routes.

## Stage 55 Community Profile Media and Owner Transfer Usability

- Status: `DONE`
- Branch: `main`
- Goal: close the remaining frontend backend gap for community profile media writes, community owner transfer usability and platform takeover reason audit.

### T55-001 Community settings media, owner-transfer read/cancel and admin takeover reason

- Status: `DONE`
- Priority: `P1`
- Dependencies: Stage 54 platform owner community override.
- Delivered:
  - Added `communities.avatar_url` and `communities.banner_url` with http/https and 2048-byte constraints.
  - Extended community settings update to accept `name`, `description`, `avatar_url` and `banner_url` as optional fields with at least one field required.
  - Added community owner transfer `expires_at` and `cancelled_at` state with 48-hour pending transfer TTL.
  - Added `GET /api/v1/communities/:slug/manage/owner-transfer`, `GET /api/v1/communities/:slug/owner-transfer/:transfer_id` and `DELETE /api/v1/communities/:slug/manage/owner-transfer/:transfer_id`.
  - Extended owner transfer response with usernames, display names, target/cancel viewer flags, expiry and cancellation timestamps.
  - Extended `POST /api/v1/admin/communities/:id/owner` to accept optional `reason` and write it into the platform audit after state.
  - Updated HTTP route/schema contracts, migration contract, README and frontend `backend-api-needs.md`.
- Rules:
  - Community moderators can read settings but cannot update settings.
  - Real community owner or platform owner override can update community profile media.
  - Creating a community owner transfer still requires the real community owner.
  - Cancelling a pending transfer requires the initiator or platform owner override.
  - Accepting an owner transfer requires the target user and a non-expired pending transfer.
- Verification:
  - `go test ./internal/community/...`
  - `go test ./internal/admin/...`
