# HTTP API Schema 契约快照

本文记录当前 HTTP API 的请求和成功响应 JSON 结构。它是给前后端协作使用的轻量 schema 快照，不是 OpenAPI，不定义前端页面，也不改变任何接口、错误码或响应格式。

schema 字段清单、接口 schema 映射和请求必填字段清单需要通过以下脚本与 delivery 层 handler 结构、HTTP 路由契约保持同步：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-api-schema-doc.ps1
```

该脚本扫描 `internal/*/delivery/*http` 下非测试 Go 文件中的 JSON struct，并与本文的“Handler JSON 结构清单”比对 package、Go type 和 JSON 字段顺序；同时读取 `docs/contracts/http-api-contract.md` 的路由表，校验本文“接口 Schema 映射”覆盖所有当前路由、没有过期路由、request/success 中引用的 schema 真实存在，且成功状态码保持在 `200/201/204` 范围内。请求 struct 上的 `binding:"required"` 也必须同步记录在“请求必填字段清单”。它不生成 OpenAPI，不校验业务枚举、数值范围、嵌套类型语义、完整示例或错误响应全文。

## 全局约定

- 错误响应统一为 `{"error":{"code":"...","message":"..."}}`，错误码和 HTTP 状态码映射以 `docs/contracts/http-error-handling.md` 为准。
- 时间字段使用 Go `time.Time` 的 JSON 输出格式，前端按 RFC3339/RFC3339Nano 字符串处理。
- `omitempty` 字段在无值时可能缺省，不应由前端假设一定存在。
- `DELETE` 成功响应当前返回 `204 No Content`，没有 JSON body。
- `POST /api/v1/uploads/images` 使用 `multipart/form-data`，请求字段是 `file` 和可选 `alt_text`；它不使用 JSON 请求体。

## 接口 Schema 映射

| Method | Path | Request | Success | Status |
|---|---|---|---|---|
| GET | /healthz | none | `{"status":"ok"}` | 200 |
| GET | /uploads/*filepath | none | static file | 200 |
| POST | /api/v1/auth/register | `authhttp.registerRequest` | `authhttp.registerResponse` | 201 |
| POST | /api/v1/auth/login | `authhttp.loginRequest` | `authhttp.loginResponse` | 200 |
| GET | /api/v1/me | none | `userhttp.currentUserResponse` | 200 |
| GET | /api/v1/me/saved-posts | query | `posthttp.listCommunityPostsResponse` | 200 |
| GET | /api/v1/me/followed-communities | query | `communityhttp.listFollowedCommunitiesResponse` | 200 |
| GET | /api/v1/me/points | none | `effecthttp.getMyPointsResponse` | 200 |
| GET | /api/v1/users/:username | none | `userhttp.getPublicUserResponse` | 200 |
| GET | /api/v1/communities | none | `communityhttp.listCommunitiesResponse` | 200 |
| GET | /api/v1/communities/:slug | none | `communityhttp.getCommunityResponse` | 200 |
| GET | /api/v1/communities/:slug/manage | none | `communityhttp.getCommunityManageContextResponse` | 200 |
| GET | /api/v1/communities/:slug/manage/posts | query | `communityhttp.listCommunityManagePostsResponse` | 200 |
| GET | /api/v1/communities/:slug/manage/comments | query | `communityhttp.listCommunityManageCommentsResponse` | 200 |
| GET | /api/v1/communities/:slug/manage/reports | query | `communityhttp.listCommunityManageReportsResponse` | 200 |
| GET | /api/v1/communities/:slug/manage/members | query | `communityhttp.listCommunityMembersResponse` | 200 |
| POST | /api/v1/communities/:slug/follow | none | none | 204 |
| DELETE | /api/v1/communities/:slug/follow | none | none | 204 |
| POST | /api/v1/community-applications | `communityhttp.submitCommunityApplicationRequest` | `communityhttp.submitCommunityApplicationResponse` | 201 |
| GET | /api/v1/community-applications | query | `communityhttp.listCommunityApplicationsResponse` | 200 |
| GET | /api/v1/community-applications/:id | none | `communityhttp.getCommunityApplicationResponse` | 200 |
| POST | /api/v1/community-applications/:id/approve | none | `communityhttp.approveCommunityApplicationResponse` | 200 |
| POST | /api/v1/community-applications/:id/reject | `communityhttp.rejectCommunityApplicationRequest` | `communityhttp.rejectCommunityApplicationResponse` | 200 |
| POST | /api/v1/communities/:slug/posts | `posthttp.publishPostRequest` | `posthttp.publishPostResponse` | 201 |
| GET | /api/v1/communities/:slug/posts | query | `posthttp.listCommunityPostsResponse` | 200 |
| GET | /api/v1/posts | query | `posthttp.listCommunityPostsResponse` | 200 |
| GET | /api/v1/posts/:id | none | `posthttp.getPostResponse` | 200 |
| GET | /api/v1/users/:username/posts | query | `posthttp.listCommunityPostsResponse` | 200 |
| PATCH | /api/v1/posts/:id | `posthttp.updatePostRequest` | `posthttp.getPostResponse` | 200 |
| DELETE | /api/v1/posts/:id | none | none | 204 |
| POST | /api/v1/posts/:id/save | none | none | 204 |
| DELETE | /api/v1/posts/:id/save | none | none | 204 |
| POST | /api/v1/posts/:id/comments | `commenthttp.publishCommentRequest` | `commenthttp.publishCommentResponse` | 201 |
| GET | /api/v1/posts/:id/comments | query | `commenthttp.listPostCommentsResponse` | 200 |
| GET | /api/v1/users/:username/comments | query | `commenthttp.listUserCommentsResponse` | 200 |
| PATCH | /api/v1/comments/:id | `commenthttp.updateCommentRequest` | `commenthttp.publishCommentResponse` | 200 |
| DELETE | /api/v1/comments/:id | none | none | 204 |
| POST | /api/v1/comments/:id/effects | `effecthttp.applyCommentEffectRequest` | `effecthttp.applyCommentEffectResponse` | 201 |
| PUT | /api/v1/comments/:id/vote | `commenthttp.setCommentVoteRequest` | `commenthttp.setCommentVoteResponse` | 200 |
| DELETE | /api/v1/comments/:id/vote | none | none | 204 |
| PUT | /api/v1/posts/:id/vote | `votehttp.setPostVoteRequest` | `votehttp.setPostVoteResponse` | 200 |
| DELETE | /api/v1/posts/:id/vote | none | none | 204 |
| POST | /api/v1/posts/:id/reports | `moderationhttp.reportContentRequest` | `moderationhttp.reportContentResponse` | 201 |
| POST | /api/v1/comments/:id/reports | `moderationhttp.reportContentRequest` | `moderationhttp.reportContentResponse` | 201 |
| POST | /api/v1/posts/:id/moderation/remove | `moderationhttp.reportContentRequest` | `moderationhttp.removeContentResponse` | 200 |
| POST | /api/v1/comments/:id/moderation/remove | `moderationhttp.reportContentRequest` | `moderationhttp.removeContentResponse` | 200 |
| GET | /api/v1/moderation/reports | query | `moderationhttp.listReportsResponse` | 200 |
| GET | /api/v1/moderation/reports/:id | none | `moderationhttp.reportContentResponse` | 200 |
| POST | /api/v1/moderation/reports/:id/dismiss | none | `moderationhttp.reportContentResponse` | 200 |
| POST | /api/v1/moderation/reports/:id/remove-target | `moderationhttp.reportContentRequest` | `moderationhttp.removeContentResponse` | 200 |
| GET | /api/v1/search | query | `searchhttp.searchResponse` | 200 |
| GET | /api/v1/notifications/unread-summary | none | `notificationhttp.unreadSummaryResponse` | 200 |
| GET | /api/v1/notifications | query | `notificationhttp.listNotificationsResponse` | 200 |
| POST | /api/v1/notifications/:id/read | none | `notificationhttp.markNotificationReadResponse` | 200 |
| POST | /api/v1/notifications/read-all | none | `notificationhttp.markAllNotificationsReadResponse` | 200 |
| POST | /api/v1/uploads/images | multipart form | `mediahttp.uploadImageResponse` | 201 |
| POST | /api/v1/link-previews/resolve | `contentrefhttp.resolveLinkPreviewRequest` | `contentrefhttp.resolveLinkPreviewResponse` | 200 |
| POST | /api/v1/embeds/resolve | `contentrefhttp.resolveEmbedRequest` | `contentrefhttp.resolveEmbedResponse` | 200 |
| GET | /api/v1/effects/catalog | none | `effecthttp.listEffectsCatalogResponse` | 200 |

## 请求必填字段清单

本表只记录 handler JSON request struct 上通过 Gin tag 声明的 `binding:"required"` 字段。它不替代 domain/usecase 中的业务校验，例如 slug 格式、正文长度、投票值枚举或分页范围。

| Schema | Required JSON fields |
|---|---|
| `authhttp.loginRequest` | `username`, `password` |
| `authhttp.registerRequest` | `username`, `password` |
| `commenthttp.publishCommentRequest` | `body` |
| `commenthttp.setCommentVoteRequest` | `value` |
| `commenthttp.updateCommentRequest` | `body` |
| `communityhttp.rejectCommunityApplicationRequest` | `reject_reason` |
| `contentrefhttp.resolveEmbedRequest` | `url` |
| `contentrefhttp.resolveLinkPreviewRequest` | `url` |
| `effecthttp.applyCommentEffectRequest` | `effect_id` |
| `communityhttp.submitCommunityApplicationRequest` | `requested_slug`, `requested_name`, `reason` |
| `moderationhttp.reportContentRequest` | `reason` |
| `posthttp.publishPostRequest` | `title`, `body` |
| `posthttp.updatePostRequest` | `title`, `body` |
| `votehttp.setPostVoteRequest` | `value` |

## Handler JSON 结构清单

| Package | Go type | JSON fields |
|---|---|---|
| `authhttp` | `registerRequest` | `username`, `password` |
| `authhttp` | `registerResponse` | `access_token`, `token_type`, `expires_in`, `user` |
| `authhttp` | `userResponse` | `id`, `username`, `status`, `created_at` |
| `authhttp` | `loginRequest` | `username`, `password` |
| `authhttp` | `loginResponse` | `access_token`, `token_type`, `expires_in`, `user` |
| `userhttp` | `currentUserResponse` | `id`, `username`, `status`, `is_platform_staff`, `created_at` |
| `userhttp` | `publicUserResponse` | `id`, `username`, `display_name`, `avatar_url`, `headline`, `bio`, `badges`, `roles`, `status`, `stats`, `created_at` |
| `userhttp` | `publicUserStatsResponse` | `post_count`, `comment_count` |
| `userhttp` | `getPublicUserResponse` | `user` |
| `communityhttp` | `listCommunitiesResponse` | `communities` |
| `communityhttp` | `getCommunityResponse` | `community` |
| `communityhttp` | `getCommunityManageContextResponse` | `community` |
| `communityhttp` | `listFollowedCommunitiesResponse` | `communities`, `limit`, `offset` |
| `communityhttp` | `listCommunityMembersResponse` | `community`, `members`, `limit`, `offset` |
| `communityhttp` | `communityMemberResponse` | `user`, `role`, `status`, `created_at`, `updated_at` |
| `communityhttp` | `communityMemberUserResponse` | `id`, `username`, `display_name`, `avatar_url`, `headline`, `badges` |
| `communityhttp` | `listCommunityManagePostsResponse` | `community`, `posts`, `status`, `limit`, `offset` |
| `communityhttp` | `communityManagePostResponse` | `id`, `community_id`, `author_id`, `title`, `body_excerpt`, `status`, `created_at`, `updated_at` |
| `communityhttp` | `listCommunityManageCommentsResponse` | `community`, `comments`, `status`, `limit`, `offset` |
| `communityhttp` | `communityManageCommentResponse` | `id`, `post_id`, `author_id`, `parent_id`, `body_excerpt`, `status`, `created_at`, `updated_at` |
| `communityhttp` | `listCommunityManageReportsResponse` | `community`, `reports`, `status`, `limit`, `offset` |
| `communityhttp` | `communityManageReportResponse` | `id`, `target_type`, `post_id`, `comment_id`, `reporter_id`, `reason`, `status`, `reviewed_by`, `reviewed_at`, `target_preview`, `created_at`, `updated_at` |
| `communityhttp` | `communityManageReportTargetPreviewResponse` | `target_type`, `post_id`, `comment_id`, `author_id`, `status`, `title`, `body_excerpt`, `created_at`, `updated_at` |
| `communityhttp` | `submitCommunityApplicationRequest` | `requested_slug`, `requested_name`, `reason` |
| `communityhttp` | `rejectCommunityApplicationRequest` | `reject_reason` |
| `communityhttp` | `communityApplicationResponse` | `id`, `applicant_id`, `requested_slug`, `requested_name`, `reason`, `status`, `reviewed_by`, `reviewed_at`, `reject_reason`, `created_at`, `updated_at` |
| `communityhttp` | `listCommunityApplicationsResponse` | `applications`, `limit`, `offset` |
| `communityhttp` | `getCommunityApplicationResponse` | `application` |
| `communityhttp` | `submitCommunityApplicationResponse` | `application` |
| `communityhttp` | `approveCommunityApplicationResponse` | `application`, `community` |
| `communityhttp` | `rejectCommunityApplicationResponse` | `application` |
| `communityhttp` | `communityResponse` | `id`, `slug`, `name`, `description`, `avatar_url`, `banner_url`, `kind`, `status`, `visibility`, `member_count`, `post_count`, `viewer_is_following`, `viewer_role`, `viewer_permissions`, `created_at`, `updated_at` |
| `communityhttp` | `communityViewerPermissionsResponse` | `can_post`, `can_manage`, `can_moderate` |
| `posthttp` | `publishPostRequest` | `title`, `body`, `attachment_ids` |
| `posthttp` | `updatePostRequest` | `title`, `body`, `attachment_ids` |
| `posthttp` | `postResponse` | `id`, `community_id`, `author_id`, `title`, `body`, `body_excerpt`, `format`, `content_refs`, `status`, `community`, `author`, `upvote_count`, `downvote_count`, `comment_count`, `save_count`, `score`, `my_vote`, `is_saved`, `preview`, `viewer_permissions`, `created_at`, `updated_at`, `attachments` |
| `posthttp` | `contentRefResponse` | `kind`, `ref_id` |
| `posthttp` | `userSummaryResponse` | `id`, `username`, `display_name`, `avatar_url`, `headline`, `badges` |
| `posthttp` | `communitySummaryResponse` | `id`, `slug`, `name`, `description`, `avatar_url`, `banner_url`, `member_count`, `post_count`, `viewer_is_following`, `viewer_role`, `viewer_permissions` |
| `posthttp` | `communityViewerPermissionsResponse` | `can_post`, `can_manage`, `can_moderate` |
| `posthttp` | `postPreviewResponse` | `kind`, `image` |
| `posthttp` | `postPreviewImageResponse` | `url`, `width`, `height`, `mime_type`, `alt_text`, `size_bytes` |
| `posthttp` | `viewerPermissionsResponse` | `can_comment`, `can_vote`, `can_report`, `can_edit`, `can_delete`, `can_moderate` |
| `posthttp` | `attachmentResponse` | `id`, `kind`, `url`, `thumbnail_url`, `width`, `height`, `size_bytes`, `mime_type`, `alt_text`, `status`, `created_at` |
| `posthttp` | `publishPostResponse` | `post` |
| `posthttp` | `listCommunityPostsResponse` | `posts`, `limit`, `offset` |
| `posthttp` | `getPostResponse` | `post` |
| `commenthttp` | `publishCommentRequest` | `body`, `parent_id`, `attachment_ids` |
| `commenthttp` | `updateCommentRequest` | `body`, `attachment_ids` |
| `commenthttp` | `setCommentVoteRequest` | `value` |
| `commenthttp` | `commentResponse` | `id`, `post_id`, `author_id`, `parent_id`, `body`, `format`, `content_refs`, `author`, `status`, `depth`, `reply_count`, `has_more_replies`, `upvote_count`, `downvote_count`, `score`, `my_vote`, `viewer_permissions`, `children`, `created_at`, `updated_at`, `attachments` |
| `commenthttp` | `contentRefResponse` | `kind`, `ref_id` |
| `commenthttp` | `userSummaryResponse` | `id`, `username`, `display_name`, `avatar_url`, `headline`, `badges` |
| `commenthttp` | `viewerPermissionsResponse` | `can_comment`, `can_vote`, `can_report`, `can_edit`, `can_delete`, `can_moderate` |
| `commenthttp` | `attachmentResponse` | `id`, `kind`, `url`, `thumbnail_url`, `width`, `height`, `size_bytes`, `mime_type`, `alt_text`, `status`, `created_at` |
| `commenthttp` | `publishCommentResponse` | `comment` |
| `commenthttp` | `commentVoteResponse` | `comment_id`, `user_id`, `value`, `created_at`, `updated_at` |
| `commenthttp` | `setCommentVoteResponse` | `vote` |
| `commenthttp` | `listPostCommentsResponse` | `comments`, `view`, `sort`, `limit`, `offset`, `max_depth` |
| `commenthttp` | `listUserCommentsResponse` | `comments`, `limit`, `offset` |
| `votehttp` | `setPostVoteRequest` | `value` |
| `votehttp` | `postVoteResponse` | `post_id`, `user_id`, `value`, `created_at`, `updated_at` |
| `votehttp` | `setPostVoteResponse` | `vote` |
| `moderationhttp` | `reportContentRequest` | `reason` |
| `moderationhttp` | `contentReportResponse` | `id`, `target_type`, `post_id`, `comment_id`, `reporter_id`, `reason`, `status`, `reviewed_by`, `reviewed_at`, `target_preview`, `created_at`, `updated_at` |
| `moderationhttp` | `reportTargetPreviewResponse` | `target_type`, `post_id`, `comment_id`, `author_id`, `status`, `title`, `body_excerpt`, `created_at`, `updated_at` |
| `moderationhttp` | `reportContentResponse` | `report` |
| `moderationhttp` | `listReportsResponse` | `reports`, `limit`, `offset` |
| `moderationhttp` | `moderationActionResponse` | `id`, `target_type`, `post_id`, `comment_id`, `actor_id`, `action`, `reason`, `created_at` |
| `moderationhttp` | `removeContentResponse` | `action` |
| `searchhttp` | `searchResponse` | `query`, `scope`, `limit`, `offset`, `communities`, `posts` |
| `searchhttp` | `searchCommunityResponse` | `id`, `slug`, `name`, `description`, `kind`, `status`, `visibility`, `created_at`, `updated_at` |
| `searchhttp` | `searchPostResponse` | `id`, `community_id`, `community_slug`, `author_id`, `title`, `body_excerpt`, `status`, `created_at`, `updated_at` |
| `notificationhttp` | `notificationResponse` | `id`, `recipient_id`, `type`, `title`, `body`, `source_type`, `source_id`, `aggregate_count`, `last_actor_id`, `read_at`, `created_at`, `updated_at` |
| `notificationhttp` | `listNotificationsResponse` | `notifications`, `category`, `status`, `limit`, `offset` |
| `notificationhttp` | `unreadSummaryResponse` | `total`, `replies`, `mentions`, `likes`, `system` |
| `notificationhttp` | `markNotificationReadResponse` | `notification` |
| `notificationhttp` | `markAllNotificationsReadResponse` | `updated_count`, `read_at` |
| `mediahttp` | `attachmentResponse` | `id`, `kind`, `url`, `thumbnail_url`, `width`, `height`, `size_bytes`, `mime_type`, `alt_text`, `status`, `created_at` |
| `mediahttp` | `uploadImageResponse` | `attachment` |
| `contentrefhttp` | `resolveLinkPreviewRequest` | `url` |
| `contentrefhttp` | `resolveEmbedRequest` | `url` |
| `contentrefhttp` | `linkPreviewResponse` | `provider`, `url`, `canonical_url`, `host`, `title`, `description`, `image_url` |
| `contentrefhttp` | `resolveLinkPreviewResponse` | `preview` |
| `contentrefhttp` | `embedResponse` | `provider`, `url`, `canonical_url`, `embed_url`, `iframe_allowed` |
| `contentrefhttp` | `resolveEmbedResponse` | `embed` |
| `effecthttp` | `effectResponse` | `id`, `name`, `description`, `cost_points`, `asset_url`, `animation_key`, `is_active`, `created_at`, `updated_at` |
| `effecthttp` | `listEffectsCatalogResponse` | `effects` |
| `effecthttp` | `pointAccountResponse` | `user_id`, `balance`, `lifetime_earned`, `lifetime_spent`, `updated_at` |
| `effecthttp` | `getMyPointsResponse` | `points` |
| `effecthttp` | `applyCommentEffectRequest` | `effect_id` |
| `effecthttp` | `commentEffectResponse` | `id`, `comment_id`, `effect_id`, `user_id`, `points_spent`, `created_at` |
| `effecthttp` | `applyCommentEffectResponse` | `comment_effect`, `points` |

## 不在本快照内

- 不新增或改变业务接口。
- 不生成 OpenAPI 或前端客户端代码。
- 不校验请求字段的完整业务规则，例如 slug 格式、分页最大值、投票值枚举和正文长度；“请求必填字段清单”只覆盖 handler struct tag 中显式存在的 `binding:"required"`。
- 不校验错误响应消息全文。
- 不定义前端路由、页面、状态管理或组件模型。
