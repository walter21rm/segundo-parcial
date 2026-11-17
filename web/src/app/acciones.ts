"use server";

import { revalidatePath } from "next/cache";

import { cancelarPedido, confirmarPedido, crearCliente, crearPedido } from "@/lib/api";

export async function accionCrearCliente(input: { nombre: string; telefono: string; distrito: string }) {
  return crearCliente(input);
}

export async function accionCrearPedido(input: {
  clienteId: string;
  notas: string;
  lineas: { producto_id: string; cantidad: string }[];
  clave: string;
}) {
  const resultado = await crearPedido(input);
  if (resultado.ok) {
    revalidatePath("/");
    revalidatePath("/pedidos");
  }
  return resultado;
}

export async function accionConfirmar(id: string) {
  const resultado = await confirmarPedido(id);
  revalidatePath("/");
  revalidatePath("/pedidos");
  revalidatePath(`/pedidos/${id}`);
  return resultado;
}

export async function accionCancelar(id: string) {
  const resultado = await cancelarPedido(id);
  revalidatePath("/");
  revalidatePath("/pedidos");
  revalidatePath(`/pedidos/${id}`);
  return resultado;
}
