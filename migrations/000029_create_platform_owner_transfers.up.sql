ALTER TABLE admin_audit_logs
    ADD COLUMN actor_ref text;

UPDATE admin_audit_logs
SET actor_ref = actor_id::text
WHERE actor_ref IS NULL;

ALTER TABLE admin_audit_logs
    ALTER COLUMN actor_id DROP NOT NULL,
    ADD CONSTRAINT admin_audit_logs_actor_ref_ck CHECK (
        actor_id IS NOT NULL OR btrim(COALESCE(actor_ref, '')) <> ''
    );

CREATE INDEX admin_audit_logs_actor_ref_idx
    ON admin_audit_logs (actor_ref, created_at DESC, id DESC);

CREATE UNIQUE INDEX users_active_platform_owner_unique_idx
    ON users (platform_role)
    WHERE status = 'active' AND platform_role = 'owner';

CREATE TABLE platform_owner_transfers (
    id uuid PRIMARY KEY,
    status text NOT NULL,
    initiated_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    target_user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    previous_owner_role text,
    reason text NOT NULL,
    expires_at timestamptz NOT NULL,
    accepted_at timestamptz,
    cancelled_at timestamptz,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,

    CONSTRAINT platform_owner_transfers_status_ck CHECK (
        status IN ('pending', 'accepted', 'cancelled', 'expired')
    ),
    CONSTRAINT platform_owner_transfers_previous_owner_role_ck CHECK (
        previous_owner_role IS NULL OR previous_owner_role = 'admin'
    ),
    CONSTRAINT platform_owner_transfers_reason_ck CHECK (
        char_length(btrim(reason)) BETWEEN 1 AND 500
    ),
    CONSTRAINT platform_owner_transfers_distinct_users_ck CHECK (
        initiated_by <> target_user_id
    ),
    CONSTRAINT platform_owner_transfers_expires_at_ck CHECK (
        expires_at > created_at
    ),
    CONSTRAINT platform_owner_transfers_status_times_ck CHECK (
        (status IN ('pending', 'expired') AND accepted_at IS NULL AND cancelled_at IS NULL)
        OR (status = 'accepted' AND accepted_at IS NOT NULL AND cancelled_at IS NULL)
        OR (status = 'cancelled' AND accepted_at IS NULL AND cancelled_at IS NOT NULL)
    )
);

CREATE UNIQUE INDEX platform_owner_transfers_pending_unique_idx
    ON platform_owner_transfers (status)
    WHERE status = 'pending';

CREATE INDEX platform_owner_transfers_target_idx
    ON platform_owner_transfers (target_user_id, created_at DESC, id DESC);

CREATE INDEX platform_owner_transfers_initiator_idx
    ON platform_owner_transfers (initiated_by, created_at DESC, id DESC);
