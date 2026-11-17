#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
mkdir -p /tmp/feria

if ! pg_isready -h 127.0.0.1 -q; then
  sudo service postgresql start
fi

export CATALOGO_DATABASE_URL="${CATALOGO_DATABASE_URL:-postgres://catalogo:catalogo_dev@127.0.0.1:5432/catalogo?sslmode=disable}"
export PEDIDOS_DATABASE_URL="${PEDIDOS_DATABASE_URL:-postgres://pedidos:pedidos_dev@127.0.0.1:5432/pedidos?sslmode=disable}"

if ! curl -sf http://127.0.0.1:8091/salud >/dev/null; then
  (cd "$ROOT/servicios/catalogo" && DATABASE_URL="$CATALOGO_DATABASE_URL" SEED=true PORT=8091 go run ./cmd/catalogo > /tmp/feria/catalogo.log 2>&1 &)
fi
if ! curl -sf http://127.0.0.1:8092/salud >/dev/null; then
  (cd "$ROOT/servicios/pedidos" && DATABASE_URL="$PEDIDOS_DATABASE_URL" SEED=true CATALOGO_URL=http://127.0.0.1:8091 PORT=8092 "$ROOT/servicios/pedidos/.venv/bin/python" -m app.main > /tmp/feria/pedidos.log 2>&1 &)
fi

for i in $(seq 1 40); do
  if curl -sf http://127.0.0.1:8091/listo >/dev/null && curl -sf http://127.0.0.1:8092/listo >/dev/null; then
    break
  fi
  sleep 0.5
done

cd "$ROOT/web"
exec env CATALOGO_URL=http://127.0.0.1:8091 PEDIDOS_URL=http://127.0.0.1:8092 PORT=3847 npm run dev
