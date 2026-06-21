UPDATE effects
SET
    is_active = false,
    updated_at = now()
WHERE id = 'fake_news';
