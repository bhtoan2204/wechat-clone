#!/bin/sh

set -eu

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<SQL
-- Debezium pgoutput needs logical replication privileges on the local app role.
ALTER ROLE "${POSTGRES_USER}" WITH REPLICATION;
SQL
