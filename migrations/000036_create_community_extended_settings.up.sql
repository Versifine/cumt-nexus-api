CREATE TABLE community_post_flairs (
    id uuid PRIMARY KEY,
    community_id uuid NOT NULL REFERENCES communities(id) ON DELETE CASCADE,
    title text NOT NULL,
    color text NOT NULL DEFAULT '',
    is_user_selectable boolean NOT NULL DEFAULT false,
    is_enabled boolean NOT NULL DEFAULT true,
    position integer NOT NULL DEFAULT 0,
    created_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    updated_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT community_post_flairs_title_not_blank_check CHECK (btrim(title) <> ''),
    CONSTRAINT community_post_flairs_title_length_check CHECK (char_length(title) <= 80),
    CONSTRAINT community_post_flairs_color_length_check CHECK (char_length(color) <= 32),
    CONSTRAINT community_post_flairs_position_check CHECK (position >= 0)
);

CREATE INDEX community_post_flairs_community_order_idx
    ON community_post_flairs (community_id, position ASC, created_at ASC, id ASC);

CREATE TABLE community_user_flairs (
    id uuid PRIMARY KEY,
    community_id uuid NOT NULL REFERENCES communities(id) ON DELETE CASCADE,
    title text NOT NULL,
    color text NOT NULL DEFAULT '',
    is_user_selectable boolean NOT NULL DEFAULT false,
    is_enabled boolean NOT NULL DEFAULT true,
    position integer NOT NULL DEFAULT 0,
    created_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    updated_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT community_user_flairs_title_not_blank_check CHECK (btrim(title) <> ''),
    CONSTRAINT community_user_flairs_title_length_check CHECK (char_length(title) <= 80),
    CONSTRAINT community_user_flairs_color_length_check CHECK (char_length(color) <= 32),
    CONSTRAINT community_user_flairs_position_check CHECK (position >= 0)
);

CREATE INDEX community_user_flairs_community_order_idx
    ON community_user_flairs (community_id, position ASC, created_at ASC, id ASC);

CREATE TABLE community_scheduled_posts (
    id uuid PRIMARY KEY,
    community_id uuid NOT NULL REFERENCES communities(id) ON DELETE CASCADE,
    created_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    updated_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    title text NOT NULL,
    body text NOT NULL DEFAULT '',
    scheduled_at timestamptz NOT NULL,
    repeat_rule text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT 'scheduled',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT community_scheduled_posts_title_not_blank_check CHECK (btrim(title) <> ''),
    CONSTRAINT community_scheduled_posts_title_length_check CHECK (char_length(title) <= 160),
    CONSTRAINT community_scheduled_posts_body_length_check CHECK (char_length(body) <= 20000),
    CONSTRAINT community_scheduled_posts_repeat_rule_length_check CHECK (char_length(repeat_rule) <= 160),
    CONSTRAINT community_scheduled_posts_status_check CHECK (status IN ('scheduled', 'paused', 'published', 'cancelled'))
);

CREATE INDEX community_scheduled_posts_community_time_idx
    ON community_scheduled_posts (community_id, scheduled_at ASC, id ASC);

CREATE TABLE community_guides (
    id uuid PRIMARY KEY,
    community_id uuid NOT NULL REFERENCES communities(id) ON DELETE CASCADE,
    created_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    updated_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    title text NOT NULL,
    body text NOT NULL DEFAULT '',
    position integer NOT NULL DEFAULT 0,
    visibility text NOT NULL DEFAULT 'public',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT community_guides_title_not_blank_check CHECK (btrim(title) <> ''),
    CONSTRAINT community_guides_title_length_check CHECK (char_length(title) <= 160),
    CONSTRAINT community_guides_body_length_check CHECK (char_length(body) <= 20000),
    CONSTRAINT community_guides_position_check CHECK (position >= 0),
    CONSTRAINT community_guides_visibility_check CHECK (visibility IN ('public', 'members', 'mods'))
);

CREATE INDEX community_guides_community_order_idx
    ON community_guides (community_id, position ASC, created_at ASC, id ASC);
