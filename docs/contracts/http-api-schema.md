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
| POST | /api/v1/auth/email-codes/register | `authhttp.emailCodeRequest` | `authhttp.emailCodeResponse` | 200 |
| POST | /api/v1/auth/register-with-email | `authhttp.registerWithEmailRequest` | `authhttp.loginResponse` | 201 |
| POST | /api/v1/auth/login | `authhttp.loginRequest` | `authhttp.loginResponse` | 200 |
| POST | /api/v1/auth/email-codes/login | `authhttp.emailCodeRequest` | `authhttp.emailCodeResponse` | 200 |
| POST | /api/v1/auth/login-with-email-code | `authhttp.loginWithEmailCodeRequest` | `authhttp.loginResponse` | 200 |
| POST | /api/v1/auth/email-codes/password-reset | `authhttp.emailCodeRequest` | `authhttp.emailCodeResponse` | 200 |
| POST | /api/v1/auth/password-reset | `authhttp.passwordResetRequest` | `authhttp.passwordUpdatedResponse` | 200 |
| GET | /api/v1/me | none | `userhttp.currentUserResponse` | 200 |
| PATCH | /api/v1/me/profile | `userhttp.updateProfileRequest` | `userhttp.updateProfileResponse` | 200 |
| GET | /api/v1/me/security | none | `authhttp.accountSecurityResponse` | 200 |
| POST | /api/v1/me/security/email-codes/change-email | `authhttp.changeEmailCodeRequest` | `authhttp.emailCodeResponse` | 200 |
| POST | /api/v1/me/security/email-codes/delete-account | `authhttp.deleteAccountCodeRequest` | `authhttp.emailCodeResponse` | 200 |
| PATCH | /api/v1/me/security/email | `authhttp.changeEmailRequest` | `authhttp.changeEmailResponse` | 200 |
| PATCH | /api/v1/me/security/password | `authhttp.changePasswordRequest` | `authhttp.passwordUpdatedResponse` | 200 |
| DELETE | /api/v1/me/account | `authhttp.deleteAccountRequest` | none | 204 |
| POST | /api/v1/auth/logout-all | none | none | 204 |
| GET | /api/v1/me/saved-posts | query | `posthttp.listCommunityPostsResponse` | 200 |
| GET | /api/v1/me/followed-communities | query | `communityhttp.listFollowedCommunitiesResponse` | 200 |
| GET | /api/v1/me/community-owner-transfers | query | `communityhttp.listCommunityOwnerTransfersResponse` | 200 |
| GET | /api/v1/me/followed-users | query | `userhttp.listFollowedUsersResponse` | 200 |
| GET | /api/v1/me/points | none | `effecthttp.getMyPointsResponse` | 200 |
| GET | /api/v1/me/point-transactions | query | `effecthttp.listMyPointTransactionsResponse` | 200 |
| GET | /api/v1/me/progression | none | `progressionhttp.getMyProgressionResponse` | 200 |
| GET | /api/v1/me/xp-events | query | `progressionhttp.listMyXPEventsResponse` | 200 |
| GET | /api/v1/me/titles | query | `progressionhttp.listMyTitlesResponse` | 200 |
| PATCH | /api/v1/me/title | `progressionhttp.setActiveTitleRequest` | `progressionhttp.setActiveTitleResponse` | 200 |
| GET | /api/v1/users/:username | none | `userhttp.getPublicUserResponse` | 200 |
| POST | /api/v1/users/:username/follow | none | none | 204 |
| DELETE | /api/v1/users/:username/follow | none | none | 204 |
| GET | /api/v1/communities | query | `communityhttp.listCommunitiesResponse` | 200 |
| GET | /api/v1/communities/:slug | none | `communityhttp.getCommunityResponse` | 200 |
| GET | /api/v1/communities/:slug/manage | none | `communityhttp.getCommunityManageContextResponse` | 200 |
| GET | /api/v1/communities/:slug/manage/posts | query | `communityhttp.listCommunityManagePostsResponse` | 200 |
| GET | /api/v1/communities/:slug/manage/comments | query | `communityhttp.listCommunityManageCommentsResponse` | 200 |
| GET | /api/v1/communities/:slug/manage/reports | query | `communityhttp.listCommunityManageReportsResponse` | 200 |
| GET | /api/v1/communities/:slug/manage/members | query | `communityhttp.listCommunityMembersResponse` | 200 |
| GET | /api/v1/communities/:slug/manage/settings | none | `communityhttp.getCommunityManageSettingsResponse` | 200 |
| PATCH | /api/v1/communities/:slug/manage/settings | `communityhttp.updateCommunityManageSettingsRequest` | `communityhttp.updateCommunityManageSettingsResponse` | 200 |
| GET | /api/v1/communities/:slug/manage/rules | none | `communityhttp.listCommunityRulesResponse` | 200 |
| POST | /api/v1/communities/:slug/manage/rules | `communityhttp.writeCommunityRuleRequest` | `communityhttp.createCommunityRuleResponse` | 201 |
| PATCH | /api/v1/communities/:slug/manage/rules/:rule_id | `communityhttp.writeCommunityRuleRequest` | `communityhttp.updateCommunityRuleResponse` | 200 |
| DELETE | /api/v1/communities/:slug/manage/rules/:rule_id | none | none | 204 |
| POST | /api/v1/communities/:slug/follow | none | none | 204 |
| DELETE | /api/v1/communities/:slug/follow | none | none | 204 |
| POST | /api/v1/community-applications | `communityhttp.submitCommunityApplicationRequest` | `communityhttp.submitCommunityApplicationResponse` | 201 |
| GET | /api/v1/community-applications | query | `communityhttp.listCommunityApplicationsResponse` | 200 |
| GET | /api/v1/community-applications/:id | none | `communityhttp.getCommunityApplicationResponse` | 200 |
| POST | /api/v1/community-applications/:id/approve | none | `communityhttp.approveCommunityApplicationResponse` | 200 |
| POST | /api/v1/community-applications/:id/reject | `communityhttp.rejectCommunityApplicationRequest` | `communityhttp.rejectCommunityApplicationResponse` | 200 |
| GET | /api/v1/admin/users | query | `adminhttp.listAdminUsersResponse` | 200 |
| PATCH | /api/v1/admin/users/:id | `adminhttp.updateAdminUserRequest` | `adminhttp.updateAdminUserResponse` | 200 |
| GET | /api/v1/admin/owner-transfer | none | `adminhttp.ownerTransferResponse` | 200 |
| POST | /api/v1/admin/owner-transfer | `adminhttp.createOwnerTransferRequest` | `adminhttp.ownerTransferResponse` | 201 |
| DELETE | /api/v1/admin/owner-transfer/:transfer_id | none | `adminhttp.ownerTransferResponse` | 200 |
| GET | /api/v1/admin/communities | query | `adminhttp.listAdminCommunitiesResponse` | 200 |
| PATCH | /api/v1/admin/communities/:id | `adminhttp.updateAdminCommunityStatusRequest` | `adminhttp.updateAdminCommunityStatusResponse` | 200 |
| GET | /api/v1/admin/effects | query | `adminhttp.listAdminEffectsResponse` | 200 |
| PATCH | /api/v1/admin/effects/:id | `adminhttp.updateAdminEffectRequest` | `adminhttp.updateAdminEffectResponse` | 200 |
| GET | /api/v1/admin/settings | none | `adminhttp.listAdminSettingsResponse` | 200 |
| PATCH | /api/v1/admin/settings/:key | `adminhttp.updateAdminSettingRequest` | `adminhttp.updateAdminSettingResponse` | 200 |
| GET | /api/v1/admin/audit-logs | query | `adminhttp.listAdminAuditLogsResponse` | 200 |
| GET | /api/v1/admin/point-transactions | query | `adminhttp.listAdminPointTransactionsResponse` | 200 |
| POST | /api/v1/admin/users/:id/points/adjust | `adminhttp.adjustAdminUserPointsRequest` | `adminhttp.adjustAdminUserPointsResponse` | 200 |
| POST | /api/v1/admin/users/:id/sanctions | `adminhttp.createAdminUserSanctionRequest` | `adminhttp.createAdminUserSanctionResponse` | 201 |
| GET | /api/v1/admin/users/:id/sanctions | query | `adminhttp.listAdminUserSanctionsResponse` | 200 |
| POST | /api/v1/admin/user-sanctions/:sanction_id/revoke | none | `adminhttp.revokeAdminUserSanctionResponse` | 200 |
| GET | /api/v1/admin/titles | query | `progressionhttp.listAdminTitlesResponse` | 200 |
| POST | /api/v1/admin/titles | `progressionhttp.createAdminTitleRequest` | `progressionhttp.createAdminTitleResponse` | 201 |
| PATCH | /api/v1/admin/titles/:id | `progressionhttp.updateAdminTitleRequest` | `progressionhttp.updateAdminTitleResponse` | 200 |
| GET | /api/v1/admin/users/:id/titles | query | `progressionhttp.listAdminUserTitleGrantsResponse` | 200 |
| POST | /api/v1/admin/users/:id/titles | `progressionhttp.grantAdminUserTitleRequest` | `progressionhttp.grantAdminUserTitleResponse` | 201 |
| DELETE | /api/v1/admin/users/:id/titles/:grant_id | none | `progressionhttp.revokeAdminUserTitleResponse` | 200 |
| GET | /api/v1/owner-transfer/:transfer_id | none | `adminhttp.ownerTransferResponse` | 200 |
| POST | /api/v1/owner-transfer/:transfer_id/accept | `adminhttp.acceptOwnerTransferRequest` | `adminhttp.ownerTransferResponse` | 200 |
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
| POST | /api/v1/communities/:slug/moderation/posts/:id/remove | `moderationhttp.reportContentRequest` | `moderationhttp.removeContentResponse` | 200 |
| POST | /api/v1/communities/:slug/moderation/comments/:id/remove | `moderationhttp.reportContentRequest` | `moderationhttp.removeContentResponse` | 200 |
| GET | /api/v1/admin/mod-queues | query | `moderationhttp.modQueueResponse` | 200 |
| POST | /api/v1/admin/mod-queues/actions | `moderationhttp.bulkActionRequest` | `moderationhttp.bulkActionResponse` | 200 |
| GET | /api/v1/communities/:slug/mod-queues | query | `moderationhttp.modQueueResponse` | 200 |
| POST | /api/v1/communities/:slug/mod-queues/actions | `moderationhttp.bulkActionRequest` | `moderationhttp.bulkActionResponse` | 200 |
| POST | /api/v1/communities/:slug/moderation/posts/:id/approve | `moderationhttp.singleActionRequest` | `moderationhttp.bulkActionResponse` | 200 |
| POST | /api/v1/communities/:slug/moderation/comments/:id/approve | `moderationhttp.singleActionRequest` | `moderationhttp.bulkActionResponse` | 200 |
| POST | /api/v1/communities/:slug/moderation/posts/:id/spam | `moderationhttp.singleActionRequest` | `moderationhttp.bulkActionResponse` | 200 |
| POST | /api/v1/communities/:slug/moderation/comments/:id/spam | `moderationhttp.singleActionRequest` | `moderationhttp.bulkActionResponse` | 200 |
| POST | /api/v1/communities/:slug/moderation/reports/:id/ignore | none | `moderationhttp.reportContentResponse` | 200 |
| POST | /api/v1/communities/:slug/moderation/posts/:id/lock | `moderationhttp.singleActionRequest` | `moderationhttp.bulkActionResponse` | 200 |
| POST | /api/v1/communities/:slug/moderation/posts/:id/pin | `moderationhttp.singleActionRequest` | `moderationhttp.bulkActionResponse` | 200 |
| POST | /api/v1/communities/:slug/moderation/posts/:id/mark-nsfw | `moderationhttp.singleActionRequest` | `moderationhttp.bulkActionResponse` | 200 |
| POST | /api/v1/communities/:slug/moderation/posts/:id/mark-spoiler | `moderationhttp.singleActionRequest` | `moderationhttp.bulkActionResponse` | 200 |
| POST | /api/v1/communities/:slug/moderation/posts/:id/flair | `moderationhttp.singleActionRequest` | `moderationhttp.bulkActionResponse` | 200 |
| GET | /api/v1/communities/:slug/moderation/removal-reasons | none | `moderationhttp.templateListResponse` | 200 |
| POST | /api/v1/communities/:slug/moderation/removal-reasons | `moderationhttp.templateRequest` | `moderationhttp.templateResponse` | 201 |
| PATCH | /api/v1/communities/:slug/moderation/removal-reasons/:id | `moderationhttp.templateRequest` | `moderationhttp.templateResponse` | 200 |
| DELETE | /api/v1/communities/:slug/moderation/removal-reasons/:id | none | none | 204 |
| POST | /api/v1/communities/:slug/moderation/removal-reasons/:id/apply | `moderationhttp.applyRemovalReasonRequest` | `moderationhttp.bulkActionResponse` | 200 |
| GET | /api/v1/communities/:slug/moderation/saved-responses | none | `moderationhttp.templateListResponse` | 200 |
| POST | /api/v1/communities/:slug/moderation/saved-responses | `moderationhttp.templateRequest` | `moderationhttp.templateResponse` | 201 |
| PATCH | /api/v1/communities/:slug/moderation/saved-responses/:id | `moderationhttp.templateRequest` | `moderationhttp.templateResponse` | 200 |
| DELETE | /api/v1/communities/:slug/moderation/saved-responses/:id | none | none | 204 |
| GET | /api/v1/communities/:slug/manage/banned-users | query | `moderationhttp.userStateListResponse` | 200 |
| POST | /api/v1/communities/:slug/manage/banned-users | `moderationhttp.userStateRequest` | `moderationhttp.userStateResponse` | 200 |
| DELETE | /api/v1/communities/:slug/manage/banned-users/:user_id | none | none | 204 |
| GET | /api/v1/communities/:slug/manage/muted-users | query | `moderationhttp.userStateListResponse` | 200 |
| POST | /api/v1/communities/:slug/manage/muted-users | `moderationhttp.userStateRequest` | `moderationhttp.userStateResponse` | 200 |
| DELETE | /api/v1/communities/:slug/manage/muted-users/:user_id | none | none | 204 |
| GET | /api/v1/communities/:slug/manage/approved-users | query | `moderationhttp.userStateListResponse` | 200 |
| POST | /api/v1/communities/:slug/manage/approved-users | `moderationhttp.userStateRequest` | `moderationhttp.userStateResponse` | 200 |
| DELETE | /api/v1/communities/:slug/manage/approved-users/:user_id | none | none | 204 |
| GET | /api/v1/communities/:slug/moderation/users/:user_id/profile | none | `moderationhttp.moderationUserProfileResponse` | 200 |
| GET | /api/v1/communities/:slug/moderation/users/:user_id/notes | query | `moderationhttp.moderatorNotesResponse` | 200 |
| POST | /api/v1/communities/:slug/moderation/users/:user_id/notes | `moderationhttp.moderatorNoteRequest` | `moderationhttp.moderatorNoteMutationResponse` | 201 |
| DELETE | /api/v1/communities/:slug/moderation/users/:user_id/notes/:note_id | none | none | 204 |
| GET | /api/v1/communities/:slug/moderation/logs | query | `moderationhttp.modLogsResponse` | 200 |
| GET | /api/v1/moderation/reports | query | `moderationhttp.listReportsResponse` | 200 |
| GET | /api/v1/moderation/reports/:id | none | `moderationhttp.reportContentResponse` | 200 |
| POST | /api/v1/moderation/reports/:id/dismiss | none | `moderationhttp.reportContentResponse` | 200 |
| POST | /api/v1/moderation/reports/:id/remove-target | `moderationhttp.reportContentRequest` | `moderationhttp.removeContentResponse` | 200 |
| GET | /api/v1/search | query | `searchhttp.searchResponse` | 200 |
| GET | /api/v1/notifications/unread-summary | none | `notificationhttp.unreadSummaryResponse` | 200 |
| GET | /api/v1/notifications | query | `notificationhttp.listNotificationsResponse` | 200 |
| POST | /api/v1/notifications/:id/read | none | `notificationhttp.markNotificationReadResponse` | 200 |
| POST | /api/v1/notifications/read-all | none | `notificationhttp.markAllNotificationsReadResponse` | 200 |
| GET | /api/v1/messages/summary | none | `messagehttp.summaryResponse` | 200 |
| GET | /api/v1/messages/conversations | query | `messagehttp.listConversationsResponse` | 200 |
| POST | /api/v1/messages/conversations | `messagehttp.startConversationRequest` | `messagehttp.conversationMutationResponse` | 201 |
| GET | /api/v1/messages/conversations/:id/messages | query | `messagehttp.listMessagesResponse` | 200 |
| POST | /api/v1/messages/conversations/:id/messages | `messagehttp.sendMessageRequest` | `messagehttp.conversationMutationResponse` | 201 |
| POST | /api/v1/messages/conversations/:id/read | none | `messagehttp.conversationMutationResponse` | 200 |
| POST | /api/v1/messages/conversations/:id/archive | none | `messagehttp.conversationMutationResponse` | 200 |
| DELETE | /api/v1/messages/conversations/:id/archive | none | `messagehttp.conversationMutationResponse` | 200 |
| POST | /api/v1/messages/requests/:id/accept | none | `messagehttp.conversationMutationResponse` | 200 |
| POST | /api/v1/messages/requests/:id/reject | none | `messagehttp.conversationMutationResponse` | 200 |
| POST | /api/v1/messages/:id/recall | none | `messagehttp.conversationMutationResponse` | 200 |
| DELETE | /api/v1/messages/:id | none | none | 204 |
| POST | /api/v1/messages/:id/report | `messagehttp.reportMessageRequest` | `messagehttp.reportResponse` | 201 |
| POST | /api/v1/users/:username/block | none | none | 204 |
| DELETE | /api/v1/users/:username/block | none | none | 204 |
| GET | /api/v1/me/privacy/messages | none | `messagehttp.privacyResponse` | 200 |
| PATCH | /api/v1/me/privacy/messages | `messagehttp.updatePrivacyRequest` | `messagehttp.privacyResponse` | 200 |
| POST | /api/v1/realtime/tickets | `messagehttp.realtimeTicketRequest` | `messagehttp.realtimeTicketResponse` | 201 |
| GET | /api/v1/realtime/messages | query | `messagehttp.realtimeHelloResponse` | 200 |
| POST | /api/v1/uploads/images | multipart form | `mediahttp.uploadImageResponse` | 201 |
| POST | /api/v1/link-previews/resolve | `contentrefhttp.resolveLinkPreviewRequest` | `contentrefhttp.resolveLinkPreviewResponse` | 200 |
| POST | /api/v1/embeds/resolve | `contentrefhttp.resolveEmbedRequest` | `contentrefhttp.resolveEmbedResponse` | 200 |
| GET | /api/v1/effects/catalog | none | `effecthttp.listEffectsCatalogResponse` | 200 |

