-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE organization (
    id UUID PRIMARY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL
    slug TEXT NOT NULL UNIQUE
    log_url TEXT
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE
    plan NOT NULL DEFAULT 'free'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)

CREATE TYPE plan_type AS ENUM (
    'free',
    'pro',
    'business'
);

CREATE INDEX idx_org_slug ON organization(slug);
CREATE INDEX idx_org_name ON organization(name);

-- +goose Down
DROP TABLE organization;
DROP TYPE plan_type;
DROP EXTENSION IF EXISTS pgcrypto;