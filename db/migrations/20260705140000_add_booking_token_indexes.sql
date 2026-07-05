-- +goose Up
CREATE UNIQUE INDEX idx_booking_reschedule_token ON booking(reschedule_token);
CREATE UNIQUE INDEX idx_booking_cancel_token ON booking(cancel_token);

-- +goose Down
DROP INDEX IF EXISTS idx_booking_cancel_token;
DROP INDEX IF EXISTS idx_booking_reschedule_token;
