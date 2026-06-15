ALTER TABLE auth_email_codes
    DROP CONSTRAINT auth_email_codes_purpose_check,
    ADD CONSTRAINT auth_email_codes_purpose_check CHECK (purpose IN ('register', 'login', 'password_reset', 'change_email'));

ALTER TABLE users
    DROP CONSTRAINT users_status_check,
    ADD CONSTRAINT users_status_check CHECK (
        status IN ('active', 'disabled')
    );

ALTER TABLE users
    DROP COLUMN deleted_at;
