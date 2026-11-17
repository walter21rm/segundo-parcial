import hashlib
import json
import re
from decimal import Decimal, ROUND_HALF_UP

CANTIDAD_RE = re.compile(r"^(?:0|[1-9][0-9]{0,6})(?:\.[0-9]{1,3})?$")
TELEFONO_RE = re.compile(r"^\+51[0-9]{9}$")
CLAVE_RE = re.compile(r"^[A-Za-z0-9._:-]{8,80}$")
UNIDADES_ENTERAS = {"unidad", "atado"}


def solicitud(mensaje: str, campo: str) -> "ApiError":
    from app.errores import ApiError

    return ApiError(400, "SOLICITUD_INVALIDA", mensaje, {"campo": campo})


def texto(valor: str, campo: str, minimo: int, maximo: int) -> str:
    valor = valor.strip()
    if any(ord(c) < 32 for c in valor):
        raise solicitud("El texto tiene caracteres de control.", campo)
    if len(valor) < minimo or len(valor) > maximo:
        raise solicitud("La longitud del campo no es válida.", campo)
    return valor


def telefono(valor: str) -> str:
    valor = valor.strip()
    if not TELEFONO_RE.fullmatch(valor):
        raise solicitud("El teléfono debe empezar con +51 y tener 9 dígitos.", "telefono")
    return valor


def parse_cantidad(raw: str, unidad: str | None = None) -> Decimal:
    raw = raw.strip()
    if not CANTIDAD_RE.fullmatch(raw):
        raise solicitud(
            "La cantidad debe ser un decimal positivo, con hasta 3 decimales.",
            "cantidad",
        )
    cantidad = Decimal(raw)
    if cantidad <= 0 or cantidad > Decimal("10000"):
        raise solicitud("La cantidad debe ser mayor que cero y como máximo 10000.", "cantidad")
    if unidad in UNIDADES_ENTERAS and cantidad != cantidad.to_integral_value():
        raise solicitud("Esta unidad solo admite cantidades enteras.", "cantidad")
    return cantidad


def formato_cantidad(cantidad: Decimal) -> str:
    return f"{cantidad.quantize(Decimal('0.001')):.3f}"


def subtotal_centavos(precio: int, cantidad: Decimal) -> int:
    valor = (Decimal(precio) * cantidad).quantize(Decimal("1"), rounding=ROUND_HALF_UP)
    return int(valor)


def hash_cuerpo(cliente_id: str, notas: str, lineas: list[dict]) -> str:
    canon = {
        "cliente_id": cliente_id,
        "notas": notas,
        "lineas": sorted(
            (
                {
                    "producto_id": linea["producto_id"],
                    "cantidad": formato_cantidad(linea["cantidad"]),
                }
                for linea in lineas
            ),
            key=lambda linea: linea["producto_id"],
        ),
    }
    crudo = json.dumps(canon, sort_keys=True, separators=(",", ":"), ensure_ascii=False)
    return hashlib.sha256(crudo.encode()).hexdigest()
