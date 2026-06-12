ALTER TABLE users
    ADD COLUMN banner_url text NOT NULL DEFAULT '',
    ADD CONSTRAINT users_banner_url_scheme_check CHECK (
        banner_url = ''
        OR banner_url ~ '^https?://'
    );
