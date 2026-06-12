CREATE TABLE post_content_refs (
    post_id uuid NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    position integer NOT NULL,
    kind text NOT NULL,
    ref_id text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),

    PRIMARY KEY (post_id, position),

    CONSTRAINT post_content_refs_position_check CHECK (
        position >= 0
    ),

    CONSTRAINT post_content_refs_kind_check CHECK (
        kind IN ('image', 'link_preview', 'embed')
    ),

    CONSTRAINT post_content_refs_ref_id_not_blank_check CHECK (
        btrim(ref_id) <> ''
    )
);

CREATE INDEX post_content_refs_post_order_idx
    ON post_content_refs (post_id, position ASC);

CREATE TABLE comment_content_refs (
    comment_id uuid NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
    position integer NOT NULL,
    kind text NOT NULL,
    ref_id text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),

    PRIMARY KEY (comment_id, position),

    CONSTRAINT comment_content_refs_position_check CHECK (
        position >= 0
    ),

    CONSTRAINT comment_content_refs_kind_check CHECK (
        kind IN ('image', 'link_preview', 'embed')
    ),

    CONSTRAINT comment_content_refs_ref_id_not_blank_check CHECK (
        btrim(ref_id) <> ''
    )
);

CREATE INDEX comment_content_refs_comment_order_idx
    ON comment_content_refs (comment_id, position ASC);
