-- Esquema canónico para sqlc. Espejo de fase_0/deploy/postgres/init.sql (sin roles/grants).
CREATE SCHEMA IF NOT EXISTS pater;
CREATE SCHEMA IF NOT EXISTS filius;
CREATE SCHEMA IF NOT EXISTS spiritus;

CREATE TABLE filius.users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    display_name  TEXT NOT NULL,
    preferences   JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE filius.trips (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES filius.users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'upcoming',
    start_date  DATE,
    end_date    DATE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    archived_at TIMESTAMPTZ
);

CREATE TABLE filius.bookings (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id      UUID NOT NULL REFERENCES filius.trips(id) ON DELETE CASCADE,
    type         TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'discovered',
    provider     TEXT NOT NULL,
    provider_ref TEXT NOT NULL,
    lat          DOUBLE PRECISION,
    lon          DOUBLE PRECISION,
    details      JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (provider, provider_ref)
);

CREATE TABLE filius.travelers (
    id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id  UUID NOT NULL REFERENCES filius.trips(id) ON DELETE CASCADE,
    name     TEXT NOT NULL,
    document TEXT,
    email    TEXT
);

CREATE TABLE filius.itinerary_shares (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id     UUID NOT NULL REFERENCES filius.trips(id) ON DELETE CASCADE,
    token       TEXT NOT NULL UNIQUE,
    permission  TEXT NOT NULL DEFAULT 'view',
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE filius.assistance_cases (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES filius.users(id) ON DELETE CASCADE,
    trip_id     UUID REFERENCES filius.trips(id) ON DELETE SET NULL,
    booking_id  UUID REFERENCES filius.bookings(id) ON DELETE SET NULL,
    status      TEXT NOT NULL DEFAULT 'open',
    reason      TEXT NOT NULL,
    severity    TEXT NOT NULL,
    agency_id   TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ
);

CREATE TABLE spiritus.mailboxes (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL,
    provider     TEXT NOT NULL,
    email        TEXT NOT NULL,
    oauth_tokens JSONB,
    last_scan_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, email)
);

CREATE TABLE spiritus.whitelist_senders (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL,
    sender     TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, sender)
);

CREATE TABLE spiritus.analytics_events (
    event_id    TEXT PRIMARY KEY,
    event_type  TEXT NOT NULL,
    user_id     UUID,
    trip_id     UUID,
    booking_id  UUID,
    payload     JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE pater.rendered_templates (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template   TEXT NOT NULL,
    locale     TEXT NOT NULL,
    hash       TEXT NOT NULL,
    body       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (template, locale, hash)
);