## 请求必填字段清单

本表只记录 handler JSON request struct 上通过 Gin tag 声明的 `binding:"required"` 字段。它不替代 domain/usecase 中的业务校验，例如 slug 格式、正文长度、投票值枚举或分页范围。

| Schema | Required JSON fields |
|---|---|
| `authhttp.registerRequest` | `username`, `password` |
| `adminhttp.adjustAdminUserPointsRequest` | `delta`, `reason` |
| `adminhttp.createAdminUserSanctionRequest` | `type`, `duration`, `reason` |
| `progressionhttp.createAdminTitleRequest` | `name` |
| `progressionhttp.grantAdminUserTitleRequest` | `title_id` |
| `adminhttp.updateAdminCommunityStatusRequest` | `status` |
| `adminhttp.updateAdminEffectRequest` | `is_active` |
| `adminhttp.updateAdminSettingRequest` | `enabled` |
| `commenthttp.publishCommentRequest` | `body` |
| `commenthttp.setCommentVoteRequest` | `value` |
| `commenthttp.updateCommentRequest` | `body` |
| `communityhttp.rejectCommunityApplicationRequest` | `reject_reason` |
| `communityhttp.writeCommunityRuleRequest` | `title` |
| `contentrefhttp.resolveEmbedRequest` | `url` |
| `contentrefhttp.resolveLinkPreviewRequest` | `url` |
| `effecthttp.applyCommentEffectRequest` | `effect_id` |
| `communityhttp.submitCommunityApplicationRequest` | `requested_slug`, `requested_name`, `reason` |
| `messagehttp.startConversationRequest` | `target_username`, `message` |
| `messagehttp.sendMessageRequest` | `message` |
| `messagehttp.reportMessageRequest` | `reason` |
| `moderationhttp.reportContentRequest` | `reason` |
| `moderationhttp.bulkActionRequest` | `action` |
| `moderationhttp.templateRequest` | `title` |
| `moderationhttp.userStateRequest` | `user_id` |
| `moderationhttp.moderatorNoteRequest` | `body` |
| `posthttp.publishPostRequest` | `title`, `body` |
| `posthttp.updatePostRequest` | `title`, `body` |
| `votehttp.setPostVoteRequest` | `value` |

