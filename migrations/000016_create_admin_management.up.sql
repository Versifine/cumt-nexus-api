CREATE TABLE admin_settings (
    key text PRIMARY KEY,
    bool_value boolean NOT NULL,
    updated_by uuid REFERENCES users(id) ON DELETE SET NULL,
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT admin_settings_key_check CHECK (
        key IN ('registration_enabled', 'posting_enabled', 'upload_enabled')
    )
);

INSERT INTO admin_settings (key, bool_value)
VALUES
    ('registration_enabled', true),
    ('posting_enabled', true),
    ('upload_enabled', true);

CREATE TABLE admin_audit_logs (
    id uuid PRIMARY KEY,
    actor_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    action text NOT NULL,
    target_type text NOT NULL,
    target_id text NOT NULL,
    before_state jsonb NOT NULL DEFAULT '{}'::jsonb,
    after_state jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT admin_audit_logs_action_not_blank_check CHECK (
        btrim(action) <> ''
    ),

    CONSTRAINT admin_audit_logs_target_type_not_blank_check CHECK (
        btrim(target_type) <> ''
    ),

    CONSTRAINT admin_audit_logs_target_id_not_blank_check CHECK (
        btrim(target_id) <> ''
    )
);

CREATE INDEX admin_audit_logs_created_idx
    ON admin_audit_logs (created_at DESC, id DESC);

CREATE INDEX admin_audit_logs_target_idx
    ON admin_audit_logs (target_type, target_id, created_at DESC, id DESC);
