CREATE TABLE comments (
    id uuid PRIMARY KEY,
    post_id uuid NOT NULL REFERENCES posts(id) ON DELETE RESTRICT,
    author_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    parent_id uuid REFERENCES comments(id) ON DELETE RESTRICT,
    body text NOT NULL,
    status text NOT NULL DEFAULT 'visible',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT comments_parent_not_self_check CHECK (
        parent_id IS NULL OR parent_id <> id
    ),

    CONSTRAINT comments_body_not_blank_check CHECK (
        btrim(body) <> ''
    ),

    CONSTRAINT comments_status_check CHECK (
        status IN ('visible', 'removed', 'deleted', 'locked', 'hidden')
    )
);

CREATE INDEX comments_post_visible_created_at_idx
    ON comments (post_id, created_at DESC, id DESC)
    WHERE status = 'visible';

CREATE INDEX comments_parent_id_idx
    ON comments (parent_id)
    WHERE parent_id IS NOT NULL;

CREATE INDEX comments_author_id_idx
    ON comments (author_id);