## Handler JSON 结构清单

| Package | Go type | JSON fields |
|---|---|---|
| `authhttp` | `registerRequest` | `username`, `password` |
| `authhttp` | `registerResponse` | `access_token`, `token_type`, `expires_in`, `user` |
| `authhttp` | `userResponse` | `id`, `username`, `status`, `email`, `email_verified`, `created_at` |
| `authhttp` | `loginRequest` | `identifier`, `username`, `password` |
| `authhttp` | `loginResponse` | `access_token`, `token_type`, `expires_in`, `user` |
| `authhttp` | `emailCodeRequest` | `email` |
| `authhttp` | `emailCodeResponse` | `email`, `purpose`, `expires_in`, `resend_after` |
| `authhttp` | `registerWithEmailRequest` | `email`, `code`, `username`, `password` |
| `authhttp` | `loginWithEmailCodeRequest` | `email`, `code` |
| `authhttp` | `passwordResetRequest` | `email`, `code`, `new_password` |
| `authhttp` | `accountSecurityResponse` | `email`, `email_verified`, `email_verified_at`, `password_set`, `last_login_at`, `created_at` |
| `authhttp` | `changeEmailCodeRequest` | `new_email` |
| `authhttp` | `changeEmailRequest` | `new_email`, `code` |
| `authhttp` | `changeEmailResponse` | `email`, `email_verified`, `email_verified_at` |
| `authhttp` | `changePasswordRequest` | `current_password`, `new_password` |
| `authhttp` | `passwordUpdatedResponse` | `updated` |
| `authhttp` | `deleteAccountCodeRequest` | `email` |
| `authhttp` | `deleteAccountRequest` | `code`, `current_password`, `confirmation` |
| `userhttp` | `currentUserResponse` | `id`, `username`, `status`, `is_platform_staff`, `created_at` |
| `userhttp` | `publicUserResponse` | `id`, `username`, `display_name`, `avatar_url`, `banner_url`, `headline`, `bio`, `badges`, `roles`, `status`, `stats`, `progression`, `dm_capability`, `viewer_is_following`, `created_at` |
| `userhttp` | `publicUserDMCapabilityResponse` | `can_start`, `requires_request`, `reason`, `direct_conversation_id`, `viewer_relation` |
| `userhttp` | `publicUserProgressionResponse` | `level`, `level_name`, `xp_total`, `current_level_xp`, `next_level_xp`, `level_progress`, `active_title`, `titles_count` |
| `userhttp` | `publicUserTitleResponse` | `grant_id`, `title_id`, `name`, `scope_type`, `scope_id` |
| `userhttp` | `publicUserStatsResponse` | `post_count`, `comment_count`, `follower_count`, `following_count` |
| `userhttp` | `getPublicUserResponse` | `user` |
| `userhttp` | `listFollowedUsersResponse` | `users`, `limit`, `offset`, `next_offset`, `has_more` |
| `userhttp` | `updateProfileRequest` | `display_name`, `avatar_url`, `banner_url`, `headline`, `bio` |
| `userhttp` | `updateProfileResponse` | `user` |
| `communityhttp` | `listCommunitiesResponse` | `communities`, `limit`, `offset`, `next_offset`, `has_more` |
| `communityhttp` | `getCommunityResponse` | `community` |
| `communityhttp` | `getCommunityManageContextResponse` | `community` |
| `communityhttp` | `getCommunityManageSettingsResponse` | `community`, `settings` |
| `communityhttp` | `updateCommunityManageSettingsResponse` | `community`, `settings` |
| `communityhttp` | `listFollowedCommunitiesResponse` | `communities`, `limit`, `offset`, `next_offset`, `has_more` |
| `communityhttp` | `listCommunityOwnerTransfersResponse` | `transfers`, `status`, `limit`, `offset`, `next_offset`, `has_more` |
| `communityhttp` | `communityOwnerTransferListItemResponse` | `id`, `community_id`, `community`, `from_user_id`, `from_username`, `from_display_name`, `to_user_id`, `to_username`, `to_display_name`, `status`, `created_at`, `updated_at`, `expires_at`, `accepted_at`, `cancelled_at`, `viewer_is_target`, `viewer_can_cancel`, `platform_owner_override` |
| `communityhttp` | `listCommunityMembersResponse` | `community`, `members`, `limit`, `offset`, `next_offset`, `has_more` |
| `communityhttp` | `communityMemberResponse` | `user`, `role`, `status`, `created_at`, `updated_at` |
| `communityhttp` | `communityMemberUserResponse` | `id`, `username`, `display_name`, `avatar_url`, `headline`, `badges` |
| `communityhttp` | `listCommunityManagePostsResponse` | `community`, `posts`, `status`, `limit`, `offset`, `next_offset`, `has_more` |
| `communityhttp` | `communityManagePostResponse` | `id`, `community_id`, `author_id`, `title`, `body_excerpt`, `status`, `created_at`, `updated_at` |
| `communityhttp` | `listCommunityManageCommentsResponse` | `community`, `comments`, `status`, `limit`, `offset`, `next_offset`, `has_more` |
| `communityhttp` | `communityManageCommentResponse` | `id`, `post_id`, `author_id`, `parent_id`, `body_excerpt`, `status`, `created_at`, `updated_at` |
| `communityhttp` | `listCommunityManageReportsResponse` | `community`, `reports`, `status`, `limit`, `offset`, `next_offset`, `has_more` |
| `communityhttp` | `communityManageReportResponse` | `id`, `target_type`, `post_id`, `comment_id`, `reporter_id`, `reason`, `status`, `reviewed_by`, `reviewed_at`, `target_preview`, `created_at`, `updated_at` |
| `communityhttp` | `communityManageReportTargetPreviewResponse` | `target_type`, `post_id`, `comment_id`, `author_id`, `status`, `title`, `body_excerpt`, `created_at`, `updated_at` |
| `communityhttp` | `listCommunityRulesResponse` | `community`, `rules` |
| `communityhttp` | `createCommunityRuleResponse` | `community`, `rule` |
| `communityhttp` | `updateCommunityRuleResponse` | `community`, `rule` |
| `communityhttp` | `communitySettingsResponse` | `name`, `description`, `avatar_url`, `banner_url`, `updated_at` |
| `communityhttp` | `communityRuleResponse` | `id`, `community_id`, `title`, `body`, `position`, `created_by`, `updated_by`, `created_at`, `updated_at` |
| `communityhttp` | `updateCommunityManageSettingsRequest` | `name`, `description`, `avatar_url`, `banner_url` |
| `communityhttp` | `writeCommunityRuleRequest` | `title`, `body`, `position` |
| `communityhttp` | `submitCommunityApplicationRequest` | `requested_slug`, `requested_name`, `reason` |
| `communityhttp` | `rejectCommunityApplicationRequest` | `reject_reason` |
| `communityhttp` | `communityApplicationResponse` | `id`, `applicant_id`, `requested_slug`, `requested_name`, `reason`, `status`, `reviewed_by`, `reviewed_at`, `reject_reason`, `created_at`, `updated_at` |
| `communityhttp` | `listCommunityApplicationsResponse` | `applications`, `limit`, `offset`, `next_offset`, `has_more` |
| `communityhttp` | `getCommunityApplicationResponse` | `application` |
| `communityhttp` | `submitCommunityApplicationResponse` | `application` |
| `communityhttp` | `approveCommunityApplicationResponse` | `application`, `community` |
| `communityhttp` | `rejectCommunityApplicationResponse` | `application` |
| `communityhttp` | `communityResponse` | `id`, `slug`, `name`, `description`, `avatar_url`, `banner_url`, `kind`, `status`, `visibility`, `member_count`, `post_count`, `viewer_is_following`, `viewer_role`, `viewer_permissions`, `created_at`, `updated_at` |
| `communityhttp` | `communityViewerPermissionsResponse` | `can_post`, `can_manage`, `can_moderate`, `platform_owner_override` |
| `adminhttp` | `adminUserResponse` | `id`, `username`, `status`, `is_platform_staff`, `platform_role`, `created_at`, `updated_at` |
| `adminhttp` | `listAdminUsersResponse` | `users`, `status`, `q`, `limit`, `offset`, `next_offset`, `has_more` |
| `adminhttp` | `updateAdminUserRequest` | `status`, `is_platform_staff` |
| `adminhttp` | `updateAdminUserResponse` | `user` |
| `adminhttp` | `createOwnerTransferRequest` | `target_user_id`, `previous_owner_role`, `reason`, `current_password` |
| `adminhttp` | `acceptOwnerTransferRequest` | `current_password` |
| `adminhttp` | `ownerTransferResponse` | `transfer` |
| `adminhttp` | `adminOwnerTransferResponse` | `id`, `status`, `initiated_by_id`, `initiated_by_username`, `target_user_id`, `target_username`, `previous_owner_role`, `reason`, `created_at`, `updated_at`, `expires_at`, `accepted_at`, `cancelled_at` |
| `adminhttp` | `adminCommunityResponse` | `id`, `slug`, `name`, `description`, `kind`, `status`, `visibility`, `created_by`, `created_at`, `updated_at` |
| `adminhttp` | `listAdminCommunitiesResponse` | `communities`, `status`, `q`, `limit`, `offset`, `next_offset`, `has_more` |
| `adminhttp` | `updateAdminCommunityStatusRequest` | `status` |
| `adminhttp` | `updateAdminCommunityStatusResponse` | `community` |
| `adminhttp` | `adminEffectResponse` | `id`, `name`, `description`, `cost_points`, `asset_url`, `animation_key`, `is_active`, `created_at`, `updated_at` |
| `adminhttp` | `listAdminEffectsResponse` | `effects`, `active`, `limit`, `offset`, `next_offset`, `has_more` |
| `adminhttp` | `updateAdminEffectRequest` | `is_active` |
| `adminhttp` | `updateAdminEffectResponse` | `effect` |
| `adminhttp` | `adminSettingResponse` | `key`, `enabled`, `updated_by`, `updated_at` |
| `adminhttp` | `listAdminSettingsResponse` | `settings` |
| `adminhttp` | `updateAdminSettingRequest` | `enabled` |
| `adminhttp` | `updateAdminSettingResponse` | `setting` |
| `adminhttp` | `adminAuditLogResponse` | `id`, `actor_id`, `action`, `target_type`, `target_id`, `before`, `after`, `created_at` |
| `adminhttp` | `listAdminAuditLogsResponse` | `audit_logs`, `q`, `limit`, `offset`, `next_offset`, `has_more` |
| `adminhttp` | `adminPointAccountResponse` | `user_id`, `balance`, `lifetime_earned`, `lifetime_spent`, `updated_at` |
| `adminhttp` | `adminPointTransactionResponse` | `id`, `user_id`, `delta`, `balance_after`, `reason`, `source_type`, `source_id`, `created_at` |
| `adminhttp` | `listAdminPointTransactionsResponse` | `transactions`, `limit`, `offset`, `next_offset`, `has_more` |
| `adminhttp` | `adjustAdminUserPointsRequest` | `delta`, `reason` |
| `adminhttp` | `adjustAdminUserPointsResponse` | `account`, `transaction` |
| `adminhttp` | `adminUserSanctionResponse` | `id`, `user_id`, `type`, `status`, `reason`, `created_by`, `starts_at`, `expires_at`, `revoked_by`, `revoked_at`, `created_at`, `updated_at` |
| `adminhttp` | `createAdminUserSanctionRequest` | `type`, `duration`, `reason` |
| `adminhttp` | `createAdminUserSanctionResponse` | `sanction` |
| `adminhttp` | `listAdminUserSanctionsResponse` | `sanctions`, `limit`, `offset`, `next_offset`, `has_more` |
| `adminhttp` | `revokeAdminUserSanctionResponse` | `sanction` |
| `posthttp` | `publishPostRequest` | `title`, `body`, `attachment_ids`, `content_refs` |
| `posthttp` | `updatePostRequest` | `title`, `body`, `attachment_ids`, `content_refs` |
| `posthttp` | `postResponse` | `id`, `community_id`, `author_id`, `title`, `body`, `body_excerpt`, `format`, `content_refs`, `status`, `is_locked`, `is_pinned`, `is_nsfw`, `is_spoiler`, `flair_text`, `community`, `author`, `upvote_count`, `downvote_count`, `comment_count`, `save_count`, `score`, `my_vote`, `is_saved`, `preview`, `viewer_permissions`, `created_at`, `updated_at`, `attachments` |
| `posthttp` | `contentRefResponse` | `kind`, `ref_id` |
| `posthttp` | `contentRefRequest` | `kind`, `ref_id` |
| `posthttp` | `userSummaryResponse` | `id`, `username`, `display_name`, `avatar_url`, `headline`, `badges` |
| `posthttp` | `communitySummaryResponse` | `id`, `slug`, `name`, `description`, `avatar_url`, `banner_url`, `member_count`, `post_count`, `viewer_is_following`, `viewer_role`, `viewer_permissions` |
| `posthttp` | `communityViewerPermissionsResponse` | `can_post`, `can_manage`, `can_moderate`, `platform_owner_override` |
| `posthttp` | `postPreviewResponse` | `kind`, `image` |
| `posthttp` | `postPreviewImageResponse` | `url`, `width`, `height`, `mime_type`, `alt_text`, `size_bytes` |
| `posthttp` | `viewerPermissionsResponse` | `can_comment`, `can_vote`, `can_report`, `can_edit`, `can_delete`, `can_moderate` |
| `posthttp` | `attachmentResponse` | `id`, `kind`, `url`, `thumbnail_url`, `width`, `height`, `size_bytes`, `mime_type`, `alt_text`, `status`, `created_at` |
| `posthttp` | `publishPostResponse` | `post` |
| `posthttp` | `listCommunityPostsResponse` | `posts`, `limit`, `offset`, `next_offset`, `has_more` |
| `posthttp` | `getPostResponse` | `post` |
| `commenthttp` | `publishCommentRequest` | `body`, `parent_id`, `attachment_ids`, `content_refs` |
| `commenthttp` | `updateCommentRequest` | `body`, `attachment_ids`, `content_refs` |
| `commenthttp` | `setCommentVoteRequest` | `value` |
| `commenthttp` | `commentResponse` | `id`, `post_id`, `author_id`, `parent_id`, `body`, `format`, `content_refs`, `author`, `status`, `depth`, `reply_count`, `has_more_replies`, `upvote_count`, `downvote_count`, `score`, `my_vote`, `viewer_permissions`, `children`, `created_at`, `updated_at`, `attachments`, `effects` |
| `commenthttp` | `contentRefResponse` | `kind`, `ref_id` |
| `commenthttp` | `contentRefRequest` | `kind`, `ref_id` |
| `commenthttp` | `userSummaryResponse` | `id`, `username`, `display_name`, `avatar_url`, `headline`, `badges` |
| `commenthttp` | `viewerPermissionsResponse` | `can_comment`, `can_vote`, `can_report`, `can_edit`, `can_delete`, `can_moderate` |
| `commenthttp` | `commentEffectResponse` | `id`, `effect_id`, `name`, `asset_url`, `animation_key`, `applied_by_user`, `points_spent`, `created_at` |
| `commenthttp` | `attachmentResponse` | `id`, `kind`, `url`, `thumbnail_url`, `width`, `height`, `size_bytes`, `mime_type`, `alt_text`, `status`, `created_at` |
| `commenthttp` | `publishCommentResponse` | `comment` |
| `commenthttp` | `commentVoteResponse` | `comment_id`, `user_id`, `value`, `created_at`, `updated_at` |
| `commenthttp` | `setCommentVoteResponse` | `vote` |
| `commenthttp` | `listPostCommentsResponse` | `comments`, `view`, `sort`, `limit`, `offset`, `next_offset`, `has_more`, `max_depth` |
| `commenthttp` | `listUserCommentsResponse` | `comments`, `limit`, `offset`, `next_offset`, `has_more` |
| `votehttp` | `setPostVoteRequest` | `value` |
| `votehttp` | `postVoteResponse` | `post_id`, `user_id`, `value`, `created_at`, `updated_at` |
| `votehttp` | `setPostVoteResponse` | `vote` |
| `moderationhttp` | `reportContentRequest` | `reason` |
| `moderationhttp` | `contentReportResponse` | `id`, `target_type`, `post_id`, `comment_id`, `reporter_id`, `reason`, `status`, `reviewed_by`, `reviewed_at`, `target_preview`, `created_at`, `updated_at` |
| `moderationhttp` | `reportTargetPreviewResponse` | `target_type`, `post_id`, `comment_id`, `author_id`, `status`, `title`, `body_excerpt`, `created_at`, `updated_at` |
| `moderationhttp` | `reportContentResponse` | `report` |
| `moderationhttp` | `listReportsResponse` | `reports`, `limit`, `offset`, `next_offset`, `has_more` |
| `moderationhttp` | `moderationActionResponse` | `id`, `target_type`, `post_id`, `comment_id`, `actor_id`, `action`, `reason`, `created_at` |
| `moderationhttp` | `removeContentResponse` | `action` |
| `moderationhttp` | `modQueueResponse` | `items`, `queue`, `limit`, `offset`, `next_offset`, `has_more` |
| `moderationhttp` | `modQueueItemResponse` | `id`, `target_type`, `target_id`, `post_id`, `community_id`, `community_slug`, `author_id`, `report_count`, `queue`, `status`, `preview`, `created_at`, `updated_at` |
| `moderationhttp` | `moderationTargetRequest` | `target_type`, `target_id` |
| `moderationhttp` | `bulkActionRequest` | `action`, `target_type`, `target_ids`, `targets`, `reason`, `removal_reason_id`, `notify_author`, `confirm`, `value`, `flair_text` |
| `moderationhttp` | `applyRemovalReasonRequest` | `target_type`, `target_ids`, `targets`, `reason`, `notify_author`, `confirm` |
| `moderationhttp` | `singleActionRequest` | `reason`, `removal_reason_id`, `notify_author`, `confirm`, `value`, `flair_text` |
| `moderationhttp` | `bulkActionResponse` | `results` |
| `moderationhttp` | `bulkActionItemResponse` | `target_type`, `target_id`, `ok`, `action`, `error_code`, `error_message` |
| `moderationhttp` | `templateListResponse` | `items` |
| `moderationhttp` | `templateResponse` | `item` |
| `moderationhttp` | `moderationTemplateResponse` | `id`, `community_id`, `title`, `body`, `rule_id`, `is_active`, `position`, `created_by`, `updated_by`, `created_at`, `updated_at` |
| `moderationhttp` | `templateRequest` | `title`, `body`, `rule_id`, `position` |
| `moderationhttp` | `userStateRequest` | `user_id`, `reason`, `expires_at` |
| `moderationhttp` | `userStateListResponse` | `users`, `kind`, `limit`, `offset`, `next_offset`, `has_more` |
| `moderationhttp` | `userStateResponse` | `user` |
| `moderationhttp` | `communityUserStateResponse` | `id`, `community_id`, `user_id`, `username`, `display_name`, `avatar_url`, `kind`, `reason`, `expires_at`, `created_by`, `updated_by`, `created_at`, `updated_at` |
| `moderationhttp` | `moderationUserProfileResponse` | `user_id`, `username`, `display_name`, `avatar_url`, `headline`, `status`, `post_count`, `comment_count`, `report_count`, `removed_count`, `is_banned`, `is_muted`, `is_approved`, `recent_notes` |
| `moderationhttp` | `moderatorNoteResponse` | `id`, `community_id`, `user_id`, `author_id`, `body`, `created_at` |
| `moderationhttp` | `moderatorNotesResponse` | `notes`, `limit`, `offset`, `next_offset`, `has_more` |
| `moderationhttp` | `moderatorNoteRequest` | `body` |
| `moderationhttp` | `moderatorNoteMutationResponse` | `note` |
| `moderationhttp` | `modLogsResponse` | `logs`, `limit`, `offset`, `next_offset`, `has_more` |
| `moderationhttp` | `modLogResponse` | `id`, `community_id`, `actor_id`, `action`, `target_type`, `target_id`, `batch_id`, `before`, `after`, `metadata`, `created_at` |
| `searchhttp` | `searchResponse` | `query`, `scope`, `limit`, `offset`, `next_offset`, `has_more`, `communities`, `posts`, `users` |
| `searchhttp` | `searchCommunityResponse` | `id`, `slug`, `name`, `description`, `kind`, `status`, `visibility`, `created_at`, `updated_at` |
| `searchhttp` | `searchPostResponse` | `id`, `community_id`, `community_slug`, `author_id`, `title`, `body_excerpt`, `status`, `created_at`, `updated_at` |
| `searchhttp` | `searchUserResponse` | `id`, `username`, `display_name`, `avatar_url`, `headline`, `bio_excerpt`, `status`, `created_at`, `updated_at` |
| `notificationhttp` | `notificationResponse` | `id`, `recipient_id`, `type`, `title`, `body`, `source_type`, `source_id`, `aggregate_count`, `last_actor_id`, `actor`, `last_actor`, `context`, `read_at`, `created_at`, `updated_at` |
| `notificationhttp` | `actorResponse` | `id`, `username`, `display_name`, `avatar_url` |
| `notificationhttp` | `communityContextResponse` | `id`, `slug`, `name` |
| `notificationhttp` | `notificationContextResponse` | `post_id`, `comment_id`, `permalink`, `post_title`, `comment_excerpt`, `comment_depth`, `community` |
| `notificationhttp` | `listNotificationsResponse` | `notifications`, `category`, `status`, `limit`, `offset`, `next_offset`, `has_more` |
| `notificationhttp` | `unreadSummaryResponse` | `total`, `replies`, `mentions`, `likes`, `system` |
| `notificationhttp` | `markNotificationReadResponse` | `notification` |
| `notificationhttp` | `markAllNotificationsReadResponse` | `updated_count`, `read_at` |
| `messagehttp` | `summaryResponse` | `unread_total`, `request_count`, `unread_conversations`, `online_status_enabled` |
| `messagehttp` | `listConversationsResponse` | `conversations`, `box`, `limit`, `offset`, `next_offset`, `has_more` |
| `messagehttp` | `conversationMutationResponse` | `conversation`, `message` |
| `messagehttp` | `conversationResponse` | `id`, `box`, `request_id`, `request_status`, `participant`, `last_message`, `unread_count`, `updated_at`, `pinned`, `muted`, `archived`, `blocked`, `can_send`, `disable_reason`, `peer_online_status_visible`, `peer_online` |
| `messagehttp` | `messageSummaryResponse` | `id`, `type`, `text`, `status`, `created_at` |
| `messagehttp` | `listMessagesResponse` | `messages`, `limit`, `has_more`, `next_before` |
| `messagehttp` | `messageResponse` | `id`, `conversation_id`, `sender`, `type`, `body`, `image_url`, `share`, `status`, `created_at`, `updated_at`, `recalled_at`, `viewer_deleted` |
| `messagehttp` | `shareSnapshotResponse` | `share_type`, `share_id`, `title`, `summary`, `thumbnail_url`, `target_url`, `snapshot_created_at` |
| `messagehttp` | `userSummaryResponse` | `id`, `username`, `display_name`, `avatar_url`, `status` |
| `messagehttp` | `startConversationRequest` | `target_username`, `message` |
| `messagehttp` | `messageDraftRequest` | `type`, `body`, `image_url`, `share` |
| `messagehttp` | `shareSnapshotRequest` | `share_type`, `share_id`, `title`, `summary`, `thumbnail_url`, `target_url`, `snapshot_created_at` |
| `messagehttp` | `sendMessageRequest` | `message` |
| `messagehttp` | `reportMessageRequest` | `reason` |
| `messagehttp` | `privacyResponse` | `allow_messages`, `online_status_enabled`, `updated_at` |
| `messagehttp` | `updatePrivacyRequest` | `allow_messages`, `online_status_enabled` |
| `messagehttp` | `realtimeTicketRequest` | `last_event_id` |
| `messagehttp` | `realtimeTicketResponse` | `ticket`, `expires_at` |
| `messagehttp` | `realtimeEventResponse` | `id`, `type`, `conversation_id`, `payload`, `created_at` |
| `messagehttp` | `realtimeHelloResponse` | `type`, `events` |
| `messagehttp` | `reportResponse` | `report` |
| `messagehttp` | `messageReportResponse` | `id`, `conversation_id`, `message_id`, `reported_user_id`, `reason`, `context_before`, `context_after`, `created_at` |
| `mediahttp` | `attachmentResponse` | `id`, `kind`, `url`, `thumbnail_url`, `width`, `height`, `size_bytes`, `mime_type`, `alt_text`, `status`, `created_at` |
| `mediahttp` | `uploadImageResponse` | `attachment` |
| `contentrefhttp` | `resolveLinkPreviewRequest` | `url` |
| `contentrefhttp` | `resolveEmbedRequest` | `url` |
| `contentrefhttp` | `linkPreviewResponse` | `provider`, `url`, `canonical_url`, `host`, `title`, `description`, `image_url` |
| `contentrefhttp` | `resolveLinkPreviewResponse` | `preview` |
| `contentrefhttp` | `embedResponse` | `id`, `provider`, `provider_ref`, `url`, `canonical_url`, `embed_url`, `iframe_allowed`, `title`, `description`, `image_url`, `author_name`, `status` |
| `contentrefhttp` | `resolveEmbedResponse` | `embed` |
| `effecthttp` | `effectResponse` | `id`, `name`, `description`, `cost_points`, `asset_url`, `animation_key`, `is_active`, `created_at`, `updated_at` |
| `effecthttp` | `listEffectsCatalogResponse` | `effects` |
| `effecthttp` | `pointAccountResponse` | `user_id`, `balance`, `lifetime_earned`, `lifetime_spent`, `updated_at` |
| `effecthttp` | `getMyPointsResponse` | `points` |
| `effecthttp` | `pointTransactionResponse` | `id`, `user_id`, `delta`, `balance_after`, `reason`, `source_type`, `source_id`, `created_at` |
| `effecthttp` | `listMyPointTransactionsResponse` | `transactions`, `limit`, `offset`, `next_offset`, `has_more` |
| `effecthttp` | `applyCommentEffectRequest` | `effect_id` |
| `effecthttp` | `commentEffectResponse` | `id`, `comment_id`, `effect_id`, `user_id`, `points_spent`, `created_at` |
| `effecthttp` | `applyCommentEffectResponse` | `comment_effect`, `points` |
| `progressionhttp` | `progressionResponse` | `user_id`, `xp_total`, `level`, `level_name`, `current_level_xp`, `next_level_xp`, `level_progress`, `active_title`, `titles_count`, `updated_at` |
| `progressionhttp` | `getMyProgressionResponse` | `progression` |
| `progressionhttp` | `xpEventResponse` | `id`, `user_id`, `delta`, `xp_total_after`, `reason`, `source_type`, `source_id`, `actor_id`, `created_at` |
| `progressionhttp` | `listMyXPEventsResponse` | `events`, `limit`, `offset`, `next_offset`, `has_more` |
| `progressionhttp` | `titleSummaryResponse` | `grant_id`, `title_id`, `name`, `scope_type`, `scope_id` |
| `progressionhttp` | `titleResponse` | `id`, `name`, `description`, `scope_type`, `scope_id`, `is_active`, `created_by`, `created_at`, `updated_at` |
| `progressionhttp` | `titleGrantResponse` | `id`, `user_id`, `title`, `granted_by`, `reason`, `expires_at`, `revoked_at`, `created_at` |
| `progressionhttp` | `listMyTitlesResponse` | `titles`, `limit`, `offset`, `next_offset`, `has_more` |
| `progressionhttp` | `setActiveTitleRequest` | `title_grant_id` |
| `progressionhttp` | `setActiveTitleResponse` | `progression` |
| `progressionhttp` | `listAdminTitlesResponse` | `titles`, `limit`, `offset`, `next_offset`, `has_more` |
| `progressionhttp` | `createAdminTitleRequest` | `name`, `description`, `scope_type`, `scope_id` |
| `progressionhttp` | `createAdminTitleResponse` | `title` |
| `progressionhttp` | `updateAdminTitleRequest` | `name`, `description`, `is_active` |
| `progressionhttp` | `updateAdminTitleResponse` | `title` |
| `progressionhttp` | `listAdminUserTitleGrantsResponse` | `titles`, `limit`, `offset`, `next_offset`, `has_more` |
| `progressionhttp` | `grantAdminUserTitleRequest` | `title_id`, `reason`, `expires_at` |
| `progressionhttp` | `grantAdminUserTitleResponse` | `grant` |
| `progressionhttp` | `revokeAdminUserTitleResponse` | `grant` |

