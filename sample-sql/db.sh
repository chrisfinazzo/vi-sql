#!/usr/bin/env bash

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Directory containing seed CSVs (go run ./sample-sql/seed generates these).
SEED_DIR="${SEED_DIR:-/tmp/vi-sql-seed}"

# Postgres config
PG_CONTAINER="tui-postgres"
PG_USER="postgres"
PG_PASSWORD="postgres"
PG_DB="tui_sample_db"
PG_PORT="5432"
PG_SQL="$SCRIPT_DIR/sample.postgres.sql"

# Postgres SSL config (separate container, port 5433)
PG_SSL_CONTAINER="tui-postgres-ssl"
PG_SSL_PORT="5433"
PG_SSL_VOLUME="tui-pg-ssl-certs"

# MySQL config
MY_CONTAINER="tui-mysql"
MY_USER="root"
MY_PASSWORD="mysql"
MY_DB="tui_sample_db"
MY_PORT="3306"
MY_SQL="$SCRIPT_DIR/sample.mysql.sql"

# MariaDB config
MARIA_CONTAINER="tui-mariadb"
MARIA_USER="root"
MARIA_PASSWORD="mariadb"
MARIA_DB="tui_sample_db"
MARIA_PORT="3307"
MARIA_SQL="$SCRIPT_DIR/sample.mariadb.sql"

# CockroachDB config
CRDB_CONTAINER="tui-cockroachdb"
CRDB_USER="root"
CRDB_DB="tui_sample_db"
CRDB_PORT="26257"
CRDB_HTTP_PORT="8080"
CRDB_SQL="$SCRIPT_DIR/sample.cockroach.sql"

# SQLite config
SQLITE_DB="$SCRIPT_DIR/sample.db"
SQLITE_SQL="$SCRIPT_DIR/sample.sqlite.sql"

usage() {
  echo "Usage: $0 <command>"
  echo ""
  echo "Commands:"
  echo "  postgres up         Start PostgreSQL container and load sample data"
  echo "  postgres up-ssl     Start PostgreSQL container with SSL (port $PG_SSL_PORT) and load sample data"
  echo "  postgres stop       Stop PostgreSQL container"
  echo "  postgres stop-ssl   Stop PostgreSQL SSL container"
  echo "  postgres rm         Remove PostgreSQL container"
  echo "  postgres rm-ssl     Remove PostgreSQL SSL container and cert volume"
  echo "  postgres url        Print PostgreSQL connection strings"
  echo "  mysql up            Start MySQL container and load sample data"
  echo "  mysql stop          Stop MySQL container"
  echo "  mysql rm            Remove MySQL container"
  echo "  mysql url           Print MySQL connection strings"
  echo "  mariadb up          Start MariaDB container and load sample data"
  echo "  mariadb stop        Stop MariaDB container"
  echo "  mariadb rm          Remove MariaDB container"
  echo "  mariadb url         Print MariaDB connection strings"
  echo "  cockroachdb up      Start CockroachDB container and load sample data"
  echo "  cockroachdb stop    Stop CockroachDB container"
  echo "  cockroachdb rm      Remove CockroachDB container"
  echo "  cockroachdb url     Print CockroachDB connection strings"
  echo "  sqlite up           Create SQLite database and load sample data"
  echo "  sqlite rm           Remove SQLite database file"
  echo "  sqlite url          Print SQLite connection string"
  echo "  stop-all            Stop all containers"
  echo "  rm-all              Remove all containers"
  echo "  clean               Stop and remove all containers"
  exit 1
}

# --- Generic helpers ---

_prune() {
  docker rm -f -v "$1" 2>/dev/null || true
}

_stop() {
  local container=$1 label=$2
  echo "Stopping $label..."
  docker stop "$container" 2>/dev/null || echo "Container not running"
}

_rm() {
  local container=$1 label=$2
  echo "Removing $label..."
  docker rm -f -v "$container" 2>/dev/null || echo "Container not found"
}

# Generate CSVs if they don't exist yet.
_ensure_seed_data() {
  if [ ! -d "$SEED_DIR" ] || [ -z "$(ls "$SEED_DIR"/*.csv 2>/dev/null)" ]; then
    echo "Seed data not found in $SEED_DIR — generating..."
    go run "$SCRIPT_DIR/seed"
  fi
}

# Copy seed CSVs into a running container at /data/.
_copy_seed_data() {
  local container=$1
  _ensure_seed_data
  echo "Copying seed data from $SEED_DIR..."
  docker exec "$container" mkdir -p /data
  docker cp "${SEED_DIR}/." "$container:/data"
}

