import Link from "next/link";

import { Aviso } from "@/components/aviso";
import { TarjetaProducto } from "@/components/tarjeta-producto";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { listarProductos } from "@/lib/api";
import { CATEGORIAS } from "@/lib/formato";

export default async function Inicio({
  searchParams,
}: {
  searchParams: Promise<{ q?: string; categoria?: string }>;
}) {
  const params = await searchParams;
  const q = params.q?.trim() ?? "";
  const categoria = params.categoria ?? "";
  const resultado = await listarProductos({ q, categoria });

  return (
    <div>
      <div className="flex flex-col gap-6 md:flex-row md:items-end md:justify-between">
        <div className="max-w-2xl">
          <p className="text-sm font-medium text-accent-foreground">Feria de hoy</p>
          <h1 className="mt-1 font-display text-4xl tracking-tight md:text-6xl">Lo que llegó esta mañana</h1>
          <p className="mt-3 text-muted-foreground">
            Cinco puestos de Lima. Eliges aquí, el pedido reserva el stock en el acto y guarda el precio de ese momento.
          </p>
        </div>
        <form className="flex w-full gap-2 md:max-w-sm" action="/">
          {categoria && <input type="hidden" name="categoria" value={categoria} />}
          <Input name="q" defaultValue={q} placeholder="Buscar palta, miel, quinua" aria-label="Buscar productos" />
          <Button type="submit">Buscar</Button>
        </form>
      </div>
      <div className="mt-6 flex gap-2 overflow-x-auto pb-1">
        <Link
          href={q ? `/?q=${encodeURIComponent(q)}` : "/"}
          className={`rounded-full border px-3 py-1 text-sm ${categoria === "" ? "border-primary bg-primary text-primary-foreground" : "border-border bg-card"}`}
        >
          Todo
        </Link>
        {CATEGORIAS.map((item) => (
          <Link
            key={item.id}
            href={`/?categoria=${item.id}${q ? `&q=${encodeURIComponent(q)}` : ""}`}
            className={`rounded-full border px-3 py-1 text-sm whitespace-nowrap ${categoria === item.id ? "border-primary bg-primary text-primary-foreground" : "border-border bg-card"}`}
          >
            {item.nombre}
          </Link>
        ))}
      </div>
      {resultado.ok ? (
        resultado.datos.datos.length === 0 ? (
          <div className="mt-8">
            <Aviso titulo="Nada con ese nombre" mensaje="Prueba otra búsqueda o mira todos los puestos." />
          </div>
        ) : (
          <div className="mt-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {resultado.datos.datos.map((producto) => (
              <TarjetaProducto key={producto.id} producto={producto} />
            ))}
          </div>
        )
      ) : (
        <div className="mt-8">
          <Aviso titulo="El catálogo no responde" mensaje={resultado.mensaje} />
        </div>
      )}
    </div>
  );
}
