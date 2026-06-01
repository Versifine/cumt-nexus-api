CREATE TABLE post_votes (
    post_id uuid NOT NULL REFERENCES posts(id) ON DELETE RESTRICT,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    value smallint NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    PRIMARY KEY (post_id, user_id),

    CONSTRAINT post_votes_value_check CHECK (
        value IN (-1, 1)
    ),

    CONSTRAINT post_votes_updated_at_check CHECK (
        updated_at >= created_at
    )
);

CREATE INDEX post_votes_post_value_idx
    ON post_votes (post_id, value);

CREATE INDEX post_votes_user_id_idx
    ON post_votes (user_id);
