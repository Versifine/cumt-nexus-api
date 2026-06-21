CREATE TABLE point_reward_claims (
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    source_type text NOT NULL,
    source_id text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),

    PRIMARY KEY (user_id, source_type, source_id),

    CONSTRAINT point_reward_claims_source_type_not_blank_check CHECK (
        btrim(source_type) <> ''
    ),

    CONSTRAINT point_reward_claims_source_id_not_blank_check CHECK (
        btrim(source_id) <> ''
    )
);

CREATE INDEX point_reward_claims_user_created_idx
    ON point_reward_claims (user_id, created_at DESC);
