#!/bin/bash
# Sandbox seed script for talent-backend
# Usage: ./scripts/seed.sh

set -e

DB_URL="${DATABASE_URL:-postgres://postgres:postgres@localhost:5432/izone_talent?sslmode=disable}"

echo "Seeding database at: $DB_URL"
echo "Running migrations..."
go run github.com/pressly/goose/v3/cmd/goose -dir sql/schema postgres "$DB_URL" up

echo "Loading seed data..."
psql "$DB_URL" -f scripts/seed.sql

echo "Done! Database seeded successfully."
