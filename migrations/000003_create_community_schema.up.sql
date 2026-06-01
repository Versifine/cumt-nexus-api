ALTER TABLE users
    ADD COLUMN is_platform_staff boolean NOT NULL DEFAULT false;

CREATE TABLE communities (
    id uuid PRIMARY KEY,
    slug text NOT NULL,
    name text NOT NULL,
    description text NOT NULL DEFAULT '',
    kind text NOT NULL,
    status text NOT NULL DEFAULT 'active',
    visibility text NOT NULL DEFAULT 'public',
    created_by uuid REFERENCES users(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT communities_slug_uq UNIQUE (slug),

    CONSTRAINT communities_slug_format_check CHECK (
        slug ~ '^[a-z0-9][a-z0-9-]{2,31}$'
    ),

    CONSTRAINT communities_name_not_blank_check CHECK (
        btrim(name) <> ''
    ),

    CONSTRAINT communities_kind_check CHECK (
        kind IN ('system', 'user_created')
    ),

    CONSTRAINT communities_status_check CHECK (
        status IN ('active', 'suspended', 'archived')
    ),

    CONSTRAINT communities_visibility_check CHECK (
        visibility IN ('public')
    ),

    CONSTRAINT communities_created_by_required_check CHECK (
        kind <> 'user_created' OR created_by IS NOT NULL
    )
);

CREATE TABLE community_memberships (
    community_id uuid NOT NULL REFERENCES communities(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role text NOT NULL,
    status text NOT NULL DEFAULT 'active',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT community_memberships_pk PRIMARY KEY (community_id, user_id),

    CONSTRAINT community_memberships_role_check CHECK (
        role IN ('owner', 'moderator', 'member')
    ),

    CONSTRAINT community_memberships_status_check CHECK (
        status IN ('active', 'left', 'banned')
    )
);

CREATE INDEX community_memberships_user_id_idx
    ON community_memberships (user_id);

CREATE TABLE community_applications (
    id uuid PRIMARY KEY,
    applicant_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    requested_slug text NOT NULL,
    requested_name text NOT NULL,
    reason text NOT NULL,
    status text NOT NULL DEFAULT 'pending',
    reviewed_by uuid REFERENCES users(id) ON DELETE RESTRICT,
    reviewed_at timestamptz,
    reject_reason text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT community_applications_requested_slug_format_check CHECK (
        requested_slug ~ '^[a-z0-9][a-z0-9-]{2,31}$'
    ),

    CONSTRAINT community_applications_requested_name_not_blank_check CHECK (
        btrim(requested_name) <> ''
    ),

    CONSTRAINT community_applications_reason_not_blank_check CHECK (
        btrim(reason) <> ''
    ),

    CONSTRAINT community_applications_status_check CHECK (
        status IN ('pending', 'approved', 'rejected', 'canceled')
    ),

    CONSTRAINT community_applications_review_fields_check CHECK (
        (
            status IN ('pending', 'canceled')
            AND reviewed_by IS NULL
            AND reviewed_at IS NULL
            AND reject_reason IS NULL
        )
        OR (
            status = 'approved'
            AND reviewed_by IS NOT NULL
            AND reviewed_at IS NOT NULL
            AND reject_reason IS NULL
        )
        OR (
            status = 'rejected'
            AND reviewed_by IS NOT NULL
            AND reviewed_at IS NOT NULL
            AND reject_reason IS NOT NULL
            AND btrim(reject_reason) <> ''
        )
    )
);

CREATE INDEX community_applications_applicant_id_idx
    ON community_applications (applicant_id);

CREATE INDEX community_applications_reviewed_by_idx
    ON community_applications (reviewed_by)
    WHERE reviewed_by IS NOT NULL;

CREATE UNIQUE INDEX community_applications_pending_slug_uq
    ON community_applications (requested_slug)
    WHERE status = 'pending';
