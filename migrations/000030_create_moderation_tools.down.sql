DROP INDEX IF EXISTS community_moderator_notes_user_idx;
DROP TABLE IF EXISTS community_moderator_notes;

DROP INDEX IF EXISTS community_user_moderation_states_kind_idx;
DROP TABLE IF EXISTS community_user_moderation_states;

DROP INDEX IF EXISTS moderation_saved_responses_community_position_idx;
DROP TABLE IF EXISTS moderation_saved_responses;

DROP INDEX IF EXISTS moderation_removal_reasons_community_position_idx;
DROP TABLE IF EXISTS moderation_removal_reasons;

DROP INDEX IF EXISTS community_moderation_logs_actor_idx;
DROP INDEX IF EXISTS community_moderation_logs_target_idx;
DROP INDEX IF EXISTS community_moderation_logs_community_created_idx;
DROP TABLE IF EXISTS community_moderation_logs;

ALTER TABLE moderation_actions
    DROP CONSTRAINT moderation_actions_action_check,
    ADD CONSTRAINT moderation_actions_action_check CHECK (
        action IN ('remove')
    );

ALTER TABLE comments
    DROP COLUMN is_locked,
    DROP CONSTRAINT comments_status_check,
    ADD CONSTRAINT comments_status_check CHECK (
        status IN ('visible', 'removed', 'deleted', 'locked', 'hidden')
    );

ALTER TABLE posts
    DROP CONSTRAINT posts_flair_text_length_check,
    DROP COLUMN flair_text,
    DROP COLUMN is_spoiler,
    DROP COLUMN is_nsfw,
    DROP COLUMN is_pinned,
    DROP COLUMN is_locked,
    DROP CONSTRAINT posts_status_check,
    ADD CONSTRAINT posts_status_check CHECK (
        status IN ('visible', 'removed', 'deleted', 'locked', 'hidden')
    );
