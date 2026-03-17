# CUMT-Nexus 单校校园论坛核心设计 (DESIGN)

本文档是当前阶段的数据结构与领域设计基线。

目标不是机械照搬 Reddit，而是做一个“类 Reddit 的单校校园论坛”：

- 讨论内容有明确板块归属
- 评论支持楼中楼
- 帖子和评论支持投票
- 额外支持“用户创建词条 + 打分 + 点评”

从现在开始，`subreddit` 不再作为正式命名使用，统一改为 `community`。
对前台产品来说可以叫“板块”，但在后端模型、数据库表、接口命名里统一使用 `community/communities`。

---

## 1. 产品定位与边界

### 1.1 产品定位

这是一个单校校园论坛，不做多校切换，不做跨校隔离。

核心有两条内容线：

1. **论坛线**：板块、帖子、评论、投票
2. **词条线**：词条、评分、点评

### 1.2 第一阶段不做的事

为了避免模型过早失控，第一阶段明确不做：

- 多校区 / 多学校模型
- 用户自由创建板块
- 板块关注 / 加入 / 订阅关系
- 一个通用 `votes` 表靠 `target_type` 支撑所有投票
- 为词条再单独做一套“评论树”

### 1.3 当前产品判断

对本项目来说，板块更接近“平台预置栏目”，不是 Reddit 那种用户自治小社区。

因此：

- 板块数量由平台设计和治理
- 用户不需要先关注或加入板块才能浏览和发帖
- 用户真正自然生长的是帖子、标签、评论、词条和点评

---

## 2. 统一命名

### 2.1 领域命名

- `User`：用户
- `Community`：板块，数据库表为 `communities`
- `CommunityModerator`：板块版主管理关系
- `Post`：帖子
- `Comment`：评论
- `PostVote`：帖子投票
- `CommentVote`：评论投票
- `Entry`：词条
- `EntryReview`：词条评分与点评

### 2.2 命名迁移规则

这份文档生效后，旧命名视为过渡状态：

- `Subreddit` -> `Community`
- `Subscription` -> 删除，不作为核心模型保留
- `Vote(target_type)` -> 拆成 `PostVote` 和 `CommentVote`

---

## 3. 核心领域关系

### 3.1 论坛线

- 一个 `Community` 可以有很多 `Post`
- 一个 `Post` 只能属于一个 `Community`
- 一个 `Post` 由一个 `User` 创建
- 一个 `Post` 可以有很多 `Comment`
- 一个 `Comment` 只能属于一个 `Post`
- 一个 `Comment` 可以回复另一个 `Comment`
- 一个 `User` 可以对多个帖子投票
- 一个 `User` 可以对多个评论投票

### 3.2 词条线

- 一个 `Entry` 由一个 `User` 发起创建
- 一个 `Entry` 可以有很多 `EntryReview`
- 一个 `User` 对同一个 `Entry` 只能有一条有效点评

### 3.3 管理线

- 一个 `Community` 可以有多个版主
- 举报、审核、封禁属于后续治理能力，不进入第一阶段主闭环

---

## 4. 数据库表结构设计

### 4.1 用户表 (`users`)

| 字段名 | 类型 | 说明 |
| :-- | :-- | :-- |
| `id` | BIGINT | 主键，自增 |
| `username` | VARCHAR(64) | 登录用户名，唯一索引 |
| `password_hash` | VARCHAR(255) | 密码哈希 |
| `nickname` | VARCHAR(64) | 显示昵称 |
| `avatar_url` | VARCHAR(255) | 头像 |
| `bio` | TEXT | 简介 |
| `role` | TINYINT | 0-普通用户，1-管理员 |
| `status` | TINYINT | 1-正常，2-封禁 |
| `created_at` | DATETIME | 创建时间 |
| `updated_at` | DATETIME | 更新时间 |

说明：

- 单校场景下，先不要拆 `campus` 表
- 以后如果要做实名、学号认证，应单独开认证表，不要直接混进主用户表

### 4.2 板块表 (`communities`)

