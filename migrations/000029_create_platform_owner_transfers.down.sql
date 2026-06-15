DROP INDEX IF EXISTS platform_owner_transfers_initiator_idx;
DROP INDEX IF EXISTS platform_owner_transfers_target_idx;
DROP INDEX IF EXISTS platform_owner_transfers_pending_unique_idx;
DROP TABLE IF EXISTS platform_owner_transfers;

DROP INDEX IF EXISTS users_active_platform_owner_unique_idx;

DROP INDEX IF EXISTS admin_audit_logs_actor_ref_idx;
ALTER TABLE admin_audit_logs
    DROP CONSTRAINT IF EXISTS admin_audit_logs_actor_ref_ck;

DELETE FROM admin_audit_logs
WHERE actor_id IS NULL;

ALTER TABLE admin_audit_logs
    ALTER COLUMN actor_id SET NOT NULL,
    DROP COLUMN IF EXISTS actor_ref;
