-- =============================================================================
-- Trotamundos — Inicialización de PostgreSQL 16 (Fase 0 · Arquitectura Trinidad)
-- Deriva de: ADR-104 (segregación por esquema), ADR-106 (usuarios y preferencias),
--            ADR-107 (adaptador analítico Fase 0), ADR-108 (RBAC por esquema)
--            y ADR-109/110 (Asistente Virtual, Fase 1 — tabla preparada).
-- =============================================================================

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- -----------------------------------------------------------------------------
-- 1. Esquemas por unidad (ADR-104)
-- -----------------------------------------------------------------------------
CREATE SCHEMA IF NOT EXISTS pater;
CREATE SCHEMA IF NOT EXISTS filius;
CREATE SCHEMA IF NOT EXISTS spiritus;

-- -----------------------------------------------------------------------------
-- 2. Roles RBAC con mínimo privilegio desde Fase 0 (ADR-108, mitiga DT-01)
-- -----------------------------------------------------------------------------
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'pater_app') THEN
        CREATE ROLE pater_app LOGIN PASSWORD 'pater_dev';
    END IF;
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'filius_app') THEN
        CREATE ROLE filius_app LOGIN PASSWORD 'filius_dev';
    END IF;
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'spiritus_app') THEN
        CREATE ROLE spiritus_app LOGIN PASSWORD 'spiritus_dev';
    END IF;
END
$$;

REVOKE ALL ON SCHEMA public, pater, filius, spiritus FROM PUBLIC;
GRANT USAGE ON SCHEMA public TO pater_app, filius_app, spiritus_app; -- pgcrypto.crypt (ADR-111)
GRANT USAGE, CREATE ON SCHEMA pater TO pater_app;
GRANT USAGE, CREATE ON SCHEMA filius TO filius_app;
GRANT USAGE, CREATE ON SCHEMA spiritus TO spiritus_app;

-- -----------------------------------------------------------------------------
-- 3. Esquema FILIUS — Dominio operacional transaccional (OLTP, ADR-104 / ADR-106)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS filius.users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    display_name  TEXT NOT NULL,
    preferences   JSONB NOT NULL DEFAULT '{"locale":"es","currency":"USD","gps_radius_km":50}'::jsonb,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_users_preferences_gin ON filius.users USING GIN (preferences);

CREATE TABLE IF NOT EXISTS filius.trips (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES filius.users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'upcoming'
                CHECK (status IN ('upcoming','active','archived')),
    start_date  DATE,
    end_date    DATE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    archived_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_trips_user_status ON filius.trips (user_id, status);

CREATE TABLE IF NOT EXISTS filius.bookings (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id      UUID NOT NULL REFERENCES filius.trips(id) ON DELETE CASCADE,
    type         TEXT NOT NULL CHECK (type IN ('lodging','transport','car_rental')),
    status       TEXT NOT NULL DEFAULT 'discovered'
                 CHECK (status IN ('discovered','pending','confirmed','cancelled','disrupted','archived')),
    provider     TEXT NOT NULL,
    provider_ref TEXT NOT NULL,
    lat          DOUBLE PRECISION,
    lon          DOUBLE PRECISION,
    details      JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (provider, provider_ref)
);
CREATE INDEX IF NOT EXISTS idx_bookings_trip ON filius.bookings (trip_id);
CREATE INDEX IF NOT EXISTS idx_bookings_status ON filius.bookings (status);
CREATE INDEX IF NOT EXISTS idx_bookings_coords ON filius.bookings (lat, lon) WHERE lat IS NOT NULL AND lon IS NOT NULL;

CREATE TABLE IF NOT EXISTS filius.travelers (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id   UUID NOT NULL REFERENCES filius.trips(id) ON DELETE CASCADE,
    name      TEXT NOT NULL,
    document  TEXT,
    email     TEXT
);
CREATE INDEX IF NOT EXISTS idx_travelers_trip ON filius.travelers (trip_id);

CREATE TABLE IF NOT EXISTS filius.itinerary_shares (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id     UUID NOT NULL REFERENCES filius.trips(id) ON DELETE CASCADE,
    token       TEXT NOT NULL UNIQUE,
    permission  TEXT NOT NULL DEFAULT 'view' CHECK (permission IN ('view')),
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);


-- -----------------------------------------------------------------------------
-- 3.1 Asistente Virtual (Fase 1, ADR-109/110) — tabla preparada, feature flag desactivado en Fase 0
--     El context_token y la sesion de chat viven SOLO en Redis (TTL), nunca aqui.
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS filius.assistance_cases (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES filius.users(id) ON DELETE CASCADE,
    trip_id      UUID REFERENCES filius.trips(id) ON DELETE SET NULL,
    booking_id   UUID REFERENCES filius.bookings(id) ON DELETE SET NULL,
    status       TEXT NOT NULL DEFAULT 'open'
                 CHECK (status IN ('open','handed_off','resolved','closed','expired')),
    reason       TEXT NOT NULL CHECK (reason IN ('complex_request','reschedule_failed','complaint','other')),
    severity     TEXT NOT NULL CHECK (severity IN ('low','medium','high')),
    agency_id    TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at  TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_assistance_cases_user_status ON filius.assistance_cases (user_id, status);

-- -----------------------------------------------------------------------------
-- 4. Esquema SPIRITUS — Email Ingest + Adaptador Analítico Fase 0 (ADR-107)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS spiritus.mailboxes (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL,
    provider      TEXT NOT NULL CHECK (provider IN ('gmail','imap')),
    email         TEXT NOT NULL,
    oauth_tokens  JSONB,
    last_scan_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, email)
);

CREATE TABLE IF NOT EXISTS spiritus.ingested_emails (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mailbox_id   UUID NOT NULL REFERENCES spiritus.mailboxes(id) ON DELETE CASCADE,
    sha256       TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'received'
                 CHECK (status IN ('received','parsed','discovered','no_match','failed')),
    received_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    parsed_at    TIMESTAMPTZ,
    UNIQUE (mailbox_id, sha256)
);
CREATE INDEX IF NOT EXISTS idx_ingested_emails_mailbox ON spiritus.ingested_emails (mailbox_id, received_at DESC);

CREATE TABLE IF NOT EXISTS spiritus.whitelist_senders (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL,
    sender     TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, sender)
);

