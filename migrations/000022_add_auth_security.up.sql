ALTER TABLE users
    ADD COLUMN email text NOT NULL DEFAULT '',
    ADD COLUMN email_verified_at timestamptz,
    ADD COLUMN last_login_at timestamptz,
    ADD COLUMN last_login_ip text NOT NULL DEFAULT '',
    ADD COLUMN password_updated_at timestamptz,
    ADD COLUMN tokens_revoked_after timestamptz,
    ADD CONSTRAINT users_email_length_check CHECK (char_length(email) <= 254),
    ADD CONSTRAINT users_email_lowercase_check CHECK (email = lower(email));

CREATE UNIQUE INDEX users_email_lower_uq
    ON users (lower(email))
    WHERE email <> '';

CREATE TABLE auth_email_codes (
    id uuid PRIMARY KEY,
    email text NOT NULL,
    purpose text NOT NULL,
    code_hash text NOT NULL,
    status text NOT NULL DEFAULT 'pending',
    attempt_count integer NOT NULL DEFAULT 0,
    sent_count integer NOT NULL DEFAULT 1,
    last_sent_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    request_ip text NOT NULL DEFAULT '',
    user_agent text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT auth_email_codes_email_length_check CHECK (char_length(email) <= 254),
    CONSTRAINT auth_email_codes_email_lowercase_check CHECK (email = lower(email)),
    CONSTRAINT auth_email_codes_purpose_check CHECK (purpose IN ('register', 'login', 'password_reset', 'change_email')),
    CONSTRAINT auth_email_codes_status_check CHECK (status IN ('pending', 'used', 'expired', 'blocked')),
    CONSTRAINT auth_email_codes_attempt_count_check CHECK (attempt_count >= 0),
    CONSTRAINT auth_email_codes_sent_count_check CHECK (sent_count > 0),
    CONSTRAINT auth_email_codes_time_check CHECK (expires_at > created_at)
);

CREATE INDEX auth_email_codes_email_purpose_created_idx
    ON auth_email_codes (email, purpose, created_at DESC);

CREATE INDEX auth_email_codes_pending_idx
    ON auth_email_codes (email, purpose, expires_at)
    WHERE status = 'pending';

CREATE TABLE auth_security_events (
    id uuid PRIMARY KEY,
    user_id uuid REFERENCES users(id) ON DELETE SET NULL,
    email text NOT NULL DEFAULT '',
    action text NOT NULL,
    ip text NOT NULL DEFAULT '',
    user_agent text NOT NULL DEFAULT '',
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT auth_security_events_email_length_check CHECK (char_length(email) <= 254),
    CONSTRAINT auth_security_events_email_lowercase_check CHECK (email = lower(email)),
    CONSTRAINT auth_security_events_action_length_check CHECK (char_length(action) BETWEEN 1 AND 80)
);

CREATE INDEX auth_security_events_user_created_idx
    ON auth_security_events (user_id, created_at DESC);

CREATE INDEX auth_security_events_email_created_idx
    ON auth_security_events (email, created_at DESC);
