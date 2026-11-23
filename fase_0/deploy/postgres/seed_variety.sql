-- =============================================================================
-- Trotamundos — Datos de demostración variados (Fase 0)
-- Viajes por Argentina y el mundo, proveedores y personas ficticias.
-- Se monta como zz-seed_variety.sql para ejecutarse después de init.sql y
-- también se puede aplicar en caliente con ./scripts/seed.sh.
-- Idempotente: identificadores de reserva y eventos generados al azar.
-- =============================================================================

DO $$
DECLARE
  uid UUID := '00000000-0000-0000-0000-000000000001';

  destinos TEXT[] := ARRAY[
    'Buenos Aires → Bariloche', 'Córdoba → Mendoza', 'Salta → Jujuy',
    'Ushuaia → El Calafate', 'Posadas → Iguazú', 'Mar del Plata → Bahía Blanca',
    'Rosario → Santa Fe', 'Tucumán → La Rioja', 'Neuquén → Esquel',
    'Buenos Aires → Lisboa', 'Buenos Aires → Roma', 'Buenos Aires → Tokio',
    'Buenos Aires → Ciudad de México', 'Buenos Aires → Estambul',
    'Buenos Aires → El Cairo', 'Buenos Aires → Reikiavik',
    'Buenos Aires → Nueva York', 'Buenos Aires → Bangkok',
    'Buenos Aires → Cusco', 'Buenos Aires → La Habana'
  ];
  lats DOUBLE PRECISION[] := ARRAY[
    -41.13, -31.42, -24.79, -54.80, -25.60, -38.00,
    -32.95, -26.80, -38.95, 38.72, 41.90, 35.68,
     19.43,  41.01,  30.04,  64.14,  40.71,  13.75,
    -13.53,  23.11
  ];
  lons DOUBLE PRECISION[] := ARRAY[
    -71.31, -64.19, -65.41, -68.30, -54.57, -57.55,
    -60.65, -65.21, -71.14,  -9.14,  12.50, 139.69,
    -99.13,  28.98,  31.24, -21.94, -74.01, 100.50,
    -71.97, -82.37
  ];

  proveedores TEXT[] := ARRAY[
    'Aerolíneas del Plata', 'Líneas Aéreas Cóndor', 'Vapores del Sur',
    'Automóviles Pampa', 'Ferrocarril del Este', 'Hostal Los Álamos',
    'Posada del Ceibo', 'Expresos Andinos', 'Naviera Estrella Polar',
    'Gran Hotel del Centro', 'Aurora Viajes', 'Casa del Olvido'
  ];
  personas TEXT[] := ARRAY[
    'Ana Ruiz', 'Bruno Ferrari', 'Camila Núñez', 'Diego Salvatierra',
    'Elena Vázquez', 'Facundo Ríos', 'Gabriela Ponce', 'Hugo Villalba',
    'Irene Castillo', 'Julián Peralta', 'Karina Sosa', 'Lucas Bertone',
    'Malena Aguirre', 'Nicolás Duarte', 'Oriana Fuentes', 'Pedro Solís',
    'Quintín Ledesma', 'Rosa Ibarra', 'Santiago Molina', 'Valeria Cáceres'
  ];
  dominios TEXT[] := ARRAY[
    'aerolineasdelplata.com', 'lineascondor.com.ar', 'vaporesdelsur.com',
    'expresosandinos.com', 'hostal-losalamos.com', 'posadadelceibo.com.ar',
    'naviera-estrella.com', 'granhoteldelcentro.com'
  ];
  tipos TEXT[] := ARRAY['transport', 'lodging', 'car_rental'];
  estados TEXT[] := ARRAY['confirmed', 'pending', 'discovered', 'disrupted'];
  orden INT[];

  i INT; k INT; j INT; v_n INT; v_trip UUID; v_start DATE;
  v_provider TEXT; v_type TEXT; v_ref TEXT;
  v_lat DOUBLE PRECISION; v_lon DOUBLE PRECISION; v_persona TEXT;
