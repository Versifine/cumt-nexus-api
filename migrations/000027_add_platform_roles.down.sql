DROP INDEX IF EXISTS users_platform_role_idx;

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_platform_role_check;

ALTER TABLE users
    DROP COLUMN IF EXISTS platform_role;
