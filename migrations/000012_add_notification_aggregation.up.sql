ALTER TABLE notifications
    ADD COLUMN aggregate_key text NOT NULL DEFAULT '',
    ADD COLUMN aggregate_count integer NOT NULL DEFAULT 1,
    ADD COLUMN last_actor_id uuid REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE notifications
    ADD CONSTRAINT notifications_aggregate_count_positive_check CHECK (
        aggregate_count >= 1
    );

CREATE UNIQUE INDEX notifications_unread_aggregate_key_uq
    ON notifications (recipient_id, type, aggregate_key)
    WHERE read_at IS NULL
        AND aggregate_key <> '';