# Shared SQL loader for MySQL-compatible CLIs (mysql / mariadb).
# Passes --local-infile=1 so LOAD DATA LOCAL INFILE works.
_load_mysql_compat() {
  local container=$1 sql_file=$2 cli=$3 user=$4 password=$5 db=$6
  echo "Copying SQL file..."
  docker cp "$sql_file" "$container:/sample.sql"
  echo "Executing SQL..."
  docker exec -i "$container" "$cli" --local-infile=1 -u "$user" -p"$password" "$db" -e "source /sample.sql"
  echo "Verifying tables..."
  docker exec -it "$container" "$cli" -u "$user" -p"$password" "$db" -e "SHOW TABLES;"
}

_pg_load_data() {
  local container=$1
  echo "Copying SQL file..."
  docker cp "$PG_SQL" "$container:/sample.sql"
  echo "Executing SQL..."
  docker exec -i "$container" psql -U "$PG_USER" -d "$PG_DB" -f /sample.sql
}

# --- PostgreSQL ---

postgres_up() {
  [ -f "$PG_SQL" ] || { echo "Error: $PG_SQL not found"; exit 1; }

  echo "Removing old container (if exists)..."
  _prune $PG_CONTAINER

  echo "Starting PostgreSQL..."
  docker run -d \
    --name $PG_CONTAINER \
    -e POSTGRES_USER=$PG_USER \
    -e POSTGRES_PASSWORD=$PG_PASSWORD \
    -e POSTGRES_DB=$PG_DB \
    -p $PG_PORT:5432 \
    postgres:16

  echo "Waiting for Postgres..."
  until docker exec $PG_CONTAINER pg_isready -U $PG_USER > /dev/null 2>&1; do
    sleep 2
  done

  _copy_seed_data $PG_CONTAINER
  _pg_load_data $PG_CONTAINER

  echo
  echo "Postgres is ready on port $PG_PORT"
  postgres_url
}

postgres_up_ssl() {
  [ -f "$PG_SQL" ] || { echo "Error: $PG_SQL not found"; exit 1; }

  echo "Removing old SSL container and cert volume (if exist)..."
  _prune $PG_SSL_CONTAINER
  docker volume rm $PG_SSL_VOLUME 2>/dev/null || true

  # Generate certs inside docker so they are owned by UID 999 (postgres user).
  # PostgreSQL refuses a key file readable by other users, so ownership matters.
  echo "Generating SSL certificates..."
  docker run --rm \
    -v $PG_SSL_VOLUME:/certs \
    --entrypoint bash \
    postgres:16 -c "
      openssl req -new -x509 -days 365 -nodes \
        -out /certs/server.crt -keyout /certs/server.key \
        -subj '/CN=localhost' 2>/dev/null
      chown 999:999 /certs/server.crt /certs/server.key
      chmod 600 /certs/server.key
    "

  echo "Starting PostgreSQL with SSL..."
  docker run -d \
    --name $PG_SSL_CONTAINER \
    -e POSTGRES_USER=$PG_USER \
    -e POSTGRES_PASSWORD=$PG_PASSWORD \
    -e POSTGRES_DB=$PG_DB \
    -p $PG_SSL_PORT:5432 \
    -v $PG_SSL_VOLUME:/var/lib/postgresql/certs:ro \
    postgres:16 \
    -c ssl=on \
    -c ssl_cert_file=/var/lib/postgresql/certs/server.crt \
    -c ssl_key_file=/var/lib/postgresql/certs/server.key

  echo "Waiting for Postgres (SSL)..."
  until docker exec $PG_SSL_CONTAINER pg_isready -U $PG_USER > /dev/null 2>&1; do
    sleep 2
  done

  _copy_seed_data $PG_SSL_CONTAINER
  _pg_load_data $PG_SSL_CONTAINER

  echo
  echo "PostgreSQL SSL is ready on port $PG_SSL_PORT"
  postgres_url
}

postgres_stop()     { _stop $PG_CONTAINER     "PostgreSQL"; }
postgres_stop_ssl() { _stop $PG_SSL_CONTAINER "PostgreSQL SSL"; }
postgres_rm()       { _rm   $PG_CONTAINER     "PostgreSQL container"; }

postgres_rm_ssl() {
  _rm $PG_SSL_CONTAINER "PostgreSQL SSL container"
  docker volume rm $PG_SSL_VOLUME 2>/dev/null || echo "Volume not found"
}

