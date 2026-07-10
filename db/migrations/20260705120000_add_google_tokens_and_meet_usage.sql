-- +goose Up
ALTER TABLE users
    ADD COLUMN google_id TEXT,
    ADD COLUMN google_refresh_token TEXT;

ALTER TABLE organization
    ADD COLUMN meet_links_used INT NOT NULL DEFAULT 0,
    ADD COLUMN meet_links_period_start TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- +goose Down
ALTER TABLE organization
    DROP COLUMN meet_links_period_start,
    DROP COLUMN meet_links_used;

ALTER TABLE users
    DROP COLUMN google_refresh_token,
    DROP COLUMN google_id;
