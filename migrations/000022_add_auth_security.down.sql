DROP TABLE IF EXISTS auth_security_events;
DROP TABLE IF EXISTS auth_email_codes;

DROP INDEX IF EXISTS users_email_lower_uq;

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_email_lowercase_check,
    DROP CONSTRAINT IF EXISTS users_email_length_check,
    DROP COLUMN IF EXISTS tokens_revoked_after,
    DROP COLUMN IF EXISTS password_updated_at,
    DROP COLUMN IF EXISTS last_login_ip,
    DROP COLUMN IF EXISTS last_login_at,
    DROP COLUMN IF EXISTS email_verified_at,
    DROP COLUMN IF EXISTS email;
