-- +goose Up
-- +goose NO TRANSACTION
ALTER TYPE meeting_provider ADD VALUE IF NOT EXISTS 'on_site';

UPDATE interview_type
SET meeting_provider = 'on_site'
WHERE meeting_provider IN ('bale_link', 'custom_url');

CREATE TYPE meeting_provider_new AS ENUM ('google_meet', 'on_site');

ALTER TABLE interview_type
  ALTER COLUMN meeting_provider TYPE meeting_provider_new
  USING meeting_provider::text::meeting_provider_new;

DROP TYPE meeting_provider;

ALTER TYPE meeting_provider_new RENAME TO meeting_provider;

-- +goose Down
CREATE TYPE meeting_provider_old AS ENUM ('google_meet', 'bale_link', 'custom_url');

ALTER TABLE interview_type
  ALTER COLUMN meeting_provider TYPE meeting_provider_old
  USING (
    CASE meeting_provider::text
      WHEN 'on_site' THEN 'custom_url'
      ELSE meeting_provider::text
    END
  )::meeting_provider_old;

DROP TYPE meeting_provider;

ALTER TYPE meeting_provider_old RENAME TO meeting_provider;