postgres_url() {
  echo "Plain:       postgres://$PG_USER:$PG_PASSWORD@localhost:$PG_PORT/$PG_DB?sslmode=disable"
  echo "SSL require: postgres://$PG_USER:$PG_PASSWORD@localhost:$PG_SSL_PORT/$PG_DB?sslmode=require"
  echo "SSL verify:  postgres://$PG_USER:$PG_PASSWORD@localhost:$PG_SSL_PORT/$PG_DB?sslmode=verify-ca"
}

# --- MySQL ---

mysql_up() {
  [ -f "$MY_SQL" ] || { echo "Error: $MY_SQL not found"; exit 1; }

  echo "Removing old container (if exists)..."
  _prune $MY_CONTAINER

  # MySQL 8 auto-generates SSL certs on first start, so tls=skip-verify works
  # against this container without any extra configuration.
  echo "Starting MySQL..."
  docker run -d \
    --name $MY_CONTAINER \
    -e MYSQL_ROOT_PASSWORD=$MY_PASSWORD \
    -e MYSQL_DATABASE=$MY_DB \
    -p $MY_PORT:3306 \
    mysql:8

  echo "Waiting for MySQL..."
  until docker exec $MY_CONTAINER mysqladmin ping -u $MY_USER -p$MY_PASSWORD --silent 2>/dev/null; do
    sleep 2
  done

  _copy_seed_data $MY_CONTAINER
  _load_mysql_compat $MY_CONTAINER $MY_SQL mysql $MY_USER $MY_PASSWORD $MY_DB

  echo
  echo "MySQL is ready on port $MY_PORT"
  mysql_url
}

mysql_stop() { _stop $MY_CONTAINER "MySQL"; }
mysql_rm()   { _rm   $MY_CONTAINER "MySQL container"; }

mysql_url() {
  echo "Plain:       mysql://$MY_USER:$MY_PASSWORD@localhost:$MY_PORT/$MY_DB"
  echo "TLS skip:    mysql://$MY_USER:$MY_PASSWORD@localhost:$MY_PORT/$MY_DB?tls=skip-verify"
  echo "TLS prefer:  mysql://$MY_USER:$MY_PASSWORD@localhost:$MY_PORT/$MY_DB?tls=preferred"
  echo "TLS require: mysql://$MY_USER:$MY_PASSWORD@localhost:$MY_PORT/$MY_DB?tls=true"
}

# --- MariaDB ---

mariadb_up() {
  [ -f "$MARIA_SQL" ] || { echo "Error: $MARIA_SQL not found"; exit 1; }

  echo "Removing old container (if exists)..."
  _prune $MARIA_CONTAINER

  echo "Starting MariaDB..."
  docker run -d \
    --name $MARIA_CONTAINER \
    -e MARIADB_ROOT_PASSWORD=$MARIA_PASSWORD \
    -e MARIADB_DATABASE=$MARIA_DB \
    -p $MARIA_PORT:3306 \
    mariadb:11

  echo "Waiting for MariaDB..."
  until docker exec $MARIA_CONTAINER mariadb-admin ping -u $MARIA_USER -p$MARIA_PASSWORD --silent 2>/dev/null; do
    sleep 2
  done

  _copy_seed_data $MARIA_CONTAINER
  _load_mysql_compat $MARIA_CONTAINER $MARIA_SQL mariadb $MARIA_USER $MARIA_PASSWORD $MARIA_DB

  echo
  echo "MariaDB is ready on port $MARIA_PORT"
  mariadb_url
}

mariadb_stop() { _stop $MARIA_CONTAINER "MariaDB"; }
mariadb_rm()   { _rm   $MARIA_CONTAINER "MariaDB container"; }

mariadb_url() {
  echo "Plain:       mariadb://$MARIA_USER:$MARIA_PASSWORD@localhost:$MARIA_PORT/$MARIA_DB"
  echo "TLS skip:    mariadb://$MARIA_USER:$MARIA_PASSWORD@localhost:$MARIA_PORT/$MARIA_DB?tls=skip-verify"
  echo "TLS prefer:  mariadb://$MARIA_USER:$MARIA_PASSWORD@localhost:$MARIA_PORT/$MARIA_DB?tls=preferred"
}

# --- CockroachDB ---

