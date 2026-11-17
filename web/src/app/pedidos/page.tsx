import Link from "next/link";

import { Aviso } from "@/components/aviso";
import { Badge } from "@/components/ui/badge";
import { listarPedidos } from "@/lib/api";
import { etiquetaEstado, soles } from "@/lib/formato";

export default async function Pedidos() {
  const resultado = await listarPedidos();
  if (!resultado.ok) return <Aviso titulo="No se pueden listar los pedidos" mensaje={resultado.mensaje} />;
  return (
    <div>
      <h1 className="font-display text-4xl md:text-5xl">Pedidos</h1>
      <p className="mt-2 max-w-xl text-muted-foreground">Pendiente reserva el stock. Confirmado lo deja apartado. Cancelado lo devuelve al puesto.</p>
      {resultado.datos.datos.length === 0 ? (
        <div className="mt-8">
          <Aviso titulo="Todavía no hay pedidos" mensaje="Arma el primero desde el puesto. Rosa, Luis y Elena ya están como clientes de ejemplo." />
        </div>
      ) : (
        <ul className="mt-8 divide-y divide-border rounded-xl border border-border bg-card">
          {resultado.datos.datos.map((pedido) => (
            <li key={pedido.id}>
              <Link href={`/pedidos/${pedido.id}`} className="flex flex-wrap items-center justify-between gap-3 px-4 py-4 hover:bg-secondary/60">
                <div>
                  <p className="font-medium">{pedido.cliente.nombre}</p>
                  <p className="text-sm text-muted-foreground">
                    {pedido.lineas.length} {pedido.lineas.length === 1 ? "producto" : "productos"} · {pedido.cliente.distrito}
                  </p>
                </div>
                <div className="flex items-center gap-3">
                  <Badge variant={pedido.estado === "cancelado" ? "outline" : "secondary"}>{etiquetaEstado(pedido.estado)}</Badge>
                  <span className="font-display text-xl">{soles(pedido.total_centavos)}</span>
                </div>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
