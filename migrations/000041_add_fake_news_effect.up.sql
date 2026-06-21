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
VALUES (
    'fake_news',
    'FakeNews',
    '',
    8,
    '',
    'fake_news',
    '📰',
    true,
    now(),
    now()
)
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
