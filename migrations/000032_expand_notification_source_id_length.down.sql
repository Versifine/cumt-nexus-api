ALTER TABLE notifications
    DROP CONSTRAINT IF EXISTS notifications_source_id_length_check;

ALTER TABLE notifications
    ADD CONSTRAINT notifications_source_id_length_check CHECK (char_length(source_id) <= 64) NOT VALID;
