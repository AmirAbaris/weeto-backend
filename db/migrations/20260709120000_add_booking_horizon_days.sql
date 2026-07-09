-- +goose Up
ALTER TABLE availability_settings
  ADD COLUMN booking_horizon_days INT NOT NULL DEFAULT 14
  CHECK (booking_horizon_days BETWEEN 1 AND 90);

-- +goose Down
ALTER TABLE availability_settings
  DROP COLUMN booking_horizon_days;
