import type { Cliente, Pagina, Pedido, Producto, Productor, Resultado } from "@/lib/tipos";

function catalogoURL() {
  return process.env.CATALOGO_URL ?? "http://127.0.0.1:8091";
}

function pedidosURL() {
  return process.env.PEDIDOS_URL ?? "http://127.0.0.1:8092";
}

async function pedir<T>(url: string, init?: RequestInit): Promise<Resultado<T>> {
  try {
    const respuesta = await fetch(url, {
      ...init,
      cache: "no-store",
      headers: { Accept: "application/json", ...init?.headers },
    });
    const cuerpo = (await respuesta.json().catch(() => null)) as {
      error?: { mensaje?: string };
    } | null;
    if (!respuesta.ok) {
      return {
        ok: false,
        mensaje: cuerpo?.error?.mensaje ?? "El servicio rechazó la solicitud.",
      };
    }
    return { ok: true, datos: cuerpo as T };
  } catch {
    return { ok: false, mensaje: "No se pudo contactar el servicio." };
  }
}

export async function consultarSalud() {
  const ping = async (url: string) => {
    try {
      const respuesta = await fetch(url, { cache: "no-store", signal: AbortSignal.timeout(1500) });
      return respuesta.ok;
    } catch {
      return false;
    }
  };
  const [catalogo, pedidos] = await Promise.all([
    ping(`${catalogoURL()}/salud`),
    ping(`${pedidosURL()}/salud`),
  ]);
  return { catalogo, pedidos };
}

export function listarProductos(busqueda: { q?: string; categoria?: string; productorId?: string }) {
  const params = new URLSearchParams({ activo: "true", limite: "100" });
  if (busqueda.q) params.set("q", busqueda.q);
  if (busqueda.categoria) params.set("categoria", busqueda.categoria);
  if (busqueda.productorId) params.set("productor_id", busqueda.productorId);
  return pedir<Pagina<Producto>>(`${catalogoURL()}/v1/productos?${params}`);
}

export function listarProductores() {
  return pedir<Pagina<Productor>>(`${catalogoURL()}/v1/productores?limite=100`);
}

export function obtenerProductor(id: string) {
  return pedir<Productor>(`${catalogoURL()}/v1/productores/${id}`);
}

export function listarClientes() {
  return pedir<Pagina<Cliente>>(`${pedidosURL()}/v1/clientes?limite=100`);
}

export function listarPedidos() {
  return pedir<Pagina<Pedido>>(`${pedidosURL()}/v1/pedidos?limite=50`);
}

export function obtenerPedido(id: string) {
  return pedir<Pedido>(`${pedidosURL()}/v1/pedidos/${id}`);
}

export async function crearCliente(input: { nombre: string; telefono: string; distrito: string }) {
  return pedir<Cliente>(`${pedidosURL()}/v1/clientes`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
}

export async function crearPedido(input: {
  clienteId: string;
  notas: string;
  lineas: { producto_id: string; cantidad: string }[];
  clave: string;
}) {
  return pedir<Pedido>(`${pedidosURL()}/v1/pedidos`, {
    method: "POST",
    headers: { "Content-Type": "application/json", "Idempotency-Key": input.clave },
    body: JSON.stringify({
      cliente_id: input.clienteId,
      notas: input.notas,
      lineas: input.lineas,
    }),
  });
}

export function confirmarPedido(id: string) {
  return pedir<Pedido>(`${pedidosURL()}/v1/pedidos/${id}/confirmar`, { method: "POST" });
}

export function cancelarPedido(id: string) {
  return pedir<Pedido>(`${pedidosURL()}/v1/pedidos/${id}/cancelar`, { method: "POST" });
}
