# Migration 契约

本文记录当前 PostgreSQL migration 的文件命名、配对和清单规则。它只维护可复用工程约定，不记录一次性执行流水账。

## 文件规则

- migration 文件放在 `migrations/`。
- 文件名必须是 `000001_name.up.sql` 和 `000001_name.down.sql` 这种格式。
- 版本号必须是 6 位数字，并且从 `000001` 开始连续递增。
- 同一版本必须同时存在 `.up.sql` 和 `.down.sql`。
- 同一版本的 up/down 文件名中的 `name` 必须完全一致。
- `name` 使用小写字母、数字和下划线。
- 不在已有版本中插入新 migration；新增 schema 变更只追加下一个版本号。

## 校验命令

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-migration-contract.ps1
```

该脚本校验：

- `migrations/` 下没有不符合命名规则的文件。
- 每个版本都有且只有一个 up 文件和一个 down 文件。
- migration 版本从 `000001` 起连续。
- up/down 文件名中的名称一致。
- 本文的 migration 清单与 `migrations/` 中的版本和名称一致。

该脚本不校验 SQL 语义、可逆性、执行顺序中的业务影响或数据库当前状态。执行验证仍由 `go run ./cmd/migrate up` 覆盖。

## 当前清单

| 版本 | 名称 | 职责 |
|---|---|---|
| 000001 | bootstrap | 初始 bootstrap migration，占位并建立迁移基线 |
| 000002 | create_users | 创建 `users` 表和用户基础约束 |
| 000003 | create_community_schema | 增加平台 staff 字段，并创建社区、成员关系和社区申请 schema |
| 000004 | create_posts | 创建 `posts` 表、状态约束和帖子查询索引 |
| 000005 | create_comments | 创建 `comments` 表、父子评论关系和评论查询索引 |
| 000006 | create_post_votes | 创建帖子投票事实表和投票查询索引 |
| 000007 | create_moderation_schema | 创建内容举报和审核动作 schema |
| 000008 | create_notifications | 创建站内通知表和未读/已读查询索引 |
| 000009 | create_media_attachments | 创建图片附件元数据表和上传/归属查询索引 |
| 000010 | create_engagement_schema | 创建帖子保存、社区关注和评论投票事实表 |
| 000011 | create_effects_points | 创建评论效果目录、用户积分账户、评论效果记录和积分流水 |
| 000012 | add_notification_aggregation | 为通知增加点赞聚合键、聚合计数和最近 actor 字段 |
| 000013 | add_search_full_text_indexes | 为公开社区和 visible 帖子搜索增加 PostgreSQL full-text 表达式索引 |
| 000014 | create_community_rules | 创建社区规则表和按社区排序查询索引 |
| 000015 | create_content_refs | 创建帖子和评论结构化内容引用表 |
| 000016 | create_admin_management | 创建平台管理设置表和平台管理审计日志表 |
| 000017 | add_user_profile_fields | 为 `users` 表增加公开资料字段和基础长度/URL 约束 |
| 000018 | add_user_banner_url | 为 `users` 表增加公开主页背景图 URL 字段和 URL 约束 |

## 阶段 19 边界

阶段 19 只沉淀 migration 契约和清单校验，不新增 schema migration，不修改已有 SQL，不改变 migration runner 行为，也不进入新的业务产品语义。
