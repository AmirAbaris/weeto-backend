-- +goose Up
CREATE TYPE meeting_provider AS ENUM ('google_meet', 'bale_link', 'custom_url');

CREATE TABLE interview_type (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    slug TEXT NOT NULL,
    duration_minutes INT NOT NULL,
    buffer_minutes INT NOT NULL DEFAULT 0,
    meeting_provider meeting_provider NOT NULL,
    meeting_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id, slug)
);

CREATE INDEX idx_interview_type_org_id ON interview_type(organization_id);

-- +goose Down
DROP TABLE interview_type;
DROP TYPE meeting_provider;
