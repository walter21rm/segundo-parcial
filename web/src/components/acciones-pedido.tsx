"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";

import { accionCancelar, accionConfirmar } from "@/app/acciones";
import { Button } from "@/components/ui/button";

export function AccionesPedido({ id, estado }: { id: string; estado: string }) {
  const router = useRouter();
  const [ocupado, setOcupado] = useState<"confirmar" | "cancelar" | null>(null);

  async function correr(accion: "confirmar" | "cancelar") {
    setOcupado(accion);
    const resultado = accion === "confirmar" ? await accionConfirmar(id) : await accionCancelar(id);
    setOcupado(null);
    if (!resultado.ok) {
      toast.error(resultado.mensaje);
      return;
    }
    toast.success(accion === "confirmar" ? "Pedido confirmado" : "Pedido cancelado y stock devuelto");
    router.refresh();
  }

  if (estado === "cancelado") {
    return <p className="text-sm text-muted-foreground">Este pedido quedó cancelado. El stock volvió al puesto.</p>;
  }

  return (
    <div className="flex flex-wrap gap-2">
      {estado === "pendiente" && (
        <Button disabled={ocupado !== null} onClick={() => correr("confirmar")}>
          {ocupado === "confirmar" ? "Confirmando…" : "Confirmar pedido"}
        </Button>
      )}
      <Button variant="outline" disabled={ocupado !== null} onClick={() => correr("cancelar")}>
        {ocupado === "cancelar" ? "Cancelando…" : "Cancelar y devolver stock"}
      </Button>
    </div>
  );
}
