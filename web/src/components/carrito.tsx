"use client";

import { createContext, useContext, useEffect, useMemo, useState } from "react";

import type { Producto } from "@/lib/tipos";

export type ItemCarrito = {
  productoId: string;
  nombre: string;
  unidad: string;
  precioCentavos: number;
  stock: string;
  productor: string;
  cantidad: string;
};

type Carrito = {
  items: ItemCarrito[];
  listo: boolean;
  agregar: (producto: Producto) => "nuevo" | "repetido";
  cambiar: (productoId: string, cantidad: string) => void;
  quitar: (productoId: string) => void;
  vaciar: () => void;
};

const CarritoContexto = createContext<Carrito | null>(null);
const CLAVE = "feria.carrito.v1";

export function CarritoProvider({ children }: { children: React.ReactNode }) {
  const [items, setItems] = useState<ItemCarrito[]>([]);
  const [listo, setListo] = useState(false);

  useEffect(() => {
    const guardado = window.localStorage.getItem(CLAVE);
    if (guardado) {
      try {
        const parsed = JSON.parse(guardado) as ItemCarrito[];
        if (Array.isArray(parsed)) setItems(parsed);
      } catch {
        window.localStorage.removeItem(CLAVE);
      }
    }
    setListo(true);
  }, []);

  useEffect(() => {
    if (listo) window.localStorage.setItem(CLAVE, JSON.stringify(items));
  }, [items, listo]);

  const valor = useMemo<Carrito>(
    () => ({
      items,
      listo,
      agregar(producto) {
        if (items.some((item) => item.productoId === producto.id)) return "repetido";
        const entera = producto.unidad === "unidad" || producto.unidad === "atado";
        setItems([
          ...items,
          {
            productoId: producto.id,
            nombre: producto.nombre,
            unidad: producto.unidad,
            precioCentavos: producto.precio_centavos,
            stock: producto.stock,
            productor: producto.productor.nombre,
            cantidad: entera ? "1" : "0.500",
          },
        ]);
        return "nuevo";
      },
      cambiar(productoId, cantidad) {
        setItems((actual) =>
          actual.map((item) => (item.productoId === productoId ? { ...item, cantidad } : item)),
        );
      },
      quitar(productoId) {
        setItems((actual) => actual.filter((item) => item.productoId !== productoId));
      },
      vaciar() {
        setItems([]);
      },
    }),
    [items, listo],
  );

  return <CarritoContexto.Provider value={valor}>{children}</CarritoContexto.Provider>;
}

export function useCarrito() {
  const valor = useContext(CarritoContexto);
  if (!valor) throw new Error("El carrito solo se usa dentro de su proveedor.");
  return valor;
}
