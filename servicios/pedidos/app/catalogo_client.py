import httpx

from app.errores import CatalogoError


class CatalogoHTTP:
    def __init__(self, base_url: str, timeout: float):
        self.client = httpx.Client(base_url=base_url.rstrip("/"), timeout=timeout)

    def close(self) -> None:
        self.client.close()

    def reservar(self, producto_id: str, cantidad: str, referencia: str) -> dict:
        cuerpo = self._post(
            f"/v1/productos/{producto_id}/reservas",
            {"cantidad": cantidad, "referencia": referencia},
        )
        requeridos = {
            "id",
            "producto_id",
            "productor_id",
            "nombre",
            "unidad",
            "precio_centavos",
            "cantidad",
            "referencia",
            "estado",
        }
        if not requeridos.issubset(cuerpo) or cuerpo.get("estado") != "activa":
            raise CatalogoError(
                503,
                "CATALOGO_NO_DISPONIBLE",
                "El catálogo devolvió una reserva incompleta.",
                {},
            )
        return cuerpo

    def compensar(self, reserva_id: str) -> None:
        self._post(f"/v1/reservas/{reserva_id}/compensar", None)

    def _post(self, path: str, body: dict | None) -> dict:
        respuesta = None
        for _ in range(2):
            try:
                respuesta = self.client.post(path, json=body) if body is not None else self.client.post(path)
                break
            except (httpx.TimeoutException, httpx.TransportError):
                respuesta = None
        if respuesta is None:
            raise CatalogoError(503, "CATALOGO_NO_DISPONIBLE", "No se pudo contactar al catálogo.", {})
        if respuesta.status_code >= 500:
            raise CatalogoError(503, "CATALOGO_NO_DISPONIBLE", "El catálogo falló.", {})
        if respuesta.status_code not in (200, 201):
            codigo = "ERROR_INTERNO"
            mensaje = "El catálogo rechazó la operación."
            detalles: dict = {}
            try:
                error = respuesta.json().get("error", {})
                codigo = error.get("codigo") or codigo
                mensaje = error.get("mensaje") or mensaje
                detalles = error.get("detalles") or {}
            except ValueError:
                pass
            if codigo == "NO_ENCONTRADO" and "/productos/" in path:
                codigo = "PRODUCTO_NO_ENCONTRADO"
            raise CatalogoError(respuesta.status_code, codigo, mensaje, detalles)
        if not respuesta.content:
            return {}
        return respuesta.json()
