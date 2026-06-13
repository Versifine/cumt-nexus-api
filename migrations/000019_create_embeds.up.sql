CREATE TABLE embeds (
    id uuid PRIMARY KEY,
    provider text NOT NULL,
    provider_ref text NOT NULL,
    url text NOT NULL,
    canonical_url text NOT NULL,
    embed_url text NOT NULL,
    iframe_allowed boolean NOT NULL DEFAULT false,
    title text NOT NULL DEFAULT '',
    description text NOT NULL DEFAULT '',
    image_url text NOT NULL DEFAULT '',
    author_name text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT 'ready',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT embeds_provider_check CHECK (
        provider IN ('bilibili_video', 'douyin_video', 'netease_music', 'qq_music')
    ),
    CONSTRAINT embeds_status_check CHECK (
        status IN ('ready', 'unavailable')
    ),
    CONSTRAINT embeds_provider_ref_not_blank_check CHECK (
        btrim(provider_ref) <> ''
    ),
    CONSTRAINT embeds_url_not_blank_check CHECK (
        btrim(url) <> ''
    ),
    CONSTRAINT embeds_canonical_url_not_blank_check CHECK (
        btrim(canonical_url) <> ''
    ),
    CONSTRAINT embeds_embed_url_not_blank_check CHECK (
        btrim(embed_url) <> ''
    )
);

CREATE UNIQUE INDEX embeds_provider_ref_uq
    ON embeds (provider, provider_ref);

