CREATE TABLE posts (
    id uuid PRIMARY KEY,
    community_id uuid NOT NULL REFERENCES communities(id) ON DELETE RESTRICT,
    author_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    title text NOT NULL,
    body text NOT NULL,
    status text NOT NULL DEFAULT 'visible',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT posts_title_not_blank_check CHECK (
        btrim(title) <> ''
    ),

    CONSTRAINT posts_body_not_blank_check CHECK (
        btrim(body) <> ''
    ),

    CONSTRAINT posts_status_check CHECK (
        status IN ('visible', 'removed', 'deleted', 'locked', 'hidden')
    )
);

CREATE INDEX posts_community_visible_created_at_idx
    ON posts (community_id, created_at DESC, id DESC)
    WHERE status = 'visible';

CREATE INDEX posts_author_id_idx
    ON posts (author_id);
