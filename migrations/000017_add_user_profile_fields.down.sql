ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_avatar_url_scheme_check,
    DROP CONSTRAINT IF EXISTS users_bio_length_check,
    DROP CONSTRAINT IF EXISTS users_headline_length_check,
    DROP CONSTRAINT IF EXISTS users_display_name_length_check,
    DROP COLUMN IF EXISTS bio,
    DROP COLUMN IF EXISTS headline,
    DROP COLUMN IF EXISTS avatar_url,
    DROP COLUMN IF EXISTS display_name;
