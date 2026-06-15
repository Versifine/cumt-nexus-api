ALTER TABLE communities
    ADD COLUMN IF NOT EXISTS avatar_url text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS banner_url text NOT NULL DEFAULT '';

ALTER TABLE communities
    DROP CONSTRAINT IF EXISTS communities_avatar_url_scheme_check,
    DROP CONSTRAINT IF EXISTS communities_banner_url_scheme_check,
    DROP CONSTRAINT IF EXISTS communities_avatar_url_length_check,
    DROP CONSTRAINT IF EXISTS communities_banner_url_length_check;

ALTER TABLE communities
    ADD CONSTRAINT communities_avatar_url_scheme_check CHECK (
        avatar_url = ''
        OR avatar_url ~ '^https?://'
    ),
    ADD CONSTRAINT communities_banner_url_scheme_check CHECK (
        banner_url = ''
        OR banner_url ~ '^https?://'
    ),
    ADD CONSTRAINT communities_avatar_url_length_check CHECK (octet_length(avatar_url) <= 2048),
    ADD CONSTRAINT communities_banner_url_length_check CHECK (octet_length(banner_url) <= 2048);

ALTER TABLE community_owner_transfers
    ADD COLUMN IF NOT EXISTS expires_at timestamptz,
    ADD COLUMN IF NOT EXISTS cancelled_at timestamptz;

UPDATE community_owner_transfers
SET expires_at = created_at + interval '48 hours'
WHERE expires_at IS NULL;

ALTER TABLE community_owner_transfers
    DROP CONSTRAINT IF EXISTS community_owner_transfers_status_check,
    DROP CONSTRAINT IF EXISTS community_owner_transfers_acceptance_check;

UPDATE community_owner_transfers
SET status = 'cancelled',
    cancelled_at = COALESCE(cancelled_at, updated_at)
WHERE status = 'canceled';

ALTER TABLE community_owner_transfers
    ALTER COLUMN expires_at SET NOT NULL;

ALTER TABLE community_owner_transfers
    ADD CONSTRAINT community_owner_transfers_status_check CHECK (
        status IN ('pending', 'accepted', 'cancelled', 'expired')
    ),
    ADD CONSTRAINT community_owner_transfers_acceptance_check CHECK (
        (status = 'accepted' AND accepted_at IS NOT NULL AND cancelled_at IS NULL)
        OR (status = 'cancelled' AND accepted_at IS NULL AND cancelled_at IS NOT NULL)
        OR (status IN ('pending', 'expired') AND accepted_at IS NULL AND cancelled_at IS NULL)
    );
