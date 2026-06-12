ALTER TABLE users
    ADD COLUMN display_name text NOT NULL DEFAULT '',
    ADD COLUMN avatar_url text NOT NULL DEFAULT '',
    ADD COLUMN headline text NOT NULL DEFAULT '',
    ADD COLUMN bio text NOT NULL DEFAULT '',
    ADD CONSTRAINT users_display_name_length_check CHECK (char_length(display_name) <= 40),
    ADD CONSTRAINT users_headline_length_check CHECK (char_length(headline) <= 80),
    ADD CONSTRAINT users_bio_length_check CHECK (char_length(bio) <= 300),
    ADD CONSTRAINT users_avatar_url_scheme_check CHECK (
        avatar_url = ''
        OR avatar_url ~ '^https?://'
    );
