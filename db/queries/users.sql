-- name: VerifyUser :one
SELECT id::text AS id, email, display_name FROM filius.users
WHERE email = sqlc.arg(email)::text AND password_hash = crypt(sqlc.arg(password)::text, password_hash);

-- name: GetPreferences :one
SELECT preferences FROM filius.users WHERE id = sqlc.arg(id)::uuid;

-- name: UpdatePreferences :exec
UPDATE filius.users SET preferences = sqlc.arg(preferences)::jsonb, updated_at = now()
WHERE id = sqlc.arg(id)::uuid;

-- name: ListTravelers :many
SELECT id::text AS id, trip_id::text AS trip_id, name, coalesce(document, '') AS document, coalesce(email, '') AS email
FROM filius.travelers WHERE trip_id = sqlc.arg(trip_id)::uuid;

-- name: CreateShare :one
INSERT INTO filius.itinerary_shares (trip_id, token, permission, expires_at)
VALUES (sqlc.arg(trip_id)::uuid, sqlc.arg(token)::text, 'view', now() + interval '7 days')
RETURNING token, trip_id::text AS trip_id, permission, expires_at::text AS expires_at;

-- name: GetShare :one
SELECT s.trip_id::text AS trip_id, s.expires_at::text AS expires_at
FROM filius.itinerary_shares s WHERE s.token = sqlc.arg(token)::text AND s.permission = 'view';
