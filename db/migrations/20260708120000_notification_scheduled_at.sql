-- +goose Up
ALTER TYPE notification_status ADD VALUE IF NOT EXISTS 'cancelled';

ALTER TABLE notification_outbox
    ADD COLUMN scheduled_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

UPDATE notification_outbox
SET scheduled_at = created_at
WHERE scheduled_at IS NOT NULL;

DROP INDEX IF EXISTS idx_notification_outbox_pending;

CREATE INDEX idx_notification_outbox_pending ON notification_outbox(status, scheduled_at)
    WHERE status = 'pending';

-- +goose Down
DROP INDEX IF EXISTS idx_notification_outbox_pending;

CREATE INDEX idx_notification_outbox_pending ON notification_outbox(status)
    WHERE status = 'pending';

ALTER TABLE notification_outbox
    DROP COLUMN IF EXISTS scheduled_at;

-- PostgreSQL does not support removing enum values; cancelled rows would need manual cleanup before down.
