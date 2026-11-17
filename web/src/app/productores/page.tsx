import Link from "next/link";

import { Aviso } from "@/components/aviso";
import { listarProductores } from "@/lib/api";

export default async function Productores() {
  const resultado = await listarProductores();
  if (!resultado.ok) return <Aviso titulo="No se ven los productores" mensaje={resultado.mensaje} />;
  return (
    <div>
      <h1 className="font-display text-4xl md:text-5xl">Los puestos</h1>
      <p className="mt-2 max-w-xl text-muted-foreground">Cada productor vende por su cuenta. El pedido junta lo que elijas de varios.</p>
      <div className="mt-8 grid gap-4 md:grid-cols-2">
        {resultado.datos.datos.map((productor) => (
          <Link key={productor.id} href={`/productores/${productor.id}`} className="rounded-xl border border-border bg-card p-5 hover:border-primary">
            <p className="text-xs uppercase tracking-wide text-muted-foreground">{productor.distrito}</p>
            <h2 className="mt-1 font-display text-3xl">{productor.nombre}</h2>
            <p className="mt-2 text-sm leading-6">{productor.descripcion}</p>
            <p className="mt-4 text-sm text-accent-foreground">
              {productor.productos_activos} {productor.productos_activos === 1 ? "producto a la venta" : "productos a la venta"}
            </p>
          </Link>
        ))}
      </div>
    </div>
  );
}
