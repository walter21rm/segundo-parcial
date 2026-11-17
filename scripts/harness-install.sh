#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

sudo DEBIAN_FRONTEND=noninteractive apt-get update -qq
sudo DEBIAN_FRONTEND=noninteractive apt-get install -y -qq postgresql postgresql-contrib python3-venv python3-pip

(cd servicios/catalogo && go mod download)
mkdir -p bin
(cd servicios/catalogo && CGO_ENABLED=0 go build -o "$PWD/../../bin/catalogo" ./cmd/catalogo)

if [[ ! -d servicios/pedidos/.venv ]]; then
  python3 -m venv servicios/pedidos/.venv
fi
servicios/pedidos/.venv/bin/pip install -q -r servicios/pedidos/requirements-dev.txt

(cd web && npm ci)
echo "instalación lista"
