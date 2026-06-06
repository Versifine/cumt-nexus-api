CREATE TABLE post_saves (
    post_id uuid NOT NULL REFERENCES posts(id) ON DELETE RESTRICT,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),

    PRIMARY KEY (post_id, user_id)
);

CREATE INDEX post_saves_user_created_idx
    ON post_saves (user_id, created_at DESC, post_id DESC);

CREATE INDEX post_saves_post_id_idx
    ON post_saves (post_id);

CREATE TABLE community_follows (
    community_id uuid NOT NULL REFERENCES communities(id) ON DELETE RESTRICT,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),

    PRIMARY KEY (community_id, user_id)
);

CREATE INDEX community_follows_user_created_idx
    ON community_follows (user_id, created_at DESC, community_id DESC);

CREATE INDEX community_follows_community_id_idx
    ON community_follows (community_id);

CREATE TABLE comment_votes (
    comment_id uuid NOT NULL REFERENCES comments(id) ON DELETE RESTRICT,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    value smallint NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    PRIMARY KEY (comment_id, user_id),

    CONSTRAINT comment_votes_value_check CHECK (
        value IN (-1, 1)
    ),

    CONSTRAINT comment_votes_updated_at_check CHECK (
        updated_at >= created_at
    )
);

CREATE INDEX comment_votes_comment_value_idx
    ON comment_votes (comment_id, value);

CREATE INDEX comment_votes_user_id_idx
    ON comment_votes (user_id);
