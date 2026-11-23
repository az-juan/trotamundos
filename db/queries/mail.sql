-- name: ListMailboxes :many
SELECT id::text AS id, provider, email, coalesce(last_scan_at::text, '')::text AS last_scan_at
FROM spiritus.mailboxes ORDER BY created_at DESC;

-- name: AddMailbox :one
INSERT INTO spiritus.mailboxes (user_id, provider, email)
VALUES (sqlc.arg(user_id)::uuid, sqlc.arg(provider)::text, sqlc.arg(email)::text)
ON CONFLICT (user_id, email) DO UPDATE SET provider = EXCLUDED.provider
RETURNING id::text AS id, provider, email, coalesce(last_scan_at::text, '')::text AS last_scan_at;

-- name: ListSenders :many
SELECT id::text AS id, sender FROM spiritus.whitelist_senders ORDER BY sender;

-- name: AddSender :one
INSERT INTO spiritus.whitelist_senders (user_id, sender)
VALUES (sqlc.arg(user_id)::uuid, lower(sqlc.arg(sender)::text))
ON CONFLICT (user_id, sender) DO UPDATE SET sender = EXCLUDED.sender
RETURNING id::text AS id, sender;

-- name: RemoveSender :exec
DELETE FROM spiritus.whitelist_senders WHERE id = sqlc.arg(id)::uuid;

-- name: InsertAnalyticsEvent :exec
INSERT INTO spiritus.analytics_events (event_id, event_type, user_id, trip_id, booking_id, payload, occurred_at)
VALUES (sqlc.arg(event_id)::text, sqlc.arg(event_type)::text, NULLIF(sqlc.arg(user_id)::text,'')::uuid,
        NULLIF(sqlc.arg(trip_id)::text,'')::uuid, NULLIF(sqlc.arg(booking_id)::text,'')::uuid,
        sqlc.arg(payload)::jsonb, now())
ON CONFLICT (event_id) DO NOTHING;

-- name: AnalyticsSummary :one
SELECT count(*)::bigint AS total,
       count(*) FILTER (WHERE event_type = 'booking.confirmed')::bigint AS confirmations,
       count(*) FILTER (WHERE event_type = 'trip.disrupted')::bigint AS disruptions,
       coalesce(max(occurred_at::text), '')::text AS last_event_at
FROM spiritus.analytics_events;

-- name: ListAnalyticsEvents :many
SELECT event_id, event_type, source, occurred_at::text AS occurred_at, payload
FROM (
  SELECT event_id, event_type, 'spiritus' AS source, occurred_at, payload FROM spiritus.analytics_events
) e ORDER BY occurred_at DESC LIMIT sqlc.arg(max_items)::bigint;
