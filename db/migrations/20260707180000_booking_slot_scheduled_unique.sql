-- +goose Up
ALTER TABLE booking DROP CONSTRAINT booking_slot_id_key;

CREATE UNIQUE INDEX idx_booking_slot_scheduled ON booking (slot_id) WHERE status = 'scheduled';

-- +goose Down
DROP INDEX IF EXISTS idx_booking_slot_scheduled;

ALTER TABLE booking ADD CONSTRAINT booking_slot_id_key UNIQUE (slot_id);
