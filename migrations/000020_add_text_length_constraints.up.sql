ALTER TABLE users
    ADD CONSTRAINT users_password_hash_length_check CHECK (char_length(password_hash) <= 255) NOT VALID,
    ADD CONSTRAINT users_avatar_url_length_check CHECK (char_length(avatar_url) <= 2048) NOT VALID,
    ADD CONSTRAINT users_banner_url_length_check CHECK (char_length(banner_url) <= 2048) NOT VALID;

ALTER TABLE communities
    ADD CONSTRAINT communities_name_length_check CHECK (char_length(name) <= 60) NOT VALID,
    ADD CONSTRAINT communities_description_length_check CHECK (char_length(description) <= 300) NOT VALID;

ALTER TABLE community_applications
    ADD CONSTRAINT community_applications_requested_name_length_check CHECK (char_length(requested_name) <= 60) NOT VALID,
    ADD CONSTRAINT community_applications_reason_length_check CHECK (char_length(reason) <= 500) NOT VALID,
    ADD CONSTRAINT community_applications_reject_reason_length_check CHECK (reject_reason IS NULL OR char_length(reject_reason) <= 500) NOT VALID;

ALTER TABLE posts
    ADD CONSTRAINT posts_title_length_check CHECK (char_length(title) <= 120) NOT VALID,
    ADD CONSTRAINT posts_body_length_check CHECK (char_length(body) <= 20000) NOT VALID;

ALTER TABLE comments
    ADD CONSTRAINT comments_body_length_check CHECK (char_length(body) <= 5000) NOT VALID;

ALTER TABLE content_reports
    ADD CONSTRAINT content_reports_reason_length_check CHECK (char_length(reason) <= 500) NOT VALID;

ALTER TABLE moderation_actions
    ADD CONSTRAINT moderation_actions_reason_length_check CHECK (char_length(reason) <= 500) NOT VALID;

ALTER TABLE notifications
    ADD CONSTRAINT notifications_type_length_check CHECK (char_length(type) <= 64) NOT VALID,
    ADD CONSTRAINT notifications_title_length_check CHECK (char_length(title) <= 120) NOT VALID,
    ADD CONSTRAINT notifications_body_length_check CHECK (char_length(body) <= 500) NOT VALID,
    ADD CONSTRAINT notifications_source_type_length_check CHECK (char_length(source_type) <= 64) NOT VALID,
    ADD CONSTRAINT notifications_source_id_length_check CHECK (char_length(source_id) <= 64) NOT VALID,
    ADD CONSTRAINT notifications_aggregate_key_length_check CHECK (char_length(aggregate_key) <= 160) NOT VALID;

ALTER TABLE media_attachments
    ADD CONSTRAINT media_attachments_bucket_length_check CHECK (char_length(bucket) <= 128) NOT VALID,
    ADD CONSTRAINT media_attachments_object_key_length_check CHECK (char_length(object_key) <= 1024) NOT VALID,
    ADD CONSTRAINT media_attachments_public_url_length_check CHECK (char_length(public_url) <= 2048) NOT VALID,
    ADD CONSTRAINT media_attachments_thumbnail_object_key_length_check CHECK (thumbnail_object_key IS NULL OR char_length(thumbnail_object_key) <= 1024) NOT VALID,
    ADD CONSTRAINT media_attachments_mime_type_length_check CHECK (char_length(mime_type) <= 64) NOT VALID;

ALTER TABLE effects
    ADD CONSTRAINT effects_name_length_check CHECK (char_length(name) <= 80) NOT VALID,
    ADD CONSTRAINT effects_description_length_check CHECK (char_length(description) <= 300) NOT VALID,
    ADD CONSTRAINT effects_asset_url_length_check CHECK (char_length(asset_url) <= 2048) NOT VALID,
    ADD CONSTRAINT effects_animation_key_length_check CHECK (char_length(animation_key) <= 80) NOT VALID;

ALTER TABLE point_transactions
    ADD CONSTRAINT point_transactions_reason_length_check CHECK (char_length(reason) <= 120) NOT VALID,
    ADD CONSTRAINT point_transactions_source_type_length_check CHECK (char_length(source_type) <= 64) NOT VALID,
    ADD CONSTRAINT point_transactions_source_id_length_check CHECK (char_length(source_id) <= 128) NOT VALID;

ALTER TABLE community_rules
    ADD CONSTRAINT community_rules_title_length_check CHECK (char_length(title) <= 80) NOT VALID,
    ADD CONSTRAINT community_rules_body_length_check CHECK (char_length(body) <= 500) NOT VALID;

ALTER TABLE post_content_refs
    ADD CONSTRAINT post_content_refs_ref_id_length_check CHECK (char_length(ref_id) <= 2048) NOT VALID;

ALTER TABLE comment_content_refs
    ADD CONSTRAINT comment_content_refs_ref_id_length_check CHECK (char_length(ref_id) <= 2048) NOT VALID;

ALTER TABLE admin_settings
    ADD CONSTRAINT admin_settings_key_length_check CHECK (char_length(key) <= 64) NOT VALID;

ALTER TABLE admin_audit_logs
    ADD CONSTRAINT admin_audit_logs_action_length_check CHECK (char_length(action) <= 128) NOT VALID,
    ADD CONSTRAINT admin_audit_logs_target_type_length_check CHECK (char_length(target_type) <= 64) NOT VALID,
    ADD CONSTRAINT admin_audit_logs_target_id_length_check CHECK (char_length(target_id) <= 128) NOT VALID;

ALTER TABLE embeds
    ADD CONSTRAINT embeds_provider_length_check CHECK (char_length(provider) <= 64) NOT VALID,
    ADD CONSTRAINT embeds_provider_ref_length_check CHECK (char_length(provider_ref) <= 256) NOT VALID,
    ADD CONSTRAINT embeds_url_length_check CHECK (char_length(url) <= 2048) NOT VALID,
    ADD CONSTRAINT embeds_canonical_url_length_check CHECK (char_length(canonical_url) <= 2048) NOT VALID,
    ADD CONSTRAINT embeds_embed_url_length_check CHECK (char_length(embed_url) <= 2048) NOT VALID,
    ADD CONSTRAINT embeds_title_length_check CHECK (char_length(title) <= 200) NOT VALID,
    ADD CONSTRAINT embeds_description_length_check CHECK (char_length(description) <= 500) NOT VALID,
    ADD CONSTRAINT embeds_image_url_length_check CHECK (char_length(image_url) <= 2048) NOT VALID,
    ADD CONSTRAINT embeds_author_name_length_check CHECK (char_length(author_name) <= 80) NOT VALID,
    ADD CONSTRAINT embeds_status_length_check CHECK (char_length(status) <= 64) NOT VALID;
