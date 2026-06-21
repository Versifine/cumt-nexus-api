ALTER TABLE effects
    ADD COLUMN emoji text NOT NULL DEFAULT '';

UPDATE effects
SET
    is_active = false,
    updated_at = now()
WHERE id IN ('sparkle', 'spotlight', 'campus_star', 'neon_ring');

INSERT INTO effects (
    id,
    name,
    description,
    cost_points,
    asset_url,
    animation_key,
    emoji,
    is_active,
    created_at,
    updated_at
)
VALUES
    ('useful', '有用', '信息有效、资料可参考。', 5, '', 'useful', '👍', true, now(), now()),
    ('cant_hold', '难绷', '好笑、离谱但不攻击。', 5, '', 'cant_hold', '😂', true, now(), now()),
    ('classic', '经典', '典型发言、值得收藏式标记。', 8, '', 'classic', '🏆', true, now(), now()),
    ('following_up', '蹲后续', '等更新、等结果。', 5, '', 'following_up', '👀', true, now(), now()),
    ('verified_true', '鉴定为真', '认可事实性或经验真实性。', 8, '', 'verified_true', '✅', true, now(), now()),
    ('abstract', '抽象', '表达混沌、反常、难评。', 5, '', 'abstract', '🌀', true, now(), now()),
    ('godlike', '神', '高质量、很强、神来之笔。', 12, '', 'godlike', '👑', true, now(), now()),
    ('clown', '小丑', '荒诞、自嘲式翻车或反讽，避免人身攻击。', 5, '', 'clown', '🤡', true, now(), now())
ON CONFLICT (id) DO UPDATE
SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    cost_points = EXCLUDED.cost_points,
    asset_url = EXCLUDED.asset_url,
    animation_key = EXCLUDED.animation_key,
    emoji = EXCLUDED.emoji,
    is_active = true,
    updated_at = EXCLUDED.updated_at;

CREATE TABLE post_effects (
    id uuid PRIMARY KEY,
    post_id uuid NOT NULL REFERENCES posts(id) ON DELETE RESTRICT,
    effect_id text NOT NULL REFERENCES effects(id) ON DELETE RESTRICT,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    points_spent integer NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT post_effects_points_spent_non_negative_check CHECK (
        points_spent >= 0
    )
);

CREATE INDEX post_effects_post_created_idx
    ON post_effects (post_id, created_at DESC, id DESC);

CREATE INDEX post_effects_user_created_idx
    ON post_effects (user_id, created_at DESC, id DESC);
