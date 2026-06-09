CREATE INDEX communities_public_search_fts_idx
    ON communities
    USING GIN ((
        setweight(to_tsvector('simple', COALESCE(name, '')), 'A') ||
        setweight(to_tsvector('simple', COALESCE(description, '')), 'B')
    ))
    WHERE status = 'active'
        AND visibility = 'public';

CREATE INDEX posts_visible_search_fts_idx
    ON posts
    USING GIN ((
        setweight(to_tsvector('simple', COALESCE(title, '')), 'A') ||
        setweight(to_tsvector('simple', COALESCE(body, '')), 'C')
    ))
    WHERE status = 'visible';
