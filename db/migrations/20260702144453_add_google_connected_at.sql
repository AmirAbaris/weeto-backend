-- +goose Up
ALTER TABLE users ADD COLUMN google_connected_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE users DROP COLUMN google_connected_at;
