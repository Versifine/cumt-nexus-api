CREATE TABLE community_modmail_conversations (
    id uuid PRIMARY KEY,
    community_id uuid NOT NULL REFERENCES communities(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    subject text NOT NULL,
    status text NOT NULL DEFAULT 'open',
    folder text NOT NULL DEFAULT 'inbox',
    assigned_to uuid REFERENCES users(id) ON DELETE SET NULL,
    created_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    last_message_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT community_modmail_conversations_subject_not_blank_check CHECK (btrim(subject) <> ''),
    CONSTRAINT community_modmail_conversations_subject_length_check CHECK (char_length(subject) <= 160),
    CONSTRAINT community_modmail_conversations_status_check CHECK (status IN ('open', 'needs_reply', 'in_progress', 'archived', 'closed')),
    CONSTRAINT community_modmail_conversations_folder_check CHECK (folder IN ('inbox', 'needs_reply', 'in_progress', 'archived')),
    CONSTRAINT community_modmail_conversations_updated_at_check CHECK (updated_at >= created_at)
);

CREATE INDEX community_modmail_conversations_folder_idx
    ON community_modmail_conversations (community_id, folder, last_message_at DESC, id DESC);

CREATE INDEX community_modmail_conversations_user_idx
    ON community_modmail_conversations (community_id, user_id, last_message_at DESC, id DESC);

CREATE TABLE community_modmail_messages (
    id uuid PRIMARY KEY,
    conversation_id uuid NOT NULL REFERENCES community_modmail_conversations(id) ON DELETE CASCADE,
    author_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    body text NOT NULL,
    is_internal boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT community_modmail_messages_body_not_blank_check CHECK (btrim(body) <> ''),
    CONSTRAINT community_modmail_messages_body_length_check CHECK (char_length(body) <= 4000)
);

CREATE INDEX community_modmail_messages_conversation_created_idx
    ON community_modmail_messages (conversation_id, created_at ASC, id ASC);

CREATE TABLE community_modmail_reads (
    conversation_id uuid NOT NULL REFERENCES community_modmail_conversations(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    read_at timestamptz NOT NULL,
    PRIMARY KEY (conversation_id, user_id)
);
