ALTER TABLE community_owner_transfers
    DROP CONSTRAINT IF EXISTS community_owner_transfers_acceptance_check,
    DROP CONSTRAINT IF EXISTS community_owner_transfers_status_check;

UPDATE community_owner_transfers
SET status = 'canceled',
    cancelled_at = NULL
WHERE status = 'cancelled';

ALTER TABLE community_owner_transfers
    ADD CONSTRAINT community_owner_transfers_status_check CHECK (
        status IN ('pending', 'accepted', 'canceled', 'expired')
    ),
    ADD CONSTRAINT community_owner_transfers_acceptance_check CHECK (
        (status = 'accepted' AND accepted_at IS NOT NULL)
        OR (status <> 'accepted' AND accepted_at IS NULL)
    ),
    DROP COLUMN IF EXISTS cancelled_at,
    DROP COLUMN IF EXISTS expires_at;

ALTER TABLE communities
    DROP CONSTRAINT IF EXISTS communities_banner_url_length_check,
    DROP CONSTRAINT IF EXISTS communities_avatar_url_length_check,
    DROP CONSTRAINT IF EXISTS communities_banner_url_scheme_check,
    DROP CONSTRAINT IF EXISTS communities_avatar_url_scheme_check,
    DROP COLUMN IF EXISTS banner_url,
    DROP COLUMN IF EXISTS avatar_url;
