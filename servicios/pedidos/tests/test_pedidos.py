import os
import uuid

import pytest
from fastapi.testclient import TestClient

from app.api import crear_aplicacion
from app.db import crear_pool, migrar
from app.dinero import subtotal_centavos, parse_cantidad
from app.errores import CatalogoError

PALTA = "10000000-0000-4000-8000-000000000005"
FRESA = "10000000-0000-4000-8000-000000000006"


class FalsoCatalogo:
    def __init__(self):
        self.productos = {
            PALTA: {"productor_id": "22222222-2222-4222-8222-222222222222", "nombre": "Palta Fuerte", "unidad": "kg", "precio_centavos": 890},
            FRESA: {"productor_id": "22222222-2222-4222-8222-222222222222", "nombre": "Fresa", "unidad": "kg", "precio_centavos": 1200},
        }
        self.reservas = {}
        self.fallar = set()

    def reservar(self, producto_id, cantidad, referencia):
        if producto_id in self.fallar:
            raise CatalogoError(409, "STOCK_INSUFICIENTE", "No hay stock suficiente.", {"producto_id": producto_id})
        if producto_id not in self.productos:
            raise CatalogoError(404, "PRODUCTO_NO_ENCONTRADO", "No existe el producto.", {"producto_id": producto_id})
        previa = self.reservas.get(referencia)
        if previa and previa["estado"] == "activa":
            return previa
        producto = self.productos[producto_id]
        reserva = {
            "id": previa["id"] if previa else str(uuid.uuid4()),
            "producto_id": producto_id,
            "productor_id": producto["productor_id"],
            "nombre": producto["nombre"],
            "unidad": producto["unidad"],
            "precio_centavos": producto["precio_centavos"],
            "cantidad": f"{parse_cantidad(cantidad):.3f}",
            "referencia": referencia,
            "estado": "activa",
        }
        self.reservas[referencia] = reserva
        return reserva

    def compensar(self, reserva_id):
        for reserva in self.reservas.values():
            if reserva["id"] == reserva_id:
                reserva["estado"] = "compensada"
                return
        raise CatalogoError(404, "NO_ENCONTRADO", "No existe la reserva.", {"id": reserva_id})


@pytest.fixture(scope="module")
def api():
    url = os.getenv("DATABASE_URL")
    if not url:
        pytest.skip("DATABASE_URL no está definida")
    pool = crear_pool(url)
    migrar(pool)
    with pool.connection() as conn:
        conn.execute("TRUNCATE lineas, pedidos, clientes CASCADE")
    catalogo = FalsoCatalogo()
    app = crear_aplicacion()
    app.state.pool = pool
    app.state.catalogo = catalogo
    app.state.version = "1.0.0"
    with TestClient(app) as cliente:
        yield cliente, catalogo
    pool.close()


def test_subtotal_de_la_palta():
    assert subtotal_centavos(890, parse_cantidad("1.5")) == 1335


def test_alta_idempotente_confirmar_y_cancelar(api):
    cliente, catalogo = api
    alta = cliente.post(
        "/v1/clientes",
        json={"nombre": "Rosa Quispe", "telefono": "+51942222001", "distrito": "Surquillo"},
    )
    assert alta.status_code == 201, alta.text
    cliente_id = alta.json()["id"]
    cuerpo = {
        "cliente_id": cliente_id,
        "notas": "Para el almuerzo",
        "lineas": [{"producto_id": PALTA, "cantidad": "1.5"}],
    }
    creado = cliente.post("/v1/pedidos", json=cuerpo, headers={"Idempotency-Key": "pedido-rosa-001"})
    assert creado.status_code == 201, creado.text
    pedido = creado.json()
    assert pedido["total_centavos"] == 1335
    assert pedido["lineas"][0]["cantidad"] == "1.500"
    assert pedido["estado"] == "pendiente"
    assert creado.headers["idempotent-replayed"] == "false"

    replay = cliente.post("/v1/pedidos", json=cuerpo, headers={"Idempotency-Key": "pedido-rosa-001"})
    assert replay.status_code == 200
    assert replay.json()["id"] == pedido["id"]
    assert replay.headers["idempotent-replayed"] == "true"

    conflicto = cliente.post(
        "/v1/pedidos",
        json={**cuerpo, "notas": "otro"},
        headers={"Idempotency-Key": "pedido-rosa-001"},
    )
    assert conflicto.status_code == 409
    assert conflicto.json()["error"]["codigo"] == "IDEMPOTENCIA_CONFLICTO"

    confirmado = cliente.post(f"/v1/pedidos/{pedido['id']}/confirmar")
    assert confirmado.status_code == 200
    assert confirmado.json()["estado"] == "confirmado"
    assert cliente.post(f"/v1/pedidos/{pedido['id']}/confirmar").json()["estado"] == "confirmado"

    cancelado = cliente.post(f"/v1/pedidos/{pedido['id']}/cancelar")
    assert cancelado.status_code == 200
    assert cancelado.json()["estado"] == "cancelado"
    assert catalogo.reservas[f"ped:{pedido['id']}:prod:{PALTA}"]["estado"] == "compensada"
    assert cliente.post(f"/v1/pedidos/{pedido['id']}/confirmar").status_code == 409


def test_falla_de_stock_compensa_y_no_deja_pedido(api):
    cliente, catalogo = api
    alta = cliente.post(
        "/v1/clientes",
        json={"nombre": "Luis Mendoza", "telefono": "+51942222002", "distrito": "Miraflores"},
    )
    assert alta.status_code == 201, alta.text
    cliente_id = alta.json()["id"]
    antes = cliente.get("/v1/pedidos").json()["paginacion"]["total"]
    catalogo.fallar.add(FRESA)
    respuesta = cliente.post(
        "/v1/pedidos",
        json={
            "cliente_id": cliente_id,
            "notas": "",
            "lineas": [
                {"producto_id": PALTA, "cantidad": "0.500"},
                {"producto_id": FRESA, "cantidad": "1.000"},
            ],
        },
        headers={"Idempotency-Key": "pedido-rosa-falla"},
    )
    assert respuesta.status_code == 409
    assert respuesta.json()["error"]["codigo"] == "STOCK_INSUFICIENTE"
    assert cliente.get("/v1/pedidos").json()["paginacion"]["total"] == antes
    compensadas = [r for r in catalogo.reservas.values() if r["producto_id"] == PALTA and r["estado"] == "compensada"]
    assert compensadas
