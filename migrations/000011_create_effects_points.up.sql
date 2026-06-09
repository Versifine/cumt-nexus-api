CREATE TABLE effects (
    id text PRIMARY KEY,
    name text NOT NULL,
    description text NOT NULL,
    cost_points integer NOT NULL,
    asset_url text NOT NULL DEFAULT '',
    animation_key text NOT NULL,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT effects_id_format_check CHECK (
        id ~ '^[a-z0-9][a-z0-9_-]{0,63}$'
    ),

    CONSTRAINT effects_name_not_blank_check CHECK (
        btrim(name) <> ''
    ),

    CONSTRAINT effects_cost_points_non_negative_check CHECK (
        cost_points >= 0
    ),

    CONSTRAINT effects_animation_key_not_blank_check CHECK (
        btrim(animation_key) <> ''
    ),

    CONSTRAINT effects_updated_at_check CHECK (
        updated_at >= created_at
    )
);

CREATE INDEX effects_active_idx
    ON effects (is_active, cost_points, id);

INSERT INTO effects (
    id,
    name,
    description,
    cost_points,
    asset_url,
    animation_key,
    is_active
)
VALUES
    ('sparkle', 'Sparkle', 'A subtle sparkle burst for a comment.', 10, '', 'sparkle', true),
    ('spotlight', 'Spotlight', 'A soft highlight around a comment.', 20, '', 'spotlight', true),
    ('campus_star', 'Campus Star', 'A premium star sticker for standout comments.', 50, '', 'campus_star', true),
    ('neon_ring', 'Neon Ring', 'A premium animated ring for high-signal comments.', 80, '', 'neon_ring', true);

CREATE TABLE user_points (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE RESTRICT,
    balance integer NOT NULL,
    lifetime_earned integer NOT NULL,
    lifetime_spent integer NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT user_points_balance_non_negative_check CHECK (
        balance >= 0
    ),

    CONSTRAINT user_points_lifetime_earned_non_negative_check CHECK (
        lifetime_earned >= 0
    ),

    CONSTRAINT user_points_lifetime_spent_non_negative_check CHECK (
        lifetime_spent >= 0
    )
);

CREATE TABLE comment_effects (
    id uuid PRIMARY KEY,
    comment_id uuid NOT NULL REFERENCES comments(id) ON DELETE RESTRICT,
    effect_id text NOT NULL REFERENCES effects(id) ON DELETE RESTRICT,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    points_spent integer NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT comment_effects_points_spent_non_negative_check CHECK (
        points_spent >= 0
    )
);

CREATE INDEX comment_effects_comment_created_idx
    ON comment_effects (comment_id, created_at DESC, id DESC);

CREATE INDEX comment_effects_user_created_idx
    ON comment_effects (user_id, created_at DESC, id DESC);

CREATE TABLE point_transactions (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    delta integer NOT NULL,
    balance_after integer NOT NULL,
    reason text NOT NULL,
    source_type text NOT NULL,
    source_id text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT point_transactions_delta_not_zero_check CHECK (
        delta <> 0
    ),

    CONSTRAINT point_transactions_balance_after_non_negative_check CHECK (
        balance_after >= 0
    ),

    CONSTRAINT point_transactions_reason_not_blank_check CHECK (
        btrim(reason) <> ''
    ),

    CONSTRAINT point_transactions_source_type_not_blank_check CHECK (
        btrim(source_type) <> ''
    ),

    CONSTRAINT point_transactions_source_id_not_blank_check CHECK (
        btrim(source_id) <> ''
    )
);

CREATE INDEX point_transactions_user_created_idx
    ON point_transactions (user_id, created_at DESC, id DESC);
