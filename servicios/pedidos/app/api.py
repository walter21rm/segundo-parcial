import logging
import re
import uuid

from fastapi import APIRouter, FastAPI, HTTPException, Request
from fastapi.exceptions import RequestValidationError
from fastapi.responses import JSONResponse
from pydantic import BaseModel, ConfigDict, Field

from app.dinero import CLAVE_RE, parse_cantidad, telefono, texto
from app.errores import ApiError
from app import servicio

router = APIRouter()
UUID_RE = re.compile(r"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")
ESTADOS = {"reservando", "pendiente", "confirmado", "cancelado", "compensacion_incompleta"}


class ClienteIn(BaseModel):
    model_config = ConfigDict(extra="forbid")
    nombre: str
    telefono: str
    distrito: str


class LineaIn(BaseModel):
    model_config = ConfigDict(extra="forbid")
    producto_id: str
    cantidad: str


class PedidoIn(BaseModel):
    model_config = ConfigDict(extra="forbid")
    cliente_id: str
    notas: str = ""
    lineas: list[LineaIn] = Field(default_factory=list)


def crear_aplicacion() -> FastAPI:
    app = FastAPI(title="Pedidos", version="1.0.0", docs_url=None, redoc_url=None, openapi_url=None)
    app.include_router(router)

    @app.middleware("http")
    async def request_id(request: Request, call_next):
        crudo = request.headers.get("x-request-id", "")
        request_id = crudo.lower() if UUID_RE.fullmatch(crudo.lower()) else str(uuid.uuid4())
        request.state.request_id = request_id
        respuesta = await call_next(request)
        respuesta.headers["X-Request-Id"] = request_id
        return respuesta

    @app.exception_handler(ApiError)
    async def api_error(request: Request, exc: ApiError):
        return _error(request, exc.status, exc.codigo, exc.mensaje, exc.detalles)

    @app.exception_handler(RequestValidationError)
    async def json_invalido(request: Request, exc: RequestValidationError):
        return _error(request, 400, "SOLICITUD_INVALIDA", "El JSON no es válido.", {})

    @app.exception_handler(Exception)
    async def error_interno(request: Request, exc: Exception):
        if isinstance(exc, HTTPException):
            return JSONResponse(status_code=exc.status_code, content={"detail": exc.detail})
        logging.getLogger("pedidos").exception("error interno")
        return _error(request, 500, "ERROR_INTERNO", "Error interno.", {})

    return app


def _error(request: Request, status: int, codigo: str, mensaje: str, detalles: dict) -> JSONResponse:
    return JSONResponse(
        status_code=status,
        content={
            "error": {
                "codigo": codigo,
                "mensaje": mensaje,
                "request_id": getattr(request.state, "request_id", ""),
                "detalles": detalles,
            }
        },
    )


@router.get("/salud")
def salud():
    return {"estado": "ok", "servicio": "pedidos"}


@router.get("/listo")
def listo(request: Request):
    try:
        with request.app.state.pool.connection() as conn:
            conn.execute("SELECT 1")
    except Exception:
        raise ApiError(503, "NO_LISTO", "La base de datos no responde.", {})
    return {"estado": "listo", "servicio": "pedidos"}


@router.get("/version")
def version(request: Request):
    return {"servicio": "pedidos", "version": request.app.state.version}


@router.get("/v1/clientes")
def listar_clientes(request: Request, limite: int = 20, desplazamiento: int = 0):
    limite, desplazamiento = _paginacion(limite, desplazamiento)
    return servicio.listar_clientes(request.app.state.pool, limite, desplazamiento)


@router.post("/v1/clientes", status_code=201)
def crear_cliente(request: Request, body: ClienteIn):
    return servicio.crear_cliente(
        request.app.state.pool,
        texto(body.nombre, "nombre", 2, 120),
        telefono(body.telefono),
        texto(body.distrito, "distrito", 2, 80),
    )


