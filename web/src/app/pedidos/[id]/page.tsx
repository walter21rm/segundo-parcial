import { notFound } from "next/navigation";

import { AccionesPedido } from "@/components/acciones-pedido";
import { Aviso } from "@/components/aviso";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { obtenerPedido } from "@/lib/api";
import { cantidadVisible, etiquetaEstado, soles, unidadVisible } from "@/lib/formato";

export default async function PedidoPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const resultado = await obtenerPedido(id);
  if (!resultado.ok) {
    if (resultado.mensaje.includes("No existe")) notFound();
    return <Aviso titulo="No se pudo abrir el pedido" mensaje={resultado.mensaje} />;
  }
  const pedido = resultado.datos;
  return (
    <article className="mx-auto max-w-2xl">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p className="text-sm text-muted-foreground">{pedido.cliente.nombre} · {pedido.cliente.distrito}</p>
          <h1 className="font-display text-4xl">Pedido</h1>
        </div>
        <Badge variant={pedido.estado === "confirmado" ? "default" : "secondary"}>{etiquetaEstado(pedido.estado)}</Badge>
      </div>
      <ul className="mt-8 space-y-4">
        {pedido.lineas.map((linea) => (
          <li key={linea.id} className="flex items-baseline justify-between gap-4">
            <div>
              <p className="font-medium">{linea.nombre_producto}</p>
              <p className="text-sm text-muted-foreground">
                {cantidadVisible(linea.cantidad, linea.unidad)} {unidadVisible(linea.unidad)} · {soles(linea.precio_unitario_centavos)}
              </p>
            </div>
            <p>{soles(linea.subtotal_centavos)}</p>
          </li>
        ))}
      </ul>
      <Separator className="my-6" />
      <p className="flex items-baseline justify-between font-display text-3xl">
        <span>Total</span>
        <span>{soles(pedido.total_centavos)}</span>
      </p>
      {pedido.notas && <p className="mt-4 text-sm text-muted-foreground">Nota: {pedido.notas}</p>}
      <div className="mt-6">
        <AccionesPedido id={pedido.id} estado={pedido.estado} />
      </div>
    </article>
  );
}
