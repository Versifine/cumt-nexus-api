ALTER TABLE posts
    DROP CONSTRAINT posts_status_check,
    ADD CONSTRAINT posts_status_check CHECK (
        status IN ('visible', 'removed', 'deleted', 'locked', 'hidden', 'spam')
    ),
    ADD COLUMN is_locked boolean NOT NULL DEFAULT false,
    ADD COLUMN is_pinned boolean NOT NULL DEFAULT false,
    ADD COLUMN is_nsfw boolean NOT NULL DEFAULT false,
    ADD COLUMN is_spoiler boolean NOT NULL DEFAULT false,
    ADD COLUMN flair_text text NOT NULL DEFAULT '',
    ADD CONSTRAINT posts_flair_text_length_check CHECK (char_length(flair_text) <= 64);

ALTER TABLE comments
    DROP CONSTRAINT comments_status_check,
    ADD CONSTRAINT comments_status_check CHECK (
        status IN ('visible', 'removed', 'deleted', 'locked', 'hidden', 'spam')
    ),
    ADD COLUMN is_locked boolean NOT NULL DEFAULT false;

ALTER TABLE moderation_actions
    DROP CONSTRAINT moderation_actions_action_check,
    ADD CONSTRAINT moderation_actions_action_check CHECK (
        action IN (
            'remove',
            'approve',
            'spam',
            'ignore_reports',
            'lock',
            'pin',
            'mark_nsfw',
            'mark_spoiler',
            'set_flair'
        )
    );

CREATE TABLE community_moderation_logs (
    id uuid PRIMARY KEY,
    community_id uuid NOT NULL REFERENCES communities(id) ON DELETE RESTRICT,
    actor_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    action text NOT NULL,
    target_type text NOT NULL,
    target_id text NOT NULL,
    batch_id uuid,
    before_state jsonb NOT NULL DEFAULT '{}'::jsonb,
    after_state jsonb NOT NULL DEFAULT '{}'::jsonb,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT community_moderation_logs_action_not_blank_check CHECK (btrim(action) <> ''),
    CONSTRAINT community_moderation_logs_target_type_not_blank_check CHECK (btrim(target_type) <> ''),
    CONSTRAINT community_moderation_logs_target_id_not_blank_check CHECK (btrim(target_id) <> '')
);

CREATE INDEX community_moderation_logs_community_created_idx
    ON community_moderation_logs (community_id, created_at DESC, id DESC);

CREATE INDEX community_moderation_logs_target_idx
    ON community_moderation_logs (community_id, target_type, target_id, created_at DESC, id DESC);

CREATE INDEX community_moderation_logs_actor_idx
    ON community_moderation_logs (community_id, actor_id, created_at DESC, id DESC);

CREATE TABLE moderation_removal_reasons (
    id uuid PRIMARY KEY,
    community_id uuid NOT NULL REFERENCES communities(id) ON DELETE CASCADE,
    title text NOT NULL,
    body text NOT NULL DEFAULT '',
    rule_id uuid REFERENCES community_rules(id) ON DELETE SET NULL,
    is_active boolean NOT NULL DEFAULT true,
    position integer NOT NULL DEFAULT 0,
    created_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    updated_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT moderation_removal_reasons_title_not_blank_check CHECK (btrim(title) <> ''),
    CONSTRAINT moderation_removal_reasons_title_length_check CHECK (char_length(title) <= 80),
    CONSTRAINT moderation_removal_reasons_body_length_check CHECK (char_length(body) <= 1000),
    CONSTRAINT moderation_removal_reasons_position_check CHECK (position >= 0),
    CONSTRAINT moderation_removal_reasons_updated_at_check CHECK (updated_at >= created_at)
);

CREATE INDEX moderation_removal_reasons_community_position_idx
    ON moderation_removal_reasons (community_id, is_active DESC, position ASC, created_at ASC, id ASC);

CREATE TABLE moderation_saved_responses (
    id uuid PRIMARY KEY,
    community_id uuid NOT NULL REFERENCES communities(id) ON DELETE CASCADE,
    title text NOT NULL,
    body text NOT NULL,
    is_active boolean NOT NULL DEFAULT true,
    position integer NOT NULL DEFAULT 0,
    created_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    updated_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT moderation_saved_responses_title_not_blank_check CHECK (btrim(title) <> ''),
    CONSTRAINT moderation_saved_responses_body_not_blank_check CHECK (btrim(body) <> ''),
    CONSTRAINT moderation_saved_responses_title_length_check CHECK (char_length(title) <= 80),
    CONSTRAINT moderation_saved_responses_body_length_check CHECK (char_length(body) <= 2000),
    CONSTRAINT moderation_saved_responses_position_check CHECK (position >= 0),
    CONSTRAINT moderation_saved_responses_updated_at_check CHECK (updated_at >= created_at)
);

CREATE INDEX moderation_saved_responses_community_position_idx
    ON moderation_saved_responses (community_id, is_active DESC, position ASC, created_at ASC, id ASC);

CREATE TABLE community_user_moderation_states (
    id uuid PRIMARY KEY,
    community_id uuid NOT NULL REFERENCES communities(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    kind text NOT NULL,
    reason text NOT NULL DEFAULT '',
    expires_at timestamptz,
    created_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    updated_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT community_user_moderation_states_kind_check CHECK (
        kind IN ('banned', 'muted', 'approved')
    ),
    CONSTRAINT community_user_moderation_states_reason_length_check CHECK (char_length(reason) <= 500),
    CONSTRAINT community_user_moderation_states_updated_at_check CHECK (updated_at >= created_at),
    CONSTRAINT community_user_moderation_states_unique_kind UNIQUE (community_id, user_id, kind)
);

CREATE INDEX community_user_moderation_states_kind_idx
    ON community_user_moderation_states (community_id, kind, updated_at DESC, id DESC);

CREATE TABLE community_moderator_notes (
    id uuid PRIMARY KEY,
    community_id uuid NOT NULL REFERENCES communities(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    author_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    body text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    deleted_by uuid REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT community_moderator_notes_body_not_blank_check CHECK (btrim(body) <> ''),
    CONSTRAINT community_moderator_notes_body_length_check CHECK (char_length(body) <= 1000)
);

CREATE INDEX community_moderator_notes_user_idx
    ON community_moderator_notes (community_id, user_id, created_at DESC, id DESC)
    WHERE deleted_at IS NULL;
