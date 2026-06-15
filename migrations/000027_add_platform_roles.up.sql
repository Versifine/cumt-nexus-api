ALTER TABLE users
    ADD COLUMN platform_role text;

UPDATE users
SET platform_role = 'owner'
WHERE is_platform_staff = true
    AND platform_role IS NULL;

ALTER TABLE users
    ADD CONSTRAINT users_platform_role_check
    CHECK (platform_role IS NULL OR platform_role IN ('owner', 'admin', 'staff'));

CREATE INDEX users_platform_role_idx
    ON users (platform_role)
    WHERE platform_role IS NOT NULL;
