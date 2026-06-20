CREATE INDEX IF NOT EXISTS notifications_recipient_created_idx
    ON notifications (recipient_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS notifications_recipient_type_created_idx
    ON notifications (recipient_id, type, created_at DESC, id DESC);
