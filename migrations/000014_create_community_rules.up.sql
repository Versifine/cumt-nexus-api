CREATE TABLE community_rules (
    id uuid PRIMARY KEY,
    community_id uuid NOT NULL REFERENCES communities(id) ON DELETE CASCADE,
    title text NOT NULL,
    body text NOT NULL DEFAULT '',
    position integer NOT NULL,
    created_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    updated_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT community_rules_title_not_blank_check CHECK (
        btrim(title) <> ''
    ),

    CONSTRAINT community_rules_position_check CHECK (
        position >= 0
    )
);

CREATE INDEX community_rules_community_order_idx
    ON community_rules (community_id, position ASC, created_at ASC, id ASC);
