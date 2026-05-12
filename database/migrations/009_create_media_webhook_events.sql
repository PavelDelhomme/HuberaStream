-- Événements reçus depuis Radarr / Sonarr (webhooks)
CREATE TABLE IF NOT EXISTS media_webhook_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source VARCHAR(32) NOT NULL,
    event_type VARCHAR(128) NOT NULL DEFAULT '',
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_media_webhook_events_source ON media_webhook_events (source);
CREATE INDEX IF NOT EXISTS idx_media_webhook_events_created_at ON media_webhook_events (created_at DESC);
