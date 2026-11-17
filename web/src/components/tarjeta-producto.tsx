"use client";

import { toast } from "sonner";

import { useCarrito } from "@/components/carrito";
import { Button } from "@/components/ui/button";
import { cantidadVisible, soles, unidadVisible } from "@/lib/formato";
import type { Producto } from "@/lib/tipos";

export function TarjetaProducto({ producto }: { producto: Producto }) {
  const { agregar } = useCarrito();
  const agotado = Number(producto.stock) <= 0;

  return (
    <article className="flex h-full flex-col rounded-xl border border-border bg-card p-4 shadow-sm">
      <p className="text-xs uppercase tracking-wide text-muted-foreground">{producto.productor.distrito}</p>
      <h2 className="mt-1 font-display text-2xl leading-tight">{producto.nombre}</h2>
      <p className="mt-1 text-sm text-muted-foreground">{producto.productor.nombre}</p>
      <p className="mt-3 flex-1 text-sm leading-6">{producto.descripcion}</p>
      <div className="mt-4 flex items-end justify-between gap-3">
        <div>
          <p className="font-display text-2xl">{soles(producto.precio_centavos)}</p>
          <p className="text-xs text-muted-foreground">
            por {unidadVisible(producto.unidad)} · quedan {cantidadVisible(producto.stock, producto.unidad)}
          </p>
        </div>
        <Button
          type="button"
          disabled={agotado}
          onClick={() => {
            const estado = agregar(producto);
            toast(estado === "nuevo" ? `${producto.nombre} entró al pedido` : "Ese producto ya está en el pedido");
          }}
        >
          {agotado ? "Agotado" : "Agregar"}
        </Button>
      </div>
    </article>
  );
}