| 字段名 | 类型 | 说明 |
| :-- | :-- | :-- |
| `id` | BIGINT | 主键，自增 |
| `slug` | VARCHAR(64) | 稳定标识，唯一索引，例如 `study` |
| `name` | VARCHAR(64) | 展示名称，例如“学习课程” |
| `description` | TEXT | 板块简介 |
| `icon_url` | VARCHAR(255) | 图标 |
| `banner_url` | VARCHAR(255) | 横幅 |
| `sort_order` | INT | 排序权重，越小越靠前 |
| `post_permission` | TINYINT | 1-所有登录用户可发帖，2-仅版主可发帖 |
| `allow_anonymous` | TINYINT | 0-不允许匿名，1-允许匿名 |
| `status` | TINYINT | 1-正常，2-只读，3-隐藏 |
| `post_count` | BIGINT | 帖子数，冗余统计 |
| `created_at` | DATETIME | 创建时间 |
| `updated_at` | DATETIME | 更新时间 |

说明：

- 板块是平台预置数据，不走普通用户创建流程
- 第一阶段默认所有板块都可浏览
- 不保留 `member_count`，因为当前没有关注/加入模型

### 4.3 板块版主表 (`community_moderators`)

| 字段名 | 类型 | 说明 |
| :-- | :-- | :-- |
| `id` | BIGINT | 主键，自增 |
| `community_id` | BIGINT | 板块 ID |
| `user_id` | BIGINT | 用户 ID |
| `role` | TINYINT | 1-owner，2-moderator |
| `created_at` | DATETIME | 创建时间 |

说明：

- `community_id + user_id` 建联合唯一索引
- 这张表解决“谁可以管理板块”的问题，不承担关注功能

### 4.4 帖子表 (`posts`)

| 字段名 | 类型 | 说明 |
| :-- | :-- | :-- |
| `id` | BIGINT | 主键，自增 |
| `community_id` | BIGINT | 所属板块 |
| `user_id` | BIGINT | 作者 ID |
| `type` | TINYINT | 1-text，2-link，3-image |
| `title` | VARCHAR(300) | 标题 |
| `content` | LONGTEXT | 文本正文 |
| `link_url` | VARCHAR(1024) | 链接帖地址 |
| `is_anonymous` | TINYINT | 0-否，1-是 |
| `score` | BIGINT | 投票分数 |
| `comment_count` | BIGINT | 评论数 |
| `view_count` | BIGINT | 浏览量 |
| `status` | TINYINT | 1-正常，2-锁定，3-删除 |
| `created_at` | DATETIME | 创建时间 |
| `updated_at` | DATETIME | 更新时间 |
| `deleted_at` | DATETIME | 逻辑删除 |

说明：

- 第一阶段先支持 `text` 和 `link`，`image` 预留字段即可
- 匿名发帖只影响展示，不影响数据库里保留真实 `user_id`
- `community_id + created_at` 是列表查询重要索引

### 4.5 评论表 (`comments`)

| 字段名 | 类型 | 说明 |
| :-- | :-- | :-- |
| `id` | BIGINT | 主键，自增 |
| `post_id` | BIGINT | 所属帖子 |
| `user_id` | BIGINT | 评论作者 |
| `parent_id` | BIGINT | 父评论 ID，顶级评论为 0 |
| `root_id` | BIGINT | 根评论 ID，顶级评论为 0 |
| `depth` | SMALLINT | 评论层级 |
| `content` | LONGTEXT | 评论内容 |
| `is_anonymous` | TINYINT | 0-否，1-是 |
| `score` | BIGINT | 投票分数 |
| `reply_count` | BIGINT | 子评论数 |
| `status` | TINYINT | 1-正常，2-删除 |
| `created_at` | DATETIME | 创建时间 |
| `updated_at` | DATETIME | 更新时间 |
| `deleted_at` | DATETIME | 逻辑删除 |

说明：

- 第一阶段只要求 `parent_id + root_id + depth` 正确
- 不强制在第一阶段引入 `path`
- 查询评论列表时，先按帖子拉平，再在 Service 层组装树结构