## 不在本快照内

- 不新增或改变业务接口。
- 不生成 OpenAPI 或前端客户端代码。
- 不校验请求字段的完整业务规则，例如 slug 格式、分页最大值、投票值枚举和正文长度；“请求必填字段清单”只覆盖 handler struct tag 中显式存在的 `binding:"required"`。
- 不校验错误响应消息全文。
- 不定义前端路由、页面、状态管理或组件模型。

## Stage 49 Community Member Governance Addendum

### Route Schema Mappings

| Method | Path | Request | Success | Status |
|---|---|---|---|---|
| POST | /api/v1/communities/:slug/manage/moderators | `communityhttp.writeCommunityModeratorRequest` | `communityhttp.communityMemberMutationResponse` | 200 |
| DELETE | /api/v1/communities/:slug/manage/moderators/:user_id | none | `communityhttp.communityMemberMutationResponse` | 200 |
| GET | /api/v1/communities/:slug/manage/owner-transfer | none | `communityhttp.communityOwnerTransferQueryResponse` | 200 |
| POST | /api/v1/communities/:slug/manage/owner-transfer | `communityhttp.createCommunityOwnerTransferRequest` | `communityhttp.communityOwnerTransferMutationResponse` | 200 |
| GET | /api/v1/communities/:slug/owner-transfer/:transfer_id | none | `communityhttp.communityOwnerTransferQueryResponse` | 200 |
| POST | /api/v1/communities/:slug/manage/owner-transfer/:transfer_id/accept | none | `communityhttp.communityOwnerTransferMutationResponse` | 200 |
| DELETE | /api/v1/communities/:slug/manage/owner-transfer/:transfer_id | none | `communityhttp.communityOwnerTransferMutationResponse` | 200 |
| POST | /api/v1/admin/communities/:id/owner | `adminhttp.updateAdminCommunityOwnerRequest` | `adminhttp.updateAdminCommunityOwnerResponse` | 200 |
| PATCH | /api/v1/admin/users/:id/platform-role | `adminhttp.updateAdminUserPlatformRoleRequest` | `adminhttp.updateAdminUserPlatformRoleResponse` | 200 |

