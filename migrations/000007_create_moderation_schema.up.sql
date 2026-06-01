CREATE TABLE content_reports (
    id uuid PRIMARY KEY,
    reporter_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    post_id uuid REFERENCES posts(id) ON DELETE RESTRICT,
    comment_id uuid REFERENCES comments(id) ON DELETE RESTRICT,
    reason text NOT NULL,
    status text NOT NULL DEFAULT 'pending',
    reviewed_by uuid REFERENCES users(id) ON DELETE RESTRICT,
    reviewed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT content_reports_target_check CHECK (
        (post_id IS NOT NULL AND comment_id IS NULL)
        OR (post_id IS NULL AND comment_id IS NOT NULL)
    ),

    CONSTRAINT content_reports_reason_not_blank_check CHECK (
        btrim(reason) <> ''
    ),

    CONSTRAINT content_reports_status_check CHECK (
        status IN ('pending', 'resolved', 'dismissed')
    ),

    CONSTRAINT content_reports_review_fields_check CHECK (
        (
            status = 'pending'
            AND reviewed_by IS NULL
            AND reviewed_at IS NULL
        )
        OR (
            status <> 'pending'
            AND reviewed_by IS NOT NULL
            AND reviewed_at IS NOT NULL
        )
    ),

    CONSTRAINT content_reports_updated_at_check CHECK (
        updated_at >= created_at
    )
);

CREATE UNIQUE INDEX content_reports_pending_post_reporter_idx
    ON content_reports (reporter_id, post_id)
    WHERE status = 'pending' AND post_id IS NOT NULL;

CREATE UNIQUE INDEX content_reports_pending_comment_reporter_idx
    ON content_reports (reporter_id, comment_id)
    WHERE status = 'pending' AND comment_id IS NOT NULL;

CREATE INDEX content_reports_post_id_idx
    ON content_reports (post_id)
    WHERE post_id IS NOT NULL;

CREATE INDEX content_reports_comment_id_idx
    ON content_reports (comment_id)
    WHERE comment_id IS NOT NULL;

CREATE INDEX content_reports_status_created_at_idx
    ON content_reports (status, created_at DESC, id DESC);

CREATE TABLE moderation_actions (
    id uuid PRIMARY KEY,
    actor_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    post_id uuid REFERENCES posts(id) ON DELETE RESTRICT,
    comment_id uuid REFERENCES comments(id) ON DELETE RESTRICT,
    action text NOT NULL,
    reason text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT moderation_actions_target_check CHECK (
        (post_id IS NOT NULL AND comment_id IS NULL)
        OR (post_id IS NULL AND comment_id IS NOT NULL)
    ),

    CONSTRAINT moderation_actions_action_check CHECK (
        action IN ('remove')
    ),

    CONSTRAINT moderation_actions_reason_not_blank_check CHECK (
        btrim(reason) <> ''
    )
);

CREATE INDEX moderation_actions_post_id_idx
    ON moderation_actions (post_id, created_at DESC)
    WHERE post_id IS NOT NULL;

CREATE INDEX moderation_actions_comment_id_idx
    ON moderation_actions (comment_id, created_at DESC)
    WHERE comment_id IS NOT NULL;

CREATE INDEX moderation_actions_actor_id_idx
    ON moderation_actions (actor_id, created_at DESC);