### 4.6 帖子投票表 (`post_votes`)

| 字段名 | 类型 | 说明 |
| :-- | :-- | :-- |
| `id` | BIGINT | 主键，自增 |
| `user_id` | BIGINT | 投票用户 |
| `post_id` | BIGINT | 帖子 ID |
| `value` | TINYINT | 只允许 `1` 或 `-1` |
| `created_at` | DATETIME | 创建时间 |
| `updated_at` | DATETIME | 更新时间 |

说明：

- `user_id + post_id` 建联合唯一索引
- 取消投票直接删除记录，先不要引入 `0`

### 4.7 评论投票表 (`comment_votes`)

| 字段名 | 类型 | 说明 |
| :-- | :-- | :-- |
| `id` | BIGINT | 主键，自增 |
| `user_id` | BIGINT | 投票用户 |
| `comment_id` | BIGINT | 评论 ID |
| `value` | TINYINT | 只允许 `1` 或 `-1` |
| `created_at` | DATETIME | 创建时间 |
| `updated_at` | DATETIME | 更新时间 |

说明：

- `user_id + comment_id` 建联合唯一索引
- 分表设计优于一个通用 `votes` 表

### 4.8 词条表 (`entries`)

| 字段名 | 类型 | 说明 |
| :-- | :-- | :-- |
| `id` | BIGINT | 主键，自增 |
| `entry_type` | VARCHAR(32) | 词条类型，如 `course`、`teacher`、`shop` |
| `title` | VARCHAR(128) | 词条标题 |
| `slug` | VARCHAR(128) | 稳定标识，可做唯一索引 |
| `summary` | VARCHAR(255) | 摘要 |
| `content` | LONGTEXT | 词条正文 |
| `created_by` | BIGINT | 创建者 |
| `status` | TINYINT | 1-pending，2-published，3-rejected，4-merged |
| `merged_to_id` | BIGINT | 合并目标词条 ID，默认 0 |
| `avg_score` | DECIMAL(3,2) | 平均分 |
| `review_count` | BIGINT | 点评数 |
| `created_at` | DATETIME | 创建时间 |
| `updated_at` | DATETIME | 更新时间 |

说明：

- 用户可以创建词条，但必须考虑重复词条问题
- `status + merged_to_id` 用于后续处理合并、驳回、去重
- 第一阶段先做通用词条，不急着拆 `course_entries`、`teacher_entries`

### 4.9 词条点评表 (`entry_reviews`)

| 字段名 | 类型 | 说明 |
| :-- | :-- | :-- |
| `id` | BIGINT | 主键，自增 |
| `entry_id` | BIGINT | 所属词条 |
| `user_id` | BIGINT | 点评用户 |
| `score` | TINYINT | 1-5 分 |
| `content` | TEXT | 点评内容 |
| `is_anonymous` | TINYINT | 0-否，1-是 |
| `status` | TINYINT | 1-正常，2-隐藏，3-删除 |
| `created_at` | DATETIME | 创建时间 |
| `updated_at` | DATETIME | 更新时间 |

说明：

- `user_id + entry_id` 建联合唯一索引
- 一个用户对一个词条只保留一条有效点评
- “打分”和“评论”合并在同一条 `review` 中，不再单独拆散表

---

## 5. 索引与约束建议

至少应保证以下唯一索引：

- `users.username`
- `communities.slug`
- `community_moderators(community_id, user_id)`
- `post_votes(user_id, post_id)`
- `comment_votes(user_id, comment_id)`
- `entry_reviews(user_id, entry_id)`

至少应保证以下常用查询索引：

- `posts(community_id, created_at)`
- `posts(user_id, created_at)`
- `comments(post_id, created_at)`
- `comments(parent_id, created_at)`
- `entries(entry_type, status, created_at)`
- `entry_reviews(entry_id, created_at)`

---

## 6. 板块初始化策略

板块属于系统预置数据，不属于用户业务数据。

因此应采用 `seed` 方式初始化：

