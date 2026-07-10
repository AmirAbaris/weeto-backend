-- +goose Up
CREATE TYPE booking_status AS ENUM ('scheduled', 'cancelled');

CREATE TABLE booking (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    interview_type_id UUID NOT NULL REFERENCES interview_type(id) ON DELETE CASCADE,
    slot_id UUID NOT NULL REFERENCES slots(id) ON DELETE RESTRICT,
    candidate_name TEXT NOT NULL,
    candidate_phone TEXT NOT NULL,
    candidate_email TEXT NOT NULL,
    status booking_status NOT NULL DEFAULT 'scheduled',
    meet_link TEXT,
    calendar_event_id TEXT,
    reschedule_token TEXT NOT NULL,
    cancel_token TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (slot_id)
);

CREATE INDEX idx_booking_organization_id ON booking(organization_id);
CREATE INDEX idx_booking_interview_type_id ON booking(interview_type_id);
CREATE INDEX idx_booking_status ON booking(status);

CREATE TYPE notification_event_type AS ENUM (
    'booking_created',
    'booking_rescheduled',
    'booking_cancelled',
    'reminder_24h'
);

CREATE TYPE notification_status AS ENUM ('pending', 'sent', 'failed');

CREATE TABLE notification_outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    event_type notification_event_type NOT NULL,
    payload JSONB NOT NULL,
    status notification_status NOT NULL DEFAULT 'pending',
    retry_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ
);

CREATE INDEX idx_notification_outbox_pending ON notification_outbox(status) WHERE status = 'pending';

-- +goose Down
DROP INDEX IF EXISTS idx_notification_outbox_pending;
DROP TABLE notification_outbox;
DROP TYPE notification_status;
DROP TYPE notification_event_type;

DROP INDEX IF EXISTS idx_booking_status;
DROP INDEX IF EXISTS idx_booking_interview_type_id;
DROP INDEX IF EXISTS idx_booking_organization_id;
DROP TABLE booking;
DROP TYPE booking_status;
