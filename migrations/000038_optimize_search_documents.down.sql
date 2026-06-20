DROP INDEX IF EXISTS communities_public_search_document_idx;
DROP INDEX IF EXISTS posts_visible_search_document_idx;

ALTER TABLE communities
    DROP COLUMN IF EXISTS search_document;

ALTER TABLE posts
    DROP COLUMN IF EXISTS search_document;