- 启动初始化或单独命令执行
- 以 `slug` 为唯一键幂等插入
- 存在则跳过，不存在则创建

推荐第一批板块控制在 6~8 个：

- `square`：综合广场
- `study`：学习课程
- `trade`：二手交易
- `lost-found`：失物招领
- `activity`：活动组队
- `life`：校园生活
- `treehole`：树洞匿名
- `notice`：资讯通知

---

## 7. API 领域设计

### 7.1 认证域 (`auth`)

- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`

### 7.2 当前用户域 (`users`)

- `GET /api/v1/users/me`

### 7.3 板块域 (`communities`)

- `GET /api/v1/communities`
- `GET /api/v1/communities/:slug`

说明：

- 第一阶段不提供普通用户创建板块接口
- 如果后面需要后台创建板块，应走管理员接口

### 7.4 帖子域 (`posts`)

- `POST /api/v1/posts`
- `GET /api/v1/communities/:slug/posts`
- `GET /api/v1/posts/:id`

发帖最小请求体建议包含：

```json
{
  "community_slug": "study",
  "type": 1,
  "title": "高数期中怎么复习？",
  "content": "想问一下学长学姐有什么建议",
  "is_anonymous": 0
}
```

### 7.5 评论域 (`comments`)

- `POST /api/v1/posts/:post_id/comments`
- `GET /api/v1/posts/:post_id/comments`

### 7.6 投票域 (`votes`)

- `POST /api/v1/posts/:post_id/votes`
- `POST /api/v1/comments/:comment_id/votes`

说明：

- 请求体最小只需要 `value`
- `value = 1` 表示赞成，`value = -1` 表示反对

### 7.7 词条域 (`entries`)

- `POST /api/v1/entries`
- `GET /api/v1/entries`
- `GET /api/v1/entries/:id`

推荐第一批词条类型：

- `course`
- `teacher`
- `shop`

### 7.8 点评域 (`entry_reviews`)

- `POST /api/v1/entries/:entry_id/reviews`
- `GET /api/v1/entries/:entry_id/reviews`
- `PATCH /api/v1/entry-reviews/:id`

---

## 8. 统一错误设计

建议继续沿用统一响应结构：

```json
{
  "code": 0,
  "msg": "success",
  "data": null
}
```

当前阶段至少补齐以下业务错误：

- `用户名已存在`
- `用户不存在`
- `板块不存在`
- `帖子不存在`
- `评论不存在`
- `父评论不合法`
- `词条不存在`
- `词条已存在或重复`
- `点评不存在`
- `用户已经点评过该词条`
- `权限不足`

---

## 9. 后端架构约束

### 9.1 分层原则

- `Controller`：HTTP 绑定、响应、鉴权上下文读取
- `Service`：业务规则、DTO 组装、事务边界
- `Repository`：数据库读写、联表查询、分页查询

### 9.2 DTO 原则

- 不直接把 GORM Model 原样返回前端
- 匿名内容要在 DTO 层处理展示名和头像
- 列表 DTO 和详情 DTO 可以不同，不要强求一个结构走到底

### 9.3 事务原则

以下场景建议使用事务：

- 创建评论并更新帖子评论数
- 写入投票并更新帖子/评论分数
- 写入点评并更新词条平均分与点评数

### 9.4 迁移原则

- `AutoMigrate` 只负责补表和补字段，不负责复杂重命名
- `subreddits -> communities`、`votes -> post_votes/comment_votes` 这类变更要准备手动迁移方案
- 如果数据库里已经有旧表，必须先想清楚旧数据保留、映射还是废弃

---

## 10. 当前阶段最重要的结论

当前项目不应该继续沿着 `subreddit/subscription` 方向推进，而应该切换到下面这条主线：

- 用 `community` 表达板块
- 不做板块关注/加入
- 帖子必须归属于板块
- 评论先走 `parent_id + root_id + depth`
- 投票拆成帖子投票和评论投票
- 新增 `entry + entry_review` 作为校园词条能力

这份文档是接下来模型重构、接口设计和代码重构的依据。
