#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p /tmp/feria

sudo service postgresql start
sudo -u postgres psql -v ON_ERROR_STOP=1 <<'SQL'
DO $$ BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'catalogo') THEN
    CREATE ROLE catalogo LOGIN PASSWORD 'catalogo_dev';
  END IF;
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'pedidos') THEN
    CREATE ROLE pedidos LOGIN PASSWORD 'pedidos_dev';
  END IF;
END $$;
SELECT 'CREATE DATABASE catalogo OWNER catalogo' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'catalogo')\gexec
SELECT 'CREATE DATABASE pedidos OWNER pedidos' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'pedidos')\gexec
SQL

if ! curl -sf http://127.0.0.1:8091/salud >/dev/null; then
  DATABASE_URL='postgres://catalogo:catalogo_dev@127.0.0.1:5432/catalogo?sslmode=disable' \
    SEED=true PORT=8091 \
    ./bin/catalogo > /tmp/feria/catalogo.log 2>&1 &
fi
if ! curl -sf http://127.0.0.1:8092/salud >/dev/null; then
  (cd servicios/pedidos && DATABASE_URL='postgres://pedidos:pedidos_dev@127.0.0.1:5432/pedidos?sslmode=disable' \
    SEED=true CATALOGO_URL=http://127.0.0.1:8091 PORT=8092 \
    .venv/bin/python -m app.main > /tmp/feria/pedidos.log 2>&1 &)
fi

for _ in $(seq 1 40); do
  if curl -sf http://127.0.0.1:8091/listo >/dev/null && curl -sf http://127.0.0.1:8092/listo >/dev/null; then
    break
  fi
  sleep 0.5
done

cd web
exec env CATALOGO_URL=http://127.0.0.1:8091 PEDIDOS_URL=http://127.0.0.1:8092 PORT=3847 npm run dev
