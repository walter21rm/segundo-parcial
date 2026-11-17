import type { Metadata } from "next";
import { Fraunces, Outfit } from "next/font/google";
import { ThemeProvider } from "next-themes";

import { CarritoProvider } from "@/components/carrito";
import { Encabezado } from "@/components/encabezado";
import { Toaster } from "@/components/ui/sonner";
import { consultarSalud } from "@/lib/api";

import "./globals.css";

const outfit = Outfit({ subsets: ["latin"], variable: "--font-outfit" });
const fraunces = Fraunces({ subsets: ["latin"], variable: "--font-fraunces" });

export const metadata: Metadata = {
  title: "Feria del Productor",
  description: "Puesto de la mañana en Lima: productos de productores locales y pedidos con stock reservado.",
};

export const dynamic = "force-dynamic";

export default async function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  const salud = await consultarSalud();
  return (
    <html lang="es" className={`${outfit.variable} ${fraunces.variable}`}>
      <body className="min-h-screen antialiased">
        <ThemeProvider attribute="class" forcedTheme="light" enableSystem={false}>
          <CarritoProvider>
            <Encabezado salud={salud} />
            <main className="mx-auto max-w-6xl px-4 py-8">{children}</main>
            <Toaster />
          </CarritoProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
