-- +goose Up
CREATE TABLE availability_settings (
    organization_id UUID PRIMARY KEY REFERENCES organization(id) ON DELETE CASCADE,
    timezone TEXT NOT NULL DEFAULT 'Asia/Tehran',
    max_interviews_per_day INT NOT NULL DEFAULT 8,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE availability_working_hours (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    day_of_week SMALLINT NOT NULL CHECK (day_of_week BETWEEN 0 AND 6),
    start_time TIME NOT NULL CHECK (start_time < end_time),
    end_time TIME NOT NULL CHECK (end_time > start_time),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id, day_of_week, start_time, end_time)
);

CREATE TABLE availability_breaks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    day_of_week SMALLINT NOT NULL CHECK (day_of_week BETWEEN 0 AND 6),
    start_time TIME NOT NULL,
    end_time TIME NOT NULL CHECK (end_time > start_time)
);

CREATE TABLE availability_time_off (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    start_date DATE NOT NULL,
    end_at TIMESTAMPTZ NOT NULL CHECK (end_at > start_date::timestamptz)
);

CREATE TABLE slots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    interview_type_id UUID NOT NULL REFERENCES interview_type(id) ON DELETE CASCADE,
    start_at TIMESTAMPTZ NOT NULL,
    end_at TIMESTAMPTZ NOT NULL CHECK (end_at > start_at),
    booked BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (interview_type_id, start_at)
);

CREATE INDEX idx_slots_org_start ON slots(organization_id, start_at);
CREATE INDEX idx_slots_type_start ON slots(interview_type_id, start_at) WHERE booked = FALSE;

-- +goose Down
DROP INDEX IF EXISTS idx_slots_type_start;
DROP INDEX IF EXISTS idx_slots_org_start;
DROP TABLE slots;
DROP TABLE availability_time_off;
DROP TABLE availability_breaks;
DROP TABLE availability_working_hours;
DROP TABLE availability_settings;
