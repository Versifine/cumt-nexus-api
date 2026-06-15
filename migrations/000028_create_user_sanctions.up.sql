CREATE TABLE user_sanctions (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type text NOT NULL,
    status text NOT NULL,
    reason text NOT NULL,
    created_by uuid NOT NULL REFERENCES users(id),
    starts_at timestamptz NOT NULL,
    expires_at timestamptz,
    revoked_by uuid REFERENCES users(id),
    revoked_at timestamptz,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CONSTRAINT user_sanctions_type_ck CHECK (type IN ('account_ban')),
    CONSTRAINT user_sanctions_status_ck CHECK (status IN ('active', 'revoked')),
    CONSTRAINT user_sanctions_reason_ck CHECK (char_length(btrim(reason)) BETWEEN 1 AND 500),
    CONSTRAINT user_sanctions_expires_at_ck CHECK (expires_at IS NULL OR expires_at > starts_at),
    CONSTRAINT user_sanctions_revoked_fields_ck CHECK (
        (status = 'active' AND revoked_by IS NULL AND revoked_at IS NULL)
        OR (status = 'revoked' AND revoked_by IS NOT NULL AND revoked_at IS NOT NULL)
    )
);

CREATE INDEX user_sanctions_user_created_idx
    ON user_sanctions (user_id, created_at DESC, id DESC);

CREATE INDEX user_sanctions_active_account_ban_idx
    ON user_sanctions (user_id, expires_at)
    WHERE type = 'account_ban' AND status = 'active';