### Required Fields

| Schema | Required JSON fields |
|---|---|
| `communityhttp.writeCommunityModeratorRequest` | `username` |
| `communityhttp.createCommunityOwnerTransferRequest` | `username` |
| `adminhttp.updateAdminCommunityOwnerRequest` | `user_id` |

### Handler JSON Schemas

| Package | Go type | JSON fields |
|---|---|---|
| `communityhttp` | `writeCommunityModeratorRequest` | `username` |
| `communityhttp` | `createCommunityOwnerTransferRequest` | `username` |
| `communityhttp` | `communityMemberMutationResponse` | `community`, `member` |
| `communityhttp` | `communityOwnerTransferResponse` | `id`, `community_id`, `from_user_id`, `from_username`, `from_display_name`, `to_user_id`, `to_username`, `to_display_name`, `status`, `created_at`, `updated_at`, `expires_at`, `accepted_at`, `cancelled_at`, `viewer_is_target`, `viewer_can_cancel`, `platform_owner_override` |
| `communityhttp` | `communityOwnerTransferMutationResponse` | `community`, `transfer` |
| `communityhttp` | `communityOwnerTransferQueryResponse` | `community`, `transfer` |
| `adminhttp` | `updateAdminCommunityOwnerRequest` | `user_id`, `reason` |
| `adminhttp` | `adminCommunityOwnerResponse` | `user_id`, `username`, `role`, `status`, `updated_at` |
| `adminhttp` | `updateAdminCommunityOwnerResponse` | `community`, `owner` |
| `adminhttp` | `updateAdminUserPlatformRoleRequest` | `role` |
| `adminhttp` | `updateAdminUserPlatformRoleResponse` | `user` |
