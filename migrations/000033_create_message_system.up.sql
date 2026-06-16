CREATE TABLE IF NOT EXISTS user_blocks (
    blocker_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    blocked_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT user_blocks_pk PRIMARY KEY (blocker_id, blocked_id),
    CONSTRAINT user_blocks_no_self_check CHECK (blocker_id <> blocked_id)
);

CREATE INDEX IF NOT EXISTS user_blocks_blocked_idx
    ON user_blocks (blocked_id, created_at DESC);

CREATE TABLE IF NOT EXISTS message_privacy_settings (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    allow_messages text NOT NULL DEFAULT 'everyone',
    online_status_enabled boolean NOT NULL DEFAULT false,
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT message_privacy_allow_messages_check CHECK (
        allow_messages IN ('everyone', 'mutuals', 'none')
    )
);

CREATE TABLE IF NOT EXISTS message_conversations (
    id uuid PRIMARY KEY,
    kind text NOT NULL DEFAULT 'direct',
    status text NOT NULL,
    created_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT message_conversations_kind_check CHECK (kind IN ('direct')),
    CONSTRAINT message_conversations_status_check CHECK (
        status IN ('accepted', 'pending', 'rejected', 'deleted')
    )
);

CREATE TABLE IF NOT EXISTS message_conversation_participants (
    conversation_id uuid NOT NULL REFERENCES message_conversations(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    peer_user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    read_at timestamptz,
    archived_at timestamptz,
    deleted_at timestamptz,
    pinned boolean NOT NULL DEFAULT false,
    muted boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT message_conversation_participants_pk PRIMARY KEY (conversation_id, user_id),
    CONSTRAINT message_conversation_participants_no_self_check CHECK (user_id <> peer_user_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS message_direct_conversation_pair_uq
    ON message_conversation_participants (
        LEAST(user_id, peer_user_id),
        GREATEST(user_id, peer_user_id)
    )
    WHERE user_id < peer_user_id;

CREATE INDEX IF NOT EXISTS message_conversation_participants_user_updated_idx
    ON message_conversation_participants (user_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS messages (
    id uuid PRIMARY KEY,
    conversation_id uuid NOT NULL REFERENCES message_conversations(id) ON DELETE CASCADE,
    sender_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    type text NOT NULL,
    body text NOT NULL DEFAULT '',
    image_url text NOT NULL DEFAULT '',
    share_type text,
    share_id text,
    share_title text NOT NULL DEFAULT '',
    share_summary text NOT NULL DEFAULT '',
    share_thumbnail_url text NOT NULL DEFAULT '',
    share_target_url text NOT NULL DEFAULT '',
    share_snapshot_created_at timestamptz,
    status text NOT NULL DEFAULT 'visible',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    recalled_at timestamptz,
    CONSTRAINT messages_type_check CHECK (
        type IN ('text', 'image', 'share_post', 'share_comment', 'share_user', 'share_community')
    ),
    CONSTRAINT messages_share_type_check CHECK (
        share_type IS NULL OR share_type IN ('post', 'comment', 'user', 'community')
    ),
    CONSTRAINT messages_status_check CHECK (
        status IN ('visible', 'recalled', 'deleted', 'image_failed', 'hidden')
    ),
    CONSTRAINT messages_body_length_check CHECK (char_length(body) <= 4000),
    CONSTRAINT messages_image_url_length_check CHECK (char_length(image_url) <= 2048),
    CONSTRAINT messages_share_target_url_length_check CHECK (char_length(share_target_url) <= 2048)
);

CREATE INDEX IF NOT EXISTS messages_conversation_created_idx
    ON messages (conversation_id, created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS message_user_states (
    message_id uuid NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    hidden_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT message_user_states_pk PRIMARY KEY (message_id, user_id)
);

CREATE TABLE IF NOT EXISTS message_requests (
    id uuid PRIMARY KEY,
    conversation_id uuid NOT NULL UNIQUE REFERENCES message_conversations(id) ON DELETE CASCADE,
    from_user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    to_user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    responded_at timestamptz,
    CONSTRAINT message_requests_status_check CHECK (
        status IN ('pending', 'accepted', 'rejected', 'deleted')
    ),
    CONSTRAINT message_requests_no_self_check CHECK (from_user_id <> to_user_id)
);

CREATE INDEX IF NOT EXISTS message_requests_to_status_created_idx
    ON message_requests (to_user_id, status, created_at DESC);

CREATE TABLE IF NOT EXISTS message_reports (
    id uuid PRIMARY KEY,
    reporter_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    conversation_id uuid NOT NULL REFERENCES message_conversations(id) ON DELETE CASCADE,
    message_id uuid NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    reported_user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    reason text NOT NULL,
    context_before text NOT NULL DEFAULT '',
    context_after text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT message_reports_reason_length_check CHECK (
        char_length(reason) BETWEEN 1 AND 500
    )
);

CREATE INDEX IF NOT EXISTS message_reports_created_idx
    ON message_reports (created_at DESC);

CREATE TABLE IF NOT EXISTS message_realtime_events (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    conversation_id uuid REFERENCES message_conversations(id) ON DELETE CASCADE,
    type text NOT NULL,
    payload text NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT message_realtime_events_type_check CHECK (
        type IN ('message_created', 'message_recalled', 'conversation_updated', 'unread_updated', 'request_accepted', 'request_rejected', 'block_changed')
    )
);

CREATE INDEX IF NOT EXISTS message_realtime_events_user_created_idx
    ON message_realtime_events (user_id, created_at ASC, id ASC);

CREATE TABLE IF NOT EXISTS message_realtime_tickets (
    ticket text PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    last_event_id uuid,
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS message_realtime_tickets_user_expires_idx
    ON message_realtime_tickets (user_id, expires_at DESC);
