CREATE TABLE user_progressions (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE RESTRICT,
    xp_total integer NOT NULL DEFAULT 0,
    active_title_grant_id uuid,
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT user_progressions_xp_total_non_negative_check CHECK (
        xp_total >= 0
    )
);

CREATE TABLE xp_event_claims (
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    source_type text NOT NULL,
    source_id text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),

    PRIMARY KEY (user_id, source_type, source_id),

    CONSTRAINT xp_event_claims_source_type_not_blank_check CHECK (
        btrim(source_type) <> ''
    ),

    CONSTRAINT xp_event_claims_source_id_not_blank_check CHECK (
        btrim(source_id) <> ''
    )
);

CREATE TABLE xp_events (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    delta integer NOT NULL,
    xp_total_after integer NOT NULL,
    reason text NOT NULL,
    source_type text NOT NULL,
    source_id text NOT NULL,
    actor_id uuid REFERENCES users(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT xp_events_delta_positive_check CHECK (
        delta > 0
    ),

    CONSTRAINT xp_events_xp_total_after_non_negative_check CHECK (
        xp_total_after >= 0
    ),

    CONSTRAINT xp_events_reason_not_blank_check CHECK (
        btrim(reason) <> ''
    ),

    CONSTRAINT xp_events_source_type_not_blank_check CHECK (
        btrim(source_type) <> ''
    ),

    CONSTRAINT xp_events_source_id_not_blank_check CHECK (
        btrim(source_id) <> ''
    )
);

CREATE UNIQUE INDEX xp_events_user_source_unique_idx
    ON xp_events (user_id, source_type, source_id);

CREATE INDEX xp_events_user_created_idx
    ON xp_events (user_id, created_at DESC, id DESC);

CREATE INDEX xp_events_user_source_day_idx
    ON xp_events (user_id, source_type, created_at);

CREATE TABLE titles (
    id uuid PRIMARY KEY,
    name text NOT NULL,
    description text NOT NULL DEFAULT '',
    scope_type text NOT NULL,
    scope_id text NOT NULL DEFAULT '',
    is_active boolean NOT NULL DEFAULT true,
    created_by uuid REFERENCES users(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT titles_name_not_blank_check CHECK (
        btrim(name) <> ''
    ),

    CONSTRAINT titles_name_length_check CHECK (
        char_length(name) <= 20
    ),

    CONSTRAINT titles_description_length_check CHECK (
        char_length(description) <= 120
    ),

    CONSTRAINT titles_scope_type_check CHECK (
        scope_type IN ('platform', 'system', 'community')
    ),

    CONSTRAINT titles_community_scope_id_check CHECK (
        scope_type <> 'community' OR btrim(scope_id) <> ''
    )
);

CREATE INDEX titles_scope_active_idx
    ON titles (scope_type, scope_id, is_active, created_at DESC, id DESC);

CREATE TABLE title_grants (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    title_id uuid NOT NULL REFERENCES titles(id) ON DELETE RESTRICT,
    granted_by uuid REFERENCES users(id) ON DELETE RESTRICT,
    reason text NOT NULL DEFAULT '',
    expires_at timestamptz,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT title_grants_reason_length_check CHECK (
        char_length(reason) <= 500
    ),

    CONSTRAINT title_grants_expiry_check CHECK (
        expires_at IS NULL OR expires_at > created_at
    ),

    CONSTRAINT title_grants_revoke_check CHECK (
        revoked_at IS NULL OR revoked_at >= created_at
    )
);

CREATE INDEX title_grants_user_active_idx
    ON title_grants (user_id, revoked_at, expires_at, created_at DESC, id DESC);

CREATE INDEX title_grants_title_idx
    ON title_grants (title_id, created_at DESC, id DESC);

ALTER TABLE user_progressions
    ADD CONSTRAINT user_progressions_active_title_grant_fk
    FOREIGN KEY (active_title_grant_id)
    REFERENCES title_grants(id)
    ON DELETE SET NULL;
