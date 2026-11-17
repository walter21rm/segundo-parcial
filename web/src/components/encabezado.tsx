"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

import { useCarrito } from "@/components/carrito";
import { cn } from "cn";

const enlaces = [
  { href: "/", nombre: "Puesto" },
  { href: "/productores", nombre: "Productores" },
  { href: "/pedidos", nombre: "Pedidos" },
];

export function Encabezado({ salud }: { salud: { catalogo: boolean; pedidos: boolean } }) {
  const pathname = usePathname();
  const { items } = useCarrito();
  const ambos = salud.catalogo && salud.pedidos;

  return (
    <header className="sticky top-0 z-20 border-b border-border/80 bg-[#f3ecdf]/90 backdrop-blur">
      <div className="mx-auto flex max-w-6xl items-center gap-4 px-4 py-3">
        <Link href="/" className="flex items-center gap-3">
          <span className="grid size-9 place-items-center rounded-md bg-primary font-display text-lg text-primary-foreground">
            F
          </span>
          <span className="leading-tight">
            <span className="block font-display text-lg tracking-tight">Feria del Productor</span>
            <span className="block text-xs text-muted-foreground">Lima, puestos de la mañana</span>
          </span>
        </Link>
        <nav className="ml-auto hidden items-center gap-1 sm:flex">
          {enlaces.map((enlace) => (
            <Link
              key={enlace.href}
              href={enlace.href}
              className={cn(
                "rounded-md px-3 py-2 text-sm",
                pathname === enlace.href ? "bg-secondary text-foreground" : "text-muted-foreground hover:text-foreground",
              )}
            >
              {enlace.nombre}
            </Link>
          ))}
        </nav>
        <Link
          href="/pedidos/nuevo"
          className="rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground"
        >
          Pedido{items.length > 0 ? ` · ${items.length}` : ""}
        </Link>
      </div>
      <div className="mx-auto flex max-w-6xl gap-4 overflow-x-auto px-4 pb-3 text-xs sm:hidden">
        {enlaces.map((enlace) => (
          <Link key={enlace.href} href={enlace.href} className="text-muted-foreground">
            {enlace.nombre}
          </Link>
        ))}
      </div>
      {!ambos && (
        <p className="bg-primary px-4 py-2 text-center text-sm text-primary-foreground">
          {!salud.catalogo && !salud.pedidos
            ? "Catálogo y pedidos no responden. Revisa que los dos servicios estén en marcha."
            : !salud.catalogo
              ? "El catálogo no responde. Puedes ver pedidos ya hechos, no armar uno nuevo."
              : "Pedidos no responde. El puesto sigue visible, pero no se pueden registrar compras."}
        </p>
      )}
    </header>
  );
}
