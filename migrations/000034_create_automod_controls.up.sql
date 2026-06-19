CREATE TABLE community_automod_configs (
    community_id uuid PRIMARY KEY REFERENCES communities(id) ON DELETE CASCADE,
    config_text text NOT NULL DEFAULT '',
    rules jsonb NOT NULL DEFAULT '{}'::jsonb,
    version integer NOT NULL DEFAULT 1,
    updated_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT community_automod_configs_version_check CHECK (version > 0),
    CONSTRAINT community_automod_configs_config_text_length_check CHECK (char_length(config_text) <= 20000)
);

CREATE TABLE community_automod_config_versions (
    id uuid PRIMARY KEY,
    community_id uuid NOT NULL REFERENCES communities(id) ON DELETE CASCADE,
    version integer NOT NULL,
    config_text text NOT NULL DEFAULT '',
    rules jsonb NOT NULL DEFAULT '{}'::jsonb,
    updated_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT community_automod_config_versions_version_check CHECK (version > 0),
    CONSTRAINT community_automod_config_versions_config_text_length_check CHECK (char_length(config_text) <= 20000),
    CONSTRAINT community_automod_config_versions_unique_version UNIQUE (community_id, version)
);

CREATE INDEX community_automod_config_versions_community_version_idx
    ON community_automod_config_versions (community_id, version DESC);

CREATE TABLE community_content_controls (
    community_id uuid PRIMARY KEY REFERENCES communities(id) ON DELETE CASCADE,
    blocked_keywords jsonb NOT NULL DEFAULT '[]'::jsonb,
    blocked_domains jsonb NOT NULL DEFAULT '[]'::jsonb,
    min_account_age_days integer NOT NULL DEFAULT 0,
    post_rate_limit_per_hour integer NOT NULL DEFAULT 0,
    comment_rate_limit_per_hour integer NOT NULL DEFAULT 0,
    block_new_accounts boolean NOT NULL DEFAULT false,
    filter_links boolean NOT NULL DEFAULT false,
    updated_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT community_content_controls_min_account_age_check CHECK (min_account_age_days >= 0),
    CONSTRAINT community_content_controls_post_rate_limit_check CHECK (post_rate_limit_per_hour >= 0),
    CONSTRAINT community_content_controls_comment_rate_limit_check CHECK (comment_rate_limit_per_hour >= 0),
    CONSTRAINT community_content_controls_keywords_array_check CHECK (jsonb_typeof(blocked_keywords) = 'array'),
    CONSTRAINT community_content_controls_domains_array_check CHECK (jsonb_typeof(blocked_domains) = 'array')
);
