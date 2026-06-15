CREATE TABLE IF NOT EXISTS user_follows (
    follower_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    following_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT user_follows_pk PRIMARY KEY (follower_id, following_id),
    CONSTRAINT user_follows_no_self_ck CHECK (follower_id <> following_id)
);

CREATE INDEX IF NOT EXISTS user_follows_following_created_idx
    ON user_follows (following_id, created_at DESC);

CREATE INDEX IF NOT EXISTS user_follows_follower_created_idx
    ON user_follows (follower_id, created_at DESC);