CREATE TABLE IF NOT EXISTS spiritus.analytics_events (
    event_id       TEXT PRIMARY KEY,
    event_type     TEXT NOT NULL,
    user_id        UUID,
    trip_id        UUID,
    booking_id     UUID,
    provider       TEXT,
    booking_type   TEXT,
    destination    TEXT,
    payload        JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at    TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_analytics_events_type_date ON spiritus.analytics_events (event_type, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_analytics_events_user ON spiritus.analytics_events (user_id, occurred_at DESC);

CREATE TABLE IF NOT EXISTS spiritus.insight_aggregates (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dimension   TEXT NOT NULL,
    key         TEXT NOT NULL,
    period      TEXT NOT NULL CHECK (period IN ('7d','30d','90d')),
    metric_val  DOUBLE PRECISION NOT NULL DEFAULT 0,
    computed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (dimension, key, period)
);

-- -----------------------------------------------------------------------------
-- 5. Esquema PATER — Facade (caché referencial de plantillas SSR)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS pater.rendered_templates (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template    TEXT NOT NULL,
    locale      TEXT NOT NULL,
    hash        TEXT NOT NULL,
    body        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (template, locale, hash)
);

-- -----------------------------------------------------------------------------
-- 6. Asignación estricta de privilegios por esquema (ADR-108)
-- -----------------------------------------------------------------------------
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA pater TO pater_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA filius TO filius_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA spiritus TO spiritus_app;

-- ADR-111: el prototipo web usa sqlc desde Pater sobre el esquema compartido.
GRANT USAGE ON SCHEMA filius, spiritus TO pater_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA filius TO pater_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA spiritus TO pater_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA pater TO filius_app;

ALTER DEFAULT PRIVILEGES IN SCHEMA pater GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO pater_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA filius GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO filius_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA spiritus GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO spiritus_app;


-- -----------------------------------------------------------------------------
-- 7. Seed de desarrollo local (usuario dev@trotamundos.app)
-- -----------------------------------------------------------------------------
INSERT INTO filius.users (id, email, password_hash, display_name, preferences)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'dev@trotamundos.app',
    crypt('dev', gen_salt('bf')),
    'Viajero Dev',
    '{"locale":"es","currency":"USD","gps_radius_km":50}'::jsonb
)
ON CONFLICT DO NOTHING;

-- -----------------------------------------------------------------------------
-- 8. Datos de demostración del prototipo HTMX/sqlc (ADR-111)
-- -----------------------------------------------------------------------------
INSERT INTO filius.trips (id, user_id, name, status, start_date, end_date)
VALUES
  ('11111111-1111-1111-1111-111111111111', '00000000-0000-0000-0000-000000000001', 'Buenos Aires → Madrid', 'upcoming', CURRENT_DATE + 12, CURRENT_DATE + 19),
  ('22222222-2222-2222-2222-222222222222', '00000000-0000-0000-0000-000000000001', 'Escapada a Montevideo', 'upcoming', NULL, NULL)
ON CONFLICT (id) DO NOTHING;

INSERT INTO filius.bookings (trip_id, type, status, provider, provider_ref, details)
VALUES
  ('11111111-1111-1111-1111-111111111111', 'transport', 'confirmed', 'Aerolíneas del Plata', 'PNR-1930', '{"lat":-34.5901,"lon":-58.3748,"destination":"MAD"}'),
  ('11111111-1111-1111-1111-111111111111', 'lodging', 'confirmed', 'Hotel Presidente', 'HT-4471', '{"lat":-34.6074,"lon":-58.3720}'),
  ('22222222-2222-2222-2222-222222222222', 'transport', 'confirmed', 'Ferrocarril del Este', 'TR-2210', '{"lat":-34.6285,"lon":-58.3683}')
ON CONFLICT (provider, provider_ref) DO NOTHING;

INSERT INTO filius.travelers (trip_id, name, document, email)
VALUES ('11111111-1111-1111-1111-111111111111', 'Ana Ruiz', '12345678Z', 'ana@example.com');

INSERT INTO spiritus.whitelist_senders (user_id, sender)
VALUES ('00000000-0000-0000-0000-000000000001', '@aerolineasdelplata.com'),
       ('00000000-0000-0000-0000-000000000001', '@hotelpresidente.com')
ON CONFLICT (user_id, sender) DO NOTHING;

INSERT INTO spiritus.mailboxes (user_id, provider, email)
VALUES ('00000000-0000-0000-0000-000000000001', 'gmail', 'dev@trotamundos.app')
ON CONFLICT (user_id, email) DO NOTHING;
