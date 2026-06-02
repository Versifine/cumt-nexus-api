CREATE TABLE notifications (
    id uuid PRIMARY KEY,
    recipient_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type text NOT NULL,
    title text NOT NULL,
    body text NOT NULL,
    source_type text NOT NULL DEFAULT '',
    source_id text NOT NULL DEFAULT '',
    read_at timestamptz,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CONSTRAINT notifications_type_not_blank CHECK (btrim(type) <> ''),
    CONSTRAINT notifications_title_not_blank CHECK (btrim(title) <> ''),
    CONSTRAINT notifications_body_not_blank CHECK (btrim(body) <> ''),
    CONSTRAINT notifications_updated_after_created CHECK (updated_at >= created_at)
);

CREATE INDEX notifications_recipient_unread_created_at_idx
    ON notifications (recipient_id, created_at DESC, id DESC)
    WHERE read_at IS NULL;

CREATE INDEX notifications_recipient_read_created_at_idx
    ON notifications (recipient_id, created_at DESC, id DESC)
    WHERE read_at IS NOT NULL;
