CREATE TABLE users (
    id uuid PRIMARY KEY,
    username text NOT NULL,
    password_hash text NOT NULL,
    status text NOT NULL DEFAULT 'active',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT users_username_uq UNIQUE (username),

    CONSTRAINT users_username_format_check CHECK (
        username ~ '^[a-z0-9_]{3,32}$'
    ),

    CONSTRAINT users_status_check CHECK (
        status IN ('active', 'disabled')
    )
);