cockroachdb_up() {
  [ -f "$CRDB_SQL" ] || { echo "Error: $CRDB_SQL not found"; exit 1; }

  echo "Removing old container (if exists)..."
  _prune $CRDB_CONTAINER

  echo "Starting CockroachDB..."
  docker run -d \
    --name $CRDB_CONTAINER \
    -p $CRDB_PORT:26257 \
    -p $CRDB_HTTP_PORT:8080 \
    cockroachdb/cockroach:latest-v24.3 \
    start-single-node --insecure

  echo "Waiting for CockroachDB..."
  until docker exec $CRDB_CONTAINER cockroach sql --insecure --execute "SELECT 1" > /dev/null 2>&1; do
    sleep 2
  done

  echo "Creating database..."
  docker exec $CRDB_CONTAINER cockroach sql --insecure \
    --execute "CREATE DATABASE IF NOT EXISTS $CRDB_DB;"

  _copy_seed_data $CRDB_CONTAINER

  echo "Copying SQL file..."
  docker cp $CRDB_SQL $CRDB_CONTAINER:/sample.sql

  echo "Executing SQL..."
  docker exec $CRDB_CONTAINER bash -c \
    "cockroach sql --insecure --database=$CRDB_DB --set=errexit=true < /sample.sql"

  echo "Verifying tables..."
  docker exec $CRDB_CONTAINER cockroach sql --insecure --database=$CRDB_DB \
    --execute "SHOW TABLES;"

  echo
  echo "CockroachDB is ready on port $CRDB_PORT (HTTP UI: $CRDB_HTTP_PORT)"
  cockroachdb_url
}

cockroachdb_stop() { _stop $CRDB_CONTAINER "CockroachDB"; }
cockroachdb_rm()   { _rm   $CRDB_CONTAINER "CockroachDB container"; }

cockroachdb_url() {
  echo "Plain:       cockroachdb://$CRDB_USER@localhost:$CRDB_PORT/$CRDB_DB?sslmode=disable"
  echo "Postgres:    postgresql://$CRDB_USER@localhost:$CRDB_PORT/$CRDB_DB?sslmode=disable"
}

# --- SQLite ---

sqlite_up() {
  [ -f "$SQLITE_SQL" ] || { echo "Error: $SQLITE_SQL not found"; exit 1; }
  _ensure_seed_data

  echo "Removing old database (if exists)..."
  rm -f "$SQLITE_DB"

  echo "Loading schema and data..."
  # Inline SEED_DIR into the SQL so .import finds the files.
  sed "s|/tmp/vi-sql-seed|${SEED_DIR}|g" "$SQLITE_SQL" | sqlite3 "$SQLITE_DB"

  echo "Verifying tables..."
  sqlite3 "$SQLITE_DB" "SELECT name FROM sqlite_master WHERE type='table' ORDER BY name;"

  echo
  echo "SQLite database created at $SQLITE_DB"
  sqlite_url
}

sqlite_rm() {
  echo "Removing SQLite database..."
  rm -f "$SQLITE_DB" && echo "Removed $SQLITE_DB" || echo "File not found"
}

sqlite_url() {
  echo "Plain:  sqlite://$SQLITE_DB"
}

# --- Bulk operations ---

stop_all() {
  postgres_stop
  postgres_stop_ssl
  mysql_stop
  mariadb_stop
  cockroachdb_stop
}

rm_all() {
  postgres_rm
  postgres_rm_ssl
  mysql_rm
  mariadb_rm
  cockroachdb_rm
}

clean() {
  stop_all
  rm_all
}

case "$1" in
  postgres)
    case "$2" in
      up)       postgres_up ;;
      up-ssl)   postgres_up_ssl ;;
      stop)     postgres_stop ;;
      stop-ssl) postgres_stop_ssl ;;
      rm)       postgres_rm ;;
      rm-ssl)   postgres_rm_ssl ;;
      url)      postgres_url ;;
      *)        usage ;;
    esac
    ;;
  mysql)
    case "$2" in
      up)   mysql_up ;;
      stop) mysql_stop ;;
      rm)   mysql_rm ;;
      url)  mysql_url ;;
      *)    usage ;;
    esac
    ;;
  mariadb)
    case "$2" in
      up)   mariadb_up ;;
      stop) mariadb_stop ;;
      rm)   mariadb_rm ;;
      url)  mariadb_url ;;
      *)    usage ;;
    esac
    ;;
  cockroachdb)
    case "$2" in
      up)   cockroachdb_up ;;
      stop) cockroachdb_stop ;;
      rm)   cockroachdb_rm ;;
      url)  cockroachdb_url ;;
      *)    usage ;;
    esac
    ;;
  sqlite)
    case "$2" in
      up)   sqlite_up ;;
      rm)   sqlite_rm ;;
      url)  sqlite_url ;;
      *)    usage ;;
    esac
    ;;
  stop-all) stop_all ;;
  rm-all)   rm_all ;;
  clean)    clean ;;
  *)        usage ;;
esac
