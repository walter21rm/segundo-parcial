import { notFound } from "next/navigation";

import { Aviso } from "@/components/aviso";
import { TarjetaProducto } from "@/components/tarjeta-producto";
import { listarProductos, obtenerProductor } from "@/lib/api";

export default async function ProductorPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const productor = await obtenerProductor(id);
  if (!productor.ok) {
    if (productor.mensaje.includes("No existe")) notFound();
    return <Aviso titulo="No se pudo abrir el puesto" mensaje={productor.mensaje} />;
  }
  const productos = await listarProductos({ productorId: id });
  return (
    <div>
      <p className="text-sm text-muted-foreground">{productor.datos.distrito} · {productor.datos.telefono}</p>
      <h1 className="mt-1 font-display text-4xl md:text-5xl">{productor.datos.nombre}</h1>
      <p className="mt-3 max-w-2xl">{productor.datos.descripcion}</p>
      {productos.ok ? (
        <div className="mt-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {productos.datos.datos.map((producto) => (
            <TarjetaProducto key={producto.id} producto={producto} />
          ))}
        </div>
      ) : (
        <div className="mt-8">
          <Aviso titulo="No se cargaron los productos" mensaje={productos.mensaje} />
        </div>
      )}
    </div>
  );
}
