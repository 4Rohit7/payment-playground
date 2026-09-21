#!/bin/sh
set -e

# 1. Initialise Postgres data directory if first run
if [ ! -f "$PGDATA/PG_VERSION" ]; then
  echo "[startup] Initialising Postgres..."
  chown -R postgres:postgres "$PGDATA" 2>/dev/null || true
  su-exec postgres initdb \
    --username="$POSTGRES_USER" \
    --pwfile=<(echo "$POSTGRES_PASSWORD") \
    -D "$PGDATA" \
    --auth-host=md5 \
    --auth-local=trust \
    -E UTF8 --locale=C > /dev/null
fi

# 2. Start Postgres
echo "[startup] Starting Postgres..."
su-exec postgres pg_ctl -D "$PGDATA" -o "-p 5432" -w start > /dev/null

# 3. Create database if it doesn't exist
su-exec postgres psql -U "$POSTGRES_USER" -tc \
  "SELECT 1 FROM pg_database WHERE datname='$POSTGRES_DB'" \
  | grep -q 1 || \
  su-exec postgres createdb -U "$POSTGRES_USER" "$POSTGRES_DB"

# 4. Start the Go server (runs DB migrations automatically)
echo "[startup] Starting payment-playground on :${PORT}..."
exec /app/server