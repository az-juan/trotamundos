-- name: ListTrips :many
SELECT t.id::text AS id, t.name, t.status,
       coalesce(t.start_date::text, '')::text AS start_date,
       coalesce(t.end_date::text, '')::text AS end_date,
       t.created_at::text AS created_at
FROM filius.trips t
WHERE (sqlc.arg(status)::text = '' OR t.status = sqlc.arg(status)::text)
ORDER BY t.created_at DESC;

-- name: GetTrip :one
SELECT t.id::text AS id, t.name, t.status,
       coalesce(t.start_date::text, '')::text AS start_date,
       coalesce(t.end_date::text, '')::text AS end_date,
       t.created_at::text AS created_at
FROM filius.trips t WHERE t.id = sqlc.arg(id)::uuid;

-- name: CreateTrip :one
INSERT INTO filius.trips (user_id, name, start_date, end_date)
VALUES (sqlc.arg(user_id)::uuid, sqlc.arg(name)::text,
        nullif(sqlc.arg(start_date)::text, '')::date,
        nullif(sqlc.arg(end_date)::text, '')::date)
RETURNING id::text AS id, name, status,
          coalesce(start_date::text, '')::text AS start_date,
          coalesce(end_date::text, '')::text AS end_date,
          created_at::text AS created_at;

-- name: ArchiveTrip :exec
UPDATE filius.trips SET status = 'archived', archived_at = now()
WHERE id = sqlc.arg(id)::uuid;

-- name: ListBookingsForTrip :many
SELECT b.id::text AS id, b.trip_id::text AS trip_id, b.type, b.status,
       b.provider, b.provider_ref, b.created_at::text AS created_at
FROM filius.bookings b WHERE b.trip_id = sqlc.arg(trip_id)::uuid ORDER BY b.created_at;

-- name: AddBooking :one
INSERT INTO filius.bookings (trip_id, type, status, provider, provider_ref, details)
VALUES (sqlc.arg(trip_id)::uuid, sqlc.arg(kind)::text, 'pending',
        sqlc.arg(provider)::text, sqlc.arg(provider_ref)::text, '{}'::jsonb)
RETURNING id::text AS id, trip_id::text AS trip_id, type, status, provider, provider_ref,
          created_at::text AS created_at;

-- name: FindBookingByRef :one
SELECT b.id::text AS id, b.trip_id::text AS trip_id, b.type, b.status,
       b.provider, b.provider_ref, b.created_at::text AS created_at
FROM filius.bookings b WHERE b.provider_ref = sqlc.arg(provider_ref)::text;

-- name: UpdateBookingStatus :exec
UPDATE filius.bookings SET status = sqlc.arg(status)::text, updated_at = now()
WHERE provider_ref = sqlc.arg(provider_ref)::text;

-- name: UpdateBookingCoords :exec
UPDATE filius.bookings SET lat = sqlc.arg(lat)::float8, lon = sqlc.arg(lon)::float8, updated_at = now()
WHERE provider_ref = sqlc.arg(provider_ref)::text;

-- name: NearbyTrips :many
SELECT DISTINCT t.id::text AS id, t.name, t.status,
       coalesce(t.start_date::text, '')::text AS start_date,
       coalesce(t.end_date::text, '')::text AS end_date,
       t.created_at::text AS created_at
FROM filius.trips t JOIN filius.bookings b ON b.trip_id = t.id
WHERE (b.details->>'lat')::float8 BETWEEN sqlc.arg(lat)::float8 - sqlc.arg(radius_km)::float8 / 111.0
                                     AND sqlc.arg(lat)::float8 + sqlc.arg(radius_km)::float8 / 111.0
  AND (b.details->>'lon')::float8 BETWEEN sqlc.arg(lon)::float8 - sqlc.arg(radius_km)::float8 / 111.0
                                     AND sqlc.arg(lon)::float8 + sqlc.arg(radius_km)::float8 / 111.0
ORDER BY t.created_at DESC;

-- name: ListGeoPoints :many
SELECT t.id::text AS trip_id, t.name AS trip_name, b.provider,
       b.provider_ref, b.type,
       (b.details->>'lat')::float8 AS lat,
       (b.details->>'lon')::float8 AS lon
FROM filius.bookings b JOIN filius.trips t ON t.id = b.trip_id
WHERE b.details->>'lat' IS NOT NULL AND b.details->>'lon' IS NOT NULL
ORDER BY t.name, b.provider;
