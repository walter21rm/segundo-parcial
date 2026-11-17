"use client";

import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";

import { accionCrearCliente, accionCrearPedido } from "@/app/acciones";
import { useCarrito } from "@/components/carrito";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { DISTRITOS, cantidadVisible, soles, unidadVisible } from "@/lib/formato";
import type { Cliente } from "@/lib/tipos";

export function FormularioPedido({ clientes }: { clientes: Cliente[] }) {
  const { items, cambiar, quitar, vaciar } = useCarrito();
  const router = useRouter();
  const [clave] = useState(() => crypto.randomUUID());
  const [modo, setModo] = useState<"existente" | "nuevo">(clientes.length ? "existente" : "nuevo");
  const [clienteId, setClienteId] = useState(clientes[0]?.id ?? "");
  const [nombre, setNombre] = useState("");
  const [telefono, setTelefono] = useState("");
  const [distrito, setDistrito] = useState(DISTRITOS[0]);
  const [notas, setNotas] = useState("");
  const [enviando, setEnviando] = useState(false);

  const total = useMemo(
    () =>
      items.reduce((suma, item) => {
        const cantidad = Number(item.cantidad);
        if (!Number.isFinite(cantidad)) return suma;
        return suma + Math.round(item.precioCentavos * cantidad);
      }, 0),
    [items],
  );

  async function enviar(evento: React.FormEvent) {
    evento.preventDefault();
    if (items.length === 0) return;
    setEnviando(true);
    try {
      let id = clienteId;
      if (modo === "nuevo") {
        const creado = await accionCrearCliente({
          nombre,
          telefono: `+51${telefono.replace(/\D/g, "")}`,
          distrito,
        });
        if (!creado.ok) {
          toast.error(creado.mensaje);
          return;
        }
        id = creado.datos.id;
      }
      const pedido = await accionCrearPedido({
        clienteId: id,
        notas,
        clave,
        lineas: items.map((item) => ({ producto_id: item.productoId, cantidad: item.cantidad })),
      });
      if (!pedido.ok) {
        toast.error(pedido.mensaje);
        return;
      }
      vaciar();
      router.push(`/pedidos/${pedido.datos.id}`);
    } finally {
      setEnviando(false);
    }
  }

  if (items.length === 0) {
    return (
      <div className="rounded-xl border border-dashed border-border bg-card px-6 py-16 text-center">
        <h1 className="font-display text-3xl">El pedido está vacío</h1>
        <p className="mx-auto mt-2 max-w-md text-muted-foreground">
          Vuelve al puesto y agrega lo que quieras llevar. El stock se reserva recién cuando confirmas aquí.
        </p>
        <Button className="mt-6" onClick={() => router.push("/")}>
          Ver el puesto
        </Button>
      </div>
    );
  }

  return (
    <form onSubmit={enviar} className="grid gap-8 lg:grid-cols-[1.2fr_0.8fr]">
      <section className="space-y-3">
        {items.map((item) => (
          <div key={item.productoId} className="rounded-xl border border-border bg-card p-4">
            <div className="flex items-start justify-between gap-3">
              <div>
                <h2 className="font-display text-2xl">{item.nombre}</h2>
                <p className="text-sm text-muted-foreground">
                  {item.productor} · {soles(item.precioCentavos)} / {unidadVisible(item.unidad)} · hay{" "}
                  {cantidadVisible(item.stock, item.unidad)}
                </p>
              </div>
              <button type="button" className="text-sm text-muted-foreground underline" onClick={() => quitar(item.productoId)}>
                Quitar
              </button>
            </div>
            <Label className="mt-3" htmlFor={`cant-${item.productoId}`}>
              Cantidad
            </Label>
            <Input
              id={`cant-${item.productoId}`}
              className="mt-1 max-w-40"
              inputMode="decimal"
              value={item.cantidad}
              onChange={(evento) => cambiar(item.productoId, evento.target.value)}
              required
            />
          </div>
        ))}
      </section>
      <section className="h-fit space-y-4 rounded-xl border border-border bg-card p-5">
        <h1 className="font-display text-3xl">Quién lo lleva</h1>
        {clientes.length > 0 && (
          <div className="flex gap-2 text-sm">
            <button type="button" className={modo === "existente" ? "underline" : "text-muted-foreground"} onClick={() => setModo("existente")}>
              Cliente del puesto
            </button>
            <button type="button" className={modo === "nuevo" ? "underline" : "text-muted-foreground"} onClick={() => setModo("nuevo")}>
              Primera vez
            </button>
          </div>
        )}
        {modo === "existente" ? (
          <div>
            <Label htmlFor="cliente">Cliente</Label>
            <select
              id="cliente"
              className="mt-1 h-9 w-full rounded-lg border border-input bg-transparent px-3 text-sm"
              value={clienteId}
              onChange={(evento) => setClienteId(evento.target.value)}
            >
              {clientes.map((cliente) => (
                <option key={cliente.id} value={cliente.id}>
                  {cliente.nombre} · {cliente.distrito}
                </option>
              ))}
            </select>
          </div>
        ) : (
          <div className="space-y-3">
            <div>
              <Label htmlFor="nombre">Nombre</Label>
              <Input id="nombre" className="mt-1" value={nombre} onChange={(e) => setNombre(e.target.value)} required minLength={2} />
            </div>
            <div>
              <Label htmlFor="telefono">Celular</Label>
              <div className="mt-1 flex">
                <span className="inline-flex items-center rounded-l-lg border border-r-0 border-input px-3 text-sm text-muted-foreground">+51</span>
                <Input
                  id="telefono"
                  className="rounded-l-none"
                  inputMode="numeric"
                  pattern="[0-9]{9}"
                  maxLength={9}
                  value={telefono}
                  onChange={(e) => setTelefono(e.target.value.replace(/\D/g, "").slice(0, 9))}
                  required
                />
              </div>
            </div>
            <div>
              <Label htmlFor="distrito">Distrito</Label>
              <select
                id="distrito"
                className="mt-1 h-9 w-full rounded-lg border border-input bg-transparent px-3 text-sm"
                value={distrito}
                onChange={(evento) => setDistrito(evento.target.value)}
              >
                {DISTRITOS.map((item) => (
                  <option key={item} value={item}>
                    {item}
                  </option>
                ))}
              </select>
            </div>
          </div>
        )}
        <div>
          <Label htmlFor="notas">Nota para el puesto</Label>
          <Textarea id="notas" className="mt-1" maxLength={500} value={notas} onChange={(e) => setNotas(e.target.value)} />
        </div>
        <p className="font-display text-3xl">{soles(total)}</p>
        <p className="text-xs text-muted-foreground">Estimado con el precio de ahora. El total definitivo lo calcula pedidos al reservar el stock.</p>
        <Button type="submit" className="w-full" disabled={enviando}>
          {enviando ? "Reservando stock…" : "Registrar pedido"}
        </Button>
      </section>
    </form>
  );
}