@router.get("/v1/clientes/{cliente_id}")
def obtener_cliente(request: Request, cliente_id: str):
    return servicio.obtener_cliente(request.app.state.pool, _id(cliente_id))


@router.get("/v1/pedidos")
def listar_pedidos(
    request: Request,
    estado: str | None = None,
    cliente_id: str | None = None,
    limite: int = 20,
    desplazamiento: int = 0,
):
    limite, desplazamiento = _paginacion(limite, desplazamiento)
    if estado is not None and estado not in ESTADOS:
        raise ApiError(400, "SOLICITUD_INVALIDA", "El estado no es válido.", {"campo": "estado"})
    if cliente_id is not None:
        cliente_id = _id(cliente_id)
    return servicio.listar_pedidos(request.app.state.pool, estado, cliente_id, limite, desplazamiento)


@router.post("/v1/pedidos", status_code=201)
def crear_pedido(request: Request, body: PedidoIn):
    if not 1 <= len(body.lineas) <= 20:
        raise ApiError(400, "SOLICITUD_INVALIDA", "Un pedido tiene entre 1 y 20 líneas.", {"campo": "lineas"})
    vistos = set()
    lineas = []
    for linea in body.lineas:
        producto_id = _id(linea.producto_id)
        if producto_id in vistos:
            raise ApiError(400, "SOLICITUD_INVALIDA", "Hay un producto repetido.", {"producto_id": producto_id})
        vistos.add(producto_id)
        lineas.append({"producto_id": producto_id, "cantidad": parse_cantidad(linea.cantidad)})
    clave = request.headers.get("idempotency-key")
    if clave is not None and not CLAVE_RE.fullmatch(clave):
        raise ApiError(
            400,
            "SOLICITUD_INVALIDA",
            "Idempotency-Key debe tener entre 8 y 80 caracteres seguros.",
            {"campo": "Idempotency-Key"},
        )
    pedido, status = servicio.crear_pedido(
        request.app.state.pool,
        request.app.state.catalogo,
        _id(body.cliente_id),
        texto(body.notas, "notas", 0, 500),
        lineas,
        clave,
    )
    return JSONResponse(status_code=status, content=pedido, headers={"Idempotent-Replayed": "true" if status == 200 else "false"})


@router.get("/v1/pedidos/{pedido_id}")
def obtener_pedido(request: Request, pedido_id: str):
    return servicio.obtener_pedido(request.app.state.pool, _id(pedido_id))


@router.post("/v1/pedidos/{pedido_id}/confirmar")
async def confirmar(request: Request, pedido_id: str):
    await _cuerpo_vacio(request)
    return servicio.confirmar_pedido(request.app.state.pool, _id(pedido_id))


@router.post("/v1/pedidos/{pedido_id}/cancelar")
async def cancelar(request: Request, pedido_id: str):
    await _cuerpo_vacio(request)
    return servicio.cancelar_pedido(request.app.state.pool, request.app.state.catalogo, _id(pedido_id))


def _id(valor: str) -> str:
    valor = valor.lower()
    if not UUID_RE.fullmatch(valor):
        raise ApiError(400, "SOLICITUD_INVALIDA", "El identificador no es un UUID.", {"campo": "id"})
    return valor


def _paginacion(limite: int, desplazamiento: int) -> tuple[int, int]:
    if not 1 <= limite <= 100:
        raise ApiError(400, "SOLICITUD_INVALIDA", "limite debe ser un entero entre 1 y 100.", {"campo": "limite"})
    if not 0 <= desplazamiento <= 10000:
        raise ApiError(
            400,
            "SOLICITUD_INVALIDA",
            "desplazamiento debe ser un entero entre 0 y 10000.",
            {"campo": "desplazamiento"},
        )
    return limite, desplazamiento


async def _cuerpo_vacio(request: Request) -> None:
    cuerpo = await request.body()
    if cuerpo.strip():
        raise ApiError(400, "SOLICITUD_INVALIDA", "Esta operación no recibe cuerpo.", {})
