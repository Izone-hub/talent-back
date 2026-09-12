#!/bin/sh
set -e

echo "=== iZone Talent Backend Startup ==="

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Load .env if present
if [ -f "$SCRIPT_DIR/.env" ]; then
    set -a
    . "$SCRIPT_DIR/.env"
    set +a
elif [ -f "$SCRIPT_DIR/../.env" ]; then
    set -a
    . "$SCRIPT_DIR/../.env"
    set +a
fi

# Determine database connection string
if [ -n "$DATABASE_URL" ]; then
    DB_URL="$DATABASE_URL"
else
    DB_HOST="${HOST_ADDRESS:-postgres}"
    DB_PORT="${HOST_PORT:-5432}"
    DB_USER="${HOST_USERNAME:-postgres}"
    DB_PASS="${HOST_PASSWORD:-postgres}"
    DB_NAME="${DATABASE:-izone_talent}"
    DB_SSL="${DB_SSLMODE:-require}"
    DB_URL="postgres://${DB_USER}:${DB_PASS}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSL}"
fi

# Ensure upload directory exists and has correct permissions
mkdir -p "$SCRIPT_DIR/uploads/cv"
if [ "$(id -u)" = "0" ] && id appuser >/dev/null 2>&1; then
    chown -R appuser:appuser "$SCRIPT_DIR/uploads"
fi

# Align docker group if docker socket is mounted and running as root
if [ "$(id -u)" = "0" ] && [ -S /var/run/docker.sock ] && id appuser >/dev/null 2>&1; then
    SOCK_GID=$(stat -c '%g' /var/run/docker.sock)
    if [ "$SOCK_GID" != "0" ]; then
        if ! getent group "$SOCK_GID" >/dev/null 2>&1; then
            groupadd -g "$SOCK_GID" hostdocker || true
        fi
        DOCKER_GRP=$(getent group "$SOCK_GID" | cut -d: -f1)
        usermod -aG "$DOCKER_GRP" appuser || true
    fi
fi

# Database wait-for-readiness loop (max 30 seconds)
echo "--- Waiting for database connection... ---"
DB_HOST_ONLY="${HOST_ADDRESS:-postgres}"
DB_PORT_ONLY="${HOST_PORT:-5432}"
RETRIES=30
until nc -z -w 2 "$DB_HOST_ONLY" "$DB_PORT_ONLY" 2>/dev/null || [ $RETRIES -eq 0 ]; do
    echo "Database at ${DB_HOST_ONLY}:${DB_PORT_ONLY} is unavailable - sleeping 1s (${RETRIES} left)"
    sleep 1
    RETRIES=$((RETRIES - 1))
done

# Run goose database migrations
GOOSE_BIN=""
if command -v goose >/dev/null 2>&1; then
    GOOSE_BIN="goose"
elif [ -x "$HOME/go/bin/goose" ]; then
    GOOSE_BIN="$HOME/go/bin/goose"
fi

if [ -n "$GOOSE_BIN" ] && [ -d "$SCRIPT_DIR/sql/schema" ]; then
    echo "--- Applying database migrations via Goose ---"
    "$GOOSE_BIN" -dir "$SCRIPT_DIR/sql/schema" postgres "$DB_URL" up || {
        echo "ERROR: Database migration failed!"
        exit 1
    }
    echo "--- Migrations applied successfully ---"
else
    echo "Notice: goose or migrations directory not found, skipping migration phase."
fi

echo "--- Starting backend server ---"
# Rebuild server if go is installed and server binary is missing or outdated
if command -v go >/dev/null 2>&1 && { [ ! -f "$SCRIPT_DIR/server" ] || [ "$SCRIPT_DIR/main.go" -nt "$SCRIPT_DIR/server" ]; }; then
    echo "--- Building backend binary ---"
    (cd "$SCRIPT_DIR" && go build -o "$SCRIPT_DIR/server" main.go)
fi

cd "$SCRIPT_DIR"
if [ "$(id -u)" = "0" ] && id appuser >/dev/null 2>&1 && command -v gosu >/dev/null 2>&1; then
    exec gosu appuser "$SCRIPT_DIR/server"
else
    exec "$SCRIPT_DIR/server"
fi
