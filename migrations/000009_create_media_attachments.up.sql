CREATE TABLE media_attachments (
    id uuid PRIMARY KEY,
    owner_type text NOT NULL DEFAULT 'none',
    owner_id uuid,
    uploader_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    kind text NOT NULL,
    storage_provider text NOT NULL,
    bucket text NOT NULL,
    object_key text NOT NULL,
    public_url text NOT NULL,
    thumbnail_object_key text,
    width integer,
    height integer,
    size_bytes bigint NOT NULL,
    mime_type text NOT NULL,
    alt_text text,
    status text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT media_attachments_owner_type_check CHECK (
        owner_type IN ('none', 'post', 'comment')
    ),

    CONSTRAINT media_attachments_owner_id_check CHECK (
        (owner_type = 'none' AND owner_id IS NULL)
        OR (owner_type IN ('post', 'comment') AND owner_id IS NOT NULL)
    ),

    CONSTRAINT media_attachments_kind_check CHECK (
        kind IN ('image')
    ),

    CONSTRAINT media_attachments_storage_provider_check CHECK (
        storage_provider IN ('r2', 'local')
    ),

    CONSTRAINT media_attachments_bucket_not_blank_check CHECK (
        btrim(bucket) <> ''
    ),

    CONSTRAINT media_attachments_object_key_not_blank_check CHECK (
        btrim(object_key) <> ''
    ),

    CONSTRAINT media_attachments_public_url_not_blank_check CHECK (
        btrim(public_url) <> ''
    ),

    CONSTRAINT media_attachments_size_bytes_positive_check CHECK (
        size_bytes > 0
    ),

    CONSTRAINT media_attachments_dimensions_positive_check CHECK (
        (width IS NULL OR width > 0)
        AND (height IS NULL OR height > 0)
    ),

    CONSTRAINT media_attachments_mime_type_check CHECK (
        mime_type IN ('image/jpeg', 'image/png', 'image/webp')
    ),

    CONSTRAINT media_attachments_alt_text_length_check CHECK (
        alt_text IS NULL OR char_length(alt_text) <= 200
    ),

    CONSTRAINT media_attachments_status_check CHECK (
        status IN ('pending', 'ready', 'blocked', 'failed')
    ),

    CONSTRAINT media_attachments_updated_after_created_check CHECK (
        updated_at >= created_at
    ),

    CONSTRAINT media_attachments_storage_object_unique UNIQUE (
        storage_provider,
        bucket,
        object_key
    )
);

CREATE INDEX media_attachments_uploader_status_created_at_idx
    ON media_attachments (uploader_id, status, created_at DESC, id DESC);

CREATE INDEX media_attachments_owner_idx
    ON media_attachments (owner_type, owner_id)
    WHERE owner_id IS NOT NULL;
