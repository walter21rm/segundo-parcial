import uuid
from datetime import datetime, timezone
from decimal import Decimal

from app.db import intentar_lock, soltar_lock
from app.dinero import formato_cantidad, hash_cuerpo, subtotal_centavos
from app.errores import ApiError, CatalogoError

ESTADOS_PUBLICOS = ("pendiente", "confirmado", "cancelado")
ESTADOS_CANCELABLES = ("pendiente", "confirmado", "reservando", "compensacion_incompleta")


def iso(valor: datetime) -> str:
    return valor.astimezone(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def cantidad_texto(valor) -> str:
    return formato_cantidad(Decimal(valor))


def crear_cliente(pool, nombre: str, telefono: str, distrito: str) -> dict:
    cliente_id = str(uuid.uuid4())
    with pool.connection() as conn:
        fila = conn.execute(
            """
            INSERT INTO clientes (id, nombre, telefono, distrito)
            VALUES (%s, %s, %s, %s)
            RETURNING id::text, nombre, telefono, distrito, creado_en
            """,
            (cliente_id, nombre, telefono, distrito),
        ).fetchone()
        conn.commit()
    return _cliente(fila)


def listar_clientes(pool, limite: int, desplazamiento: int) -> dict:
    with pool.connection() as conn:
        total = conn.execute("SELECT count(*) AS n FROM clientes").fetchone()["n"]
        filas = conn.execute(
            """
            SELECT id::text, nombre, telefono, distrito, creado_en
            FROM clientes
            ORDER BY nombre ASC, id ASC
            LIMIT %s OFFSET %s
            """,
            (limite, desplazamiento),
        ).fetchall()
    return _pagina([_cliente(f) for f in filas], limite, desplazamiento, total)


def obtener_cliente(pool, cliente_id: str) -> dict:
    with pool.connection() as conn:
        fila = conn.execute(
            "SELECT id::text, nombre, telefono, distrito, creado_en FROM clientes WHERE id = %s",
            (cliente_id,),
        ).fetchone()
    if fila is None:
        raise ApiError(404, "CLIENTE_NO_ENCONTRADO", "No existe el cliente.", {"id": cliente_id})
    return _cliente(fila)


def listar_pedidos(pool, estado: str | None, cliente_id: str | None, limite: int, desplazamiento: int) -> dict:
    where = ["TRUE"]
    args: list = []
    if estado:
        where.append("estado = %s")
        args.append(estado)
    else:
        where.append("estado = ANY(%s)")
        args.append(list(ESTADOS_PUBLICOS))
    if cliente_id:
        where.append("cliente_id = %s")
        args.append(cliente_id)
    filtro = " AND ".join(where)
    with pool.connection() as conn:
        total = conn.execute(f"SELECT count(*) AS n FROM pedidos WHERE {filtro}", args).fetchone()["n"]
        filas = conn.execute(
            f"""
            SELECT id::text FROM pedidos
            WHERE {filtro}
            ORDER BY creado_en DESC, id ASC
            LIMIT %s OFFSET %s
            """,
            [*args, limite, desplazamiento],
        ).fetchall()
        datos = [_cargar(conn, fila["id"]) for fila in filas]
    return _pagina(datos, limite, desplazamiento, total)


def obtener_pedido(pool, pedido_id: str) -> dict:
    with pool.connection() as conn:
        existe = conn.execute("SELECT 1 FROM pedidos WHERE id = %s", (pedido_id,)).fetchone()
        if existe is None:
            raise ApiError(404, "NO_ENCONTRADO", "No existe el pedido.", {"recurso": "pedido", "id": pedido_id})
        return _cargar(conn, pedido_id)


def crear_pedido(pool, catalogo, cliente_id: str, notas: str, lineas: list[dict], clave: str | None):
    cuerpo = hash_cuerpo(cliente_id, notas, lineas)
    with pool.connection() as conn:
        if clave and not intentar_lock(conn, f"clave:{clave}"):
            raise ApiError(
                409,
                "IDEMPOTENCIA_EN_CURSO",
                "Ese pedido todavía se está registrando.",
                {"clave": clave},
            )
        try:
            pedido_id, estado = _reclamar(conn, cliente_id, notas, clave, cuerpo)
            if estado in ESTADOS_PUBLICOS:
                return _cargar(conn, pedido_id), 200
            if not intentar_lock(conn, f"mutar:{pedido_id}"):
                raise ApiError(409, "PEDIDO_EN_CURSO", "Ese pedido está siendo actualizado.", {"id": pedido_id})
            try:
                if estado == "compensacion_incompleta":
                    if not _compensar(conn, catalogo, pedido_id):
                        raise _incompleta(pedido_id)
                    with conn.transaction():
                        conn.execute("DELETE FROM lineas WHERE pedido_id = %s", (pedido_id,))
                        conn.execute("DELETE FROM pedidos WHERE id = %s", (pedido_id,))
                    soltar_lock(conn, f"mutar:{pedido_id}")
                    pedido_id, estado = _reclamar(conn, cliente_id, notas, clave, cuerpo)
                    if not intentar_lock(conn, f"mutar:{pedido_id}"):
                        raise ApiError(409, "PEDIDO_EN_CURSO", "Ese pedido está siendo actualizado.", {"id": pedido_id})
                _reservar_lineas(conn, catalogo, pedido_id, lineas)
                with conn.transaction():
                    total = conn.execute(
                        "SELECT COALESCE(SUM(subtotal_centavos), 0) AS n FROM lineas WHERE pedido_id = %s",
                        (pedido_id,),
                    ).fetchone()["n"]
                    conn.execute(
                        """
                        UPDATE pedidos
                        SET estado = 'pendiente', total_centavos = %s, actualizado_en = now()
                        WHERE id = %s
                        """,
                        (total, pedido_id),
                    )
                return _cargar(conn, pedido_id), 201
            finally:
                soltar_lock(conn, f"mutar:{pedido_id}")
        finally:
            if clave:
                soltar_lock(conn, f"clave:{clave}")


def confirmar_pedido(pool, pedido_id: str) -> dict:
    with pool.connection() as conn:
        if not intentar_lock(conn, f"mutar:{pedido_id}"):
            raise ApiError(409, "PEDIDO_EN_CURSO", "Ese pedido está siendo actualizado.", {"id": pedido_id})
        try:
            with conn.transaction():
                fila = conn.execute(
                    "SELECT estado FROM pedidos WHERE id = %s FOR UPDATE",
                    (pedido_id,),
                ).fetchone()
                if fila is None:
                    raise ApiError(404, "NO_ENCONTRADO", "No existe el pedido.", {"recurso": "pedido", "id": pedido_id})
                if fila["estado"] == "confirmado":
                    pass
                elif fila["estado"] == "pendiente":
                    conn.execute(
                        "UPDATE pedidos SET estado = 'confirmado', actualizado_en = now() WHERE id = %s",
                        (pedido_id,),
                    )
                else:
                    raise ApiError(
                        409,
                        "ESTADO_INVALIDO",
                        "Solo se puede confirmar un pedido pendiente.",
                        {"estado": fila["estado"]},
                    )
            return _cargar(conn, pedido_id)
        finally:
            soltar_lock(conn, f"mutar:{pedido_id}")


def cancelar_pedido(pool, catalogo, pedido_id: str) -> dict:
    with pool.connection() as conn:
        previa = conn.execute(
            "SELECT clave_idempotencia FROM pedidos WHERE id = %s",
            (pedido_id,),
        ).fetchone()
        if previa is None:
            raise ApiError(404, "NO_ENCONTRADO", "No existe el pedido.", {"recurso": "pedido", "id": pedido_id})
        clave = previa["clave_idempotencia"]
        if clave and not intentar_lock(conn, f"clave:{clave}"):
            raise ApiError(409, "IDEMPOTENCIA_EN_CURSO", "Ese pedido todavía se está registrando.", {"clave": clave})
        try:
            if not intentar_lock(conn, f"mutar:{pedido_id}"):
                raise ApiError(409, "PEDIDO_EN_CURSO", "Ese pedido está siendo actualizado.", {"id": pedido_id})
            try:
                fila = conn.execute("SELECT estado FROM pedidos WHERE id = %s", (pedido_id,)).fetchone()
                if fila["estado"] == "cancelado":
                    return _cargar(conn, pedido_id)
                if fila["estado"] not in ESTADOS_CANCELABLES:
                    raise ApiError(
                        409,
                        "ESTADO_INVALIDO",
                        "Ese pedido no se puede cancelar.",
                        {"estado": fila["estado"]},
                    )
                if not _compensar(conn, catalogo, pedido_id):
                    if fila["estado"] in ("reservando", "compensacion_incompleta"):
                        with conn.transaction():
                            conn.execute(
                                """
                                UPDATE pedidos
                                SET estado = 'compensacion_incompleta', actualizado_en = now()
                                WHERE id = %s
                                """,
                                (pedido_id,),
                            )
                    raise _incompleta(pedido_id)
                with conn.transaction():
                    conn.execute(
                        "UPDATE pedidos SET estado = 'cancelado', actualizado_en = now() WHERE id = %s",
                        (pedido_id,),
                    )
                return _cargar(conn, pedido_id)
            finally:
                soltar_lock(conn, f"mutar:{pedido_id}")
        finally:
            if clave:
                soltar_lock(conn, f"clave:{clave}")


def _reclamar(conn, cliente_id: str, notas: str, clave: str | None, cuerpo: str):
    with conn.transaction():
        cliente = conn.execute("SELECT 1 FROM clientes WHERE id = %s", (cliente_id,)).fetchone()
        if cliente is None:
            raise ApiError(404, "CLIENTE_NO_ENCONTRADO", "No existe el cliente.", {"id": cliente_id})
        pedido = None
        if clave:
            pedido = conn.execute(
                """
                SELECT id::text, estado, cuerpo_hash
                FROM pedidos
                WHERE clave_idempotencia = %s
                FOR UPDATE
                """,
                (clave,),
            ).fetchone()
        if pedido is not None:
            if pedido["cuerpo_hash"] != cuerpo:
                raise ApiError(
                    409,
                    "IDEMPOTENCIA_CONFLICTO",
                    "La clave de idempotencia ya se usó con otro pedido.",
                    {"clave": clave},
                )
            return pedido["id"], pedido["estado"]
        pedido_id = str(uuid.uuid4())
        conn.execute(
            """
            INSERT INTO pedidos (id, cliente_id, estado, total_centavos, notas, clave_idempotencia, cuerpo_hash)
            VALUES (%s, %s, 'reservando', 0, %s, %s, %s)
            """,
            (pedido_id, cliente_id, notas, clave, cuerpo),
        )
        return pedido_id, "reservando"


def _reservar_lineas(conn, catalogo, pedido_id: str, lineas: list[dict]) -> None:
    with conn.transaction():
        hechas = {
            fila["producto_id"]
            for fila in conn.execute(
                "SELECT producto_id::text FROM lineas WHERE pedido_id = %s",
                (pedido_id,),
            ).fetchall()
        }
    for linea in lineas:
        if linea["producto_id"] in hechas:
            continue
        referencia = f"ped:{pedido_id}:prod:{linea['producto_id']}"
        try:
            reserva = catalogo.reservar(
                linea["producto_id"],
                formato_cantidad(linea["cantidad"]),
                referencia,
            )
        except CatalogoError as exc:
            if not _compensar(conn, catalogo, pedido_id):
                with conn.transaction():
                    conn.execute(
                        """
                        UPDATE pedidos
                        SET estado = 'compensacion_incompleta', actualizado_en = now()
                        WHERE id = %s
                        """,
                        (pedido_id,),
                    )
                raise _incompleta(pedido_id) from exc
            with conn.transaction():
                conn.execute("DELETE FROM lineas WHERE pedido_id = %s", (pedido_id,))
                conn.execute("DELETE FROM pedidos WHERE id = %s", (pedido_id,))
            raise
        cantidad = Decimal(reserva["cantidad"])
        with conn.transaction():
            conn.execute(
                """
                INSERT INTO lineas (
                    id, pedido_id, producto_id, productor_id, reserva_id, nombre_producto,
                    unidad, precio_unitario_centavos, cantidad, subtotal_centavos
                )
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s::numeric, %s)
                """,
                (
                    str(uuid.uuid4()),
                    pedido_id,
                    reserva["producto_id"],
                    reserva["productor_id"],
                    reserva["id"],
                    reserva["nombre"],
                    reserva["unidad"],
                    int(reserva["precio_centavos"]),
                    formato_cantidad(cantidad),
                    subtotal_centavos(int(reserva["precio_centavos"]), cantidad),
                ),
            )


def _compensar(conn, catalogo, pedido_id: str) -> bool:
    lineas = conn.execute(
        """
        SELECT reserva_id::text
        FROM lineas
        WHERE pedido_id = %s
        ORDER BY id DESC
        """,
        (pedido_id,),
    ).fetchall()
    bien = True
    for linea in lineas:
        try:
            catalogo.compensar(linea["reserva_id"])
        except CatalogoError:
            bien = False
    return bien


def _incompleta(pedido_id: str) -> ApiError:
    return ApiError(
        500,
        "COMPENSACION_INCOMPLETA",
        "No se pudo devolver todo el stock reservado.",
        {"pedido_id": pedido_id},
    )


def _cargar(conn, pedido_id: str) -> dict:
    pedido = conn.execute(
        """
        SELECT p.id::text, p.estado, p.total_centavos, p.notas, p.creado_en, p.actualizado_en,
               c.id::text AS cliente_id, c.nombre, c.telefono, c.distrito
        FROM pedidos p
        JOIN clientes c ON c.id = p.cliente_id
        WHERE p.id = %s
        """,
        (pedido_id,),
    ).fetchone()
    lineas = conn.execute(
        """
        SELECT id::text, producto_id::text, productor_id::text, reserva_id::text, nombre_producto,
               unidad, precio_unitario_centavos, cantidad, subtotal_centavos
        FROM lineas
        WHERE pedido_id = %s
        ORDER BY nombre_producto ASC, id ASC
        """,
        (pedido_id,),
    ).fetchall()
    return {
        "id": pedido["id"],
        "cliente": {
            "id": pedido["cliente_id"],
            "nombre": pedido["nombre"],
            "telefono": pedido["telefono"],
            "distrito": pedido["distrito"],
        },
        "estado": pedido["estado"],
        "total_centavos": pedido["total_centavos"],
        "notas": pedido["notas"],
        "lineas": [
            {
                "id": linea["id"],
                "producto_id": linea["producto_id"],
                "productor_id": linea["productor_id"],
                "reserva_id": linea["reserva_id"],
                "nombre_producto": linea["nombre_producto"],
                "unidad": linea["unidad"],
                "precio_unitario_centavos": linea["precio_unitario_centavos"],
                "cantidad": cantidad_texto(linea["cantidad"]),
                "subtotal_centavos": linea["subtotal_centavos"],
            }
            for linea in lineas
        ],
        "creado_en": iso(pedido["creado_en"]),
        "actualizado_en": iso(pedido["actualizado_en"]),
    }


def _cliente(fila) -> dict:
    return {
        "id": fila["id"],
        "nombre": fila["nombre"],
        "telefono": fila["telefono"],
        "distrito": fila["distrito"],
        "creado_en": iso(fila["creado_en"]),
    }


def _pagina(datos, limite, desplazamiento, total) -> dict:
    return {
        "datos": datos,
        "paginacion": {"limite": limite, "desplazamiento": desplazamiento, "total": total},
    }
