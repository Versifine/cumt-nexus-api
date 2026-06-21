DROP TABLE IF EXISTS post_effects;

DELETE FROM effects
WHERE id IN (
    'useful',
    'cant_hold',
    'classic',
    'following_up',
    'verified_true',
    'abstract',
    'godlike',
    'clown'
);

UPDATE effects
SET
    is_active = true,
    updated_at = now()
WHERE id IN ('sparkle', 'spotlight', 'campus_star', 'neon_ring');

ALTER TABLE effects
    DROP COLUMN IF EXISTS emoji;
