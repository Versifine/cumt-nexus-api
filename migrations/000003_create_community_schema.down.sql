DROP TABLE IF EXISTS community_applications;

DROP TABLE IF EXISTS community_memberships;

DROP TABLE IF EXISTS communities;

ALTER TABLE users
    DROP COLUMN IF EXISTS is_platform_staff;
