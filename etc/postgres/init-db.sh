#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE USER go WITH PASSWORD 'go';
    CREATE DATABASE deadlocks;
    GRANT ALL PRIVILEGES ON DATABASE deadlocks TO go;
EOSQL
