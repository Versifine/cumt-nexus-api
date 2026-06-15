ALTER TABLE user_progressions
    DROP CONSTRAINT IF EXISTS user_progressions_active_title_grant_fk;

DROP INDEX IF EXISTS title_grants_title_idx;
DROP INDEX IF EXISTS title_grants_user_active_idx;
DROP TABLE IF EXISTS title_grants;

DROP INDEX IF EXISTS titles_scope_active_idx;
DROP TABLE IF EXISTS titles;

DROP INDEX IF EXISTS xp_events_user_source_day_idx;
DROP INDEX IF EXISTS xp_events_user_created_idx;
DROP INDEX IF EXISTS xp_events_user_source_unique_idx;
DROP TABLE IF EXISTS xp_events;

DROP TABLE IF EXISTS xp_event_claims;
DROP TABLE IF EXISTS user_progressions;
