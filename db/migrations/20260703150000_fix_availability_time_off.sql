-- +goose Up
ALTER TABLE availability_time_off
    ADD COLUMN start_at TIMESTAMPTZ;

UPDATE availability_time_off
SET start_at = start_date::timestamptz
WHERE start_at IS NULL;

ALTER TABLE availability_time_off
    DROP COLUMN start_date;

ALTER TABLE availability_time_off
    ALTER COLUMN start_at SET NOT NULL;

ALTER TABLE availability_time_off
    DROP CONSTRAINT IF EXISTS availability_time_off_end_at_check;

ALTER TABLE availability_time_off
    ADD CONSTRAINT availability_time_off_range CHECK (end_at > start_at);

-- +goose Down
ALTER TABLE availability_time_off
    DROP CONSTRAINT IF EXISTS availability_time_off_range;

ALTER TABLE availability_time_off
    ADD COLUMN start_date DATE;

UPDATE availability_time_off
SET start_date = (start_at AT TIME ZONE 'UTC')::date;

ALTER TABLE availability_time_off
    DROP COLUMN start_at;

ALTER TABLE availability_time_off
    ALTER COLUMN start_date SET NOT NULL;

ALTER TABLE availability_time_off
    ADD CONSTRAINT availability_time_off_end_at_check CHECK (end_at > start_date::timestamptz);
