ALTER TABLE posts
    ADD COLUMN IF NOT EXISTS search_document tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('simple', COALESCE(title, '')), 'A') ||
        setweight(to_tsvector('simple', COALESCE(body, '')), 'C')
    ) STORED;

ALTER TABLE communities
    ADD COLUMN IF NOT EXISTS search_document tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('simple', COALESCE(name, '')), 'A') ||
        setweight(to_tsvector('simple', COALESCE(description, '')), 'B') ||
        setweight(to_tsvector('simple', COALESCE(slug, '')), 'B')
    ) STORED;

CREATE INDEX IF NOT EXISTS posts_visible_search_document_idx
    ON posts USING GIN (search_document)
    WHERE status = 'visible';

CREATE INDEX IF NOT EXISTS communities_public_search_document_idx
    ON communities USING GIN (search_document)
    WHERE status = 'active'
        AND visibility = 'public';
