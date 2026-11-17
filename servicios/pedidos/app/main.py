import os
from urllib.parse import quote

import uvicorn

from app.api import crear_aplicacion
from app.catalogo_client import CatalogoHTTP
from app.db import crear_pool, migrar, sembrar

app = crear_aplicacion()


def url_base() -> str:
    if os.getenv("DATABASE_URL"):
        return os.environ["DATABASE_URL"]
    usuario = os.getenv("PGUSER", "")
    clave = os.getenv("PGPASSWORD", "")
    base = os.getenv("PGDATABASE", "")
    if not usuario or not clave or not base:
        raise RuntimeError("falta DATABASE_URL o PGUSER, PGPASSWORD y PGDATABASE")
    host = os.getenv("PGHOST", "127.0.0.1")
    puerto = os.getenv("PGPORT", "5432")
    return f"postgresql://{quote(usuario)}:{quote(clave)}@{host}:{puerto}/{quote(base)}?sslmode=disable"


def configurar() -> None:
    database_url = url_base()
    pool = crear_pool(database_url)
    migrar(pool)
    if os.getenv("SEED", "false").lower() == "true":
        sembrar(pool)
    timeout = float(os.getenv("CATALOGO_TIMEOUT_SEGUNDOS", "3"))
    app.state.pool = pool
    app.state.catalogo = CatalogoHTTP(os.environ["CATALOGO_URL"], timeout)
    app.state.version = os.getenv("VERSION", "1.0.0")


def servir() -> None:
    configurar()
    uvicorn.run(app, host="0.0.0.0", port=int(os.getenv("PORT", "8092")))


if __name__ == "__main__":
    servir()
