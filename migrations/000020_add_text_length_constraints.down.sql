ALTER TABLE embeds
    DROP CONSTRAINT IF EXISTS embeds_status_length_check,
    DROP CONSTRAINT IF EXISTS embeds_author_name_length_check,
    DROP CONSTRAINT IF EXISTS embeds_image_url_length_check,
    DROP CONSTRAINT IF EXISTS embeds_description_length_check,
    DROP CONSTRAINT IF EXISTS embeds_title_length_check,
    DROP CONSTRAINT IF EXISTS embeds_embed_url_length_check,
    DROP CONSTRAINT IF EXISTS embeds_canonical_url_length_check,
    DROP CONSTRAINT IF EXISTS embeds_url_length_check,
    DROP CONSTRAINT IF EXISTS embeds_provider_ref_length_check,
    DROP CONSTRAINT IF EXISTS embeds_provider_length_check;

ALTER TABLE admin_audit_logs
    DROP CONSTRAINT IF EXISTS admin_audit_logs_target_id_length_check,
    DROP CONSTRAINT IF EXISTS admin_audit_logs_target_type_length_check,
    DROP CONSTRAINT IF EXISTS admin_audit_logs_action_length_check;

ALTER TABLE admin_settings
    DROP CONSTRAINT IF EXISTS admin_settings_key_length_check;

ALTER TABLE comment_content_refs
    DROP CONSTRAINT IF EXISTS comment_content_refs_ref_id_length_check;

ALTER TABLE post_content_refs
    DROP CONSTRAINT IF EXISTS post_content_refs_ref_id_length_check;

ALTER TABLE community_rules
    DROP CONSTRAINT IF EXISTS community_rules_body_length_check,
    DROP CONSTRAINT IF EXISTS community_rules_title_length_check;

ALTER TABLE point_transactions
    DROP CONSTRAINT IF EXISTS point_transactions_source_id_length_check,
    DROP CONSTRAINT IF EXISTS point_transactions_source_type_length_check,
    DROP CONSTRAINT IF EXISTS point_transactions_reason_length_check;

ALTER TABLE effects
    DROP CONSTRAINT IF EXISTS effects_animation_key_length_check,
    DROP CONSTRAINT IF EXISTS effects_asset_url_length_check,
    DROP CONSTRAINT IF EXISTS effects_description_length_check,
    DROP CONSTRAINT IF EXISTS effects_name_length_check;

ALTER TABLE media_attachments
    DROP CONSTRAINT IF EXISTS media_attachments_mime_type_length_check,
    DROP CONSTRAINT IF EXISTS media_attachments_thumbnail_object_key_length_check,
    DROP CONSTRAINT IF EXISTS media_attachments_public_url_length_check,
    DROP CONSTRAINT IF EXISTS media_attachments_object_key_length_check,
    DROP CONSTRAINT IF EXISTS media_attachments_bucket_length_check;

ALTER TABLE notifications
    DROP CONSTRAINT IF EXISTS notifications_aggregate_key_length_check,
    DROP CONSTRAINT IF EXISTS notifications_source_id_length_check,
    DROP CONSTRAINT IF EXISTS notifications_source_type_length_check,
    DROP CONSTRAINT IF EXISTS notifications_body_length_check,
    DROP CONSTRAINT IF EXISTS notifications_title_length_check,
    DROP CONSTRAINT IF EXISTS notifications_type_length_check;

ALTER TABLE moderation_actions
    DROP CONSTRAINT IF EXISTS moderation_actions_reason_length_check;

ALTER TABLE content_reports
    DROP CONSTRAINT IF EXISTS content_reports_reason_length_check;

ALTER TABLE comments
    DROP CONSTRAINT IF EXISTS comments_body_length_check;

ALTER TABLE posts
    DROP CONSTRAINT IF EXISTS posts_body_length_check,
    DROP CONSTRAINT IF EXISTS posts_title_length_check;

ALTER TABLE community_applications
    DROP CONSTRAINT IF EXISTS community_applications_reject_reason_length_check,
    DROP CONSTRAINT IF EXISTS community_applications_reason_length_check,
    DROP CONSTRAINT IF EXISTS community_applications_requested_name_length_check;

ALTER TABLE communities
    DROP CONSTRAINT IF EXISTS communities_description_length_check,
    DROP CONSTRAINT IF EXISTS communities_name_length_check;

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_banner_url_length_check,
    DROP CONSTRAINT IF EXISTS users_avatar_url_length_check,
    DROP CONSTRAINT IF EXISTS users_password_hash_length_check;
