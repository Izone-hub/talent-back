#!/bin/sh
set -e

echo "--- Running database migrations ---"

# Construct database URL from env vars
DB_URL="postgres://${HOST_USERNAME:-postgres}:${HOST_PASSWORD:-postgres}@${HOST_ADDRESS:-postgres}:${HOST_PORT:-5432}/${DATABASE:-izone_talent}?sslmode=disable"

# Run goose migrations
goose -dir /app/sql/schema postgres "$DB_URL" up

echo "--- Migrations complete, starting server ---"

# Start the Go server
exec /app/server
