CREATE UNIQUE INDEX community_memberships_one_active_owner_uq
    ON community_memberships (community_id)
    WHERE role = 'owner' AND status = 'active';

CREATE TABLE community_owner_transfers (
    id uuid PRIMARY KEY,
    community_id uuid NOT NULL REFERENCES communities(id) ON DELETE CASCADE,
    from_user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    to_user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status text NOT NULL DEFAULT 'pending',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    accepted_at timestamptz,

    CONSTRAINT community_owner_transfers_status_check CHECK (
        status IN ('pending', 'accepted', 'canceled', 'expired')
    ),
    CONSTRAINT community_owner_transfers_distinct_users_check CHECK (
        from_user_id <> to_user_id
    ),
    CONSTRAINT community_owner_transfers_acceptance_check CHECK (
        (status = 'accepted' AND accepted_at IS NOT NULL)
        OR (status <> 'accepted' AND accepted_at IS NULL)
    )
);

CREATE UNIQUE INDEX community_owner_transfers_one_pending_uq
    ON community_owner_transfers (community_id)
    WHERE status = 'pending';

CREATE INDEX community_owner_transfers_target_pending_idx
    ON community_owner_transfers (to_user_id, created_at DESC)
    WHERE status = 'pending';
