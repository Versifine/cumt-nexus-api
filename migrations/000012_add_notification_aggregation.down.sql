DROP INDEX IF EXISTS notifications_unread_aggregate_key_uq;

ALTER TABLE notifications
    DROP CONSTRAINT IF EXISTS notifications_aggregate_count_positive_check;

ALTER TABLE notifications
    DROP COLUMN IF EXISTS last_actor_id,
    DROP COLUMN IF EXISTS aggregate_count,
    DROP COLUMN IF EXISTS aggregate_key;

