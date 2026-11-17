from pathlib import Path

from psycopg.rows import dict_row
from psycopg_pool import ConnectionPool

SCHEMA = (Path(__file__).parent / "schema.sql").read_text()

CLIENTES_SEMILLA = [
    ("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1", "Rosa Quispe", "+51942222001", "Surquillo"),
    ("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa2", "Luis Mendoza", "+51942222002", "Miraflores"),
    ("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa3", "Elena Vargas", "+51942222003", "San Borja"),
]


def crear_pool(database_url: str) -> ConnectionPool:
    return ConnectionPool(
        database_url,
        min_size=1,
        max_size=5,
        kwargs={"row_factory": dict_row, "autocommit": True},
        configure=_utc,
        open=True,
    )


def _utc(conn) -> None:
    conn.execute("SET TIME ZONE 'UTC'")


def migrar(pool: ConnectionPool) -> None:
    with pool.connection() as conn:
        for sentencia in SCHEMA.split(";"):
            if sentencia.strip():
                conn.execute(sentencia)
        fila = conn.execute("SELECT version FROM esquema LIMIT 1").fetchone()
        if fila is None:
            conn.execute("INSERT INTO esquema (version) VALUES (1)")
        elif fila["version"] != 1:
            raise RuntimeError(
                f"esquema de pedidos en versión {fila['version']}; esta imagen solo acepta la versión 1"
            )


def sembrar(pool: ConnectionPool) -> None:
    with pool.connection() as conn:
        for cliente in CLIENTES_SEMILLA:
            conn.execute(
                """
                INSERT INTO clientes (id, nombre, telefono, distrito)
                VALUES (%s, %s, %s, %s)
                ON CONFLICT (id) DO NOTHING
                """,
                cliente,
            )


def intentar_lock(conn, clave: str) -> bool:
    fila = conn.execute(
        "SELECT pg_try_advisory_lock(hashtextextended(%s, 0))",
        (clave,),
    ).fetchone()
    return bool(fila["pg_try_advisory_lock"])


def soltar_lock(conn, clave: str) -> None:
    conn.execute("SELECT pg_advisory_unlock(hashtextextended(%s, 0))", (clave,))
