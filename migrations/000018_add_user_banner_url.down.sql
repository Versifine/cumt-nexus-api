ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_banner_url_scheme_check,
    DROP COLUMN IF EXISTS banner_url;