BEGIN
  -- Permutación aleatoria: 20 viajes, 20 destinos distintos.
  SELECT ARRAY(SELECT g FROM generate_series(1, array_length(destinos, 1)) AS g
               ORDER BY random()) INTO orden;

  FOR i IN 1..20 LOOP
    j := orden[i];
    v_lat := lats[j] + (random() - 0.5) * 0.6;
    v_lon := lons[j] + (random() - 0.5) * 0.6;
    v_start := CURRENT_DATE + floor(random() * 150)::INT;

    INSERT INTO filius.trips (user_id, name, status, start_date, end_date, created_at)
    VALUES (
      uid, destinos[j],
      (ARRAY['upcoming', 'upcoming', 'active', 'archived'])[floor(random() * 4)::INT + 1],
      v_start,
      v_start + (3 + floor(random() * 12))::INT,
      now() - floor(random() * 60)::INT * interval '1 hour'
    )
    RETURNING id INTO v_trip;

    v_n := 1 + floor(random() * 3)::INT;
    FOR k IN 1..v_n LOOP
      v_provider := proveedores[floor(random() * array_length(proveedores, 1))::INT + 1];
      v_type := tipos[floor(random() * array_length(tipos, 1))::INT + 1];
      v_ref := CASE v_type
                 WHEN 'transport' THEN 'TR'
                 WHEN 'lodging' THEN 'HT'
                 ELSE 'AU'
               END || '-' || upper(substr(md5(random()::text), 1, 8));

      INSERT INTO filius.bookings (trip_id, type, status, provider, provider_ref, lat, lon, details)
      VALUES (
        v_trip, v_type,
        estados[floor(random() * array_length(estados, 1))::INT + 1],
        v_provider, v_ref, v_lat, v_lon,
        jsonb_build_object('lat', v_lat, 'lon', v_lon, 'destination', destinos[j])
      )
      ON CONFLICT (provider, provider_ref) DO NOTHING;

      INSERT INTO spiritus.analytics_events (event_id, event_type, user_id, trip_id, payload, occurred_at)
      VALUES (
        md5(random()::text || clock_timestamp()::text),
        (ARRAY['booking.confirmed', 'booking.discovered', 'trip.disrupted'])[floor(random() * 3)::INT + 1],
        uid, v_trip,
        jsonb_build_object('provider', v_provider, 'provider_ref', v_ref, 'type', v_type),
        now() - floor(random() * 30)::INT * interval '1 day'
      )
      ON CONFLICT (event_id) DO NOTHING;
    END LOOP;

    v_n := 1 + floor(random() * 2)::INT;
    FOR k IN 1..v_n LOOP
      v_persona := personas[floor(random() * array_length(personas, 1))::INT + 1];
      INSERT INTO filius.travelers (trip_id, name, document, email)
      VALUES (
        v_trip, v_persona,
        (10000000 + floor(random() * 89999999)::INT)::TEXT || 'X',
        lower(translate(replace(v_persona, ' ', '.'), 'áéíóúñÁÉÍÓÚÑ', 'aeiounAEIOUN')) || '@correo-ficticio.test'
      );
    END LOOP;
  END LOOP;

  INSERT INTO spiritus.whitelist_senders (user_id, sender)
  SELECT uid, '@' || d FROM unnest(dominios) AS d
  ON CONFLICT (user_id, sender) DO NOTHING;

  INSERT INTO spiritus.mailboxes (user_id, provider, email)
  SELECT uid, CASE WHEN random() < 0.5 THEN 'gmail' ELSE 'imap' END,
         lower(translate(replace(p, ' ', '.'), 'áéíóúñÁÉÍÓÚÑ', 'aeiounAEIOUN')) || '@correo-ficticio.test'
  FROM unnest(personas[1:6]) AS p
  ON CONFLICT (user_id, email) DO NOTHING;
END $$;
