class ApiError(Exception):
    def __init__(self, status: int, codigo: str, mensaje: str, detalles: dict | None = None):
        super().__init__(codigo)
        self.status = status
        self.codigo = codigo
        self.mensaje = mensaje
        self.detalles = detalles or {}


class CatalogoError(ApiError):
    pass
