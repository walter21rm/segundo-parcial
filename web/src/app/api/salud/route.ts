import { consultarSalud } from "@/lib/api";

export const dynamic = "force-dynamic";

export async function GET() {
  const salud = await consultarSalud();
  return Response.json({
    web: "ok",
    catalogo: salud.catalogo ? "ok" : "caido",
    pedidos: salud.pedidos ? "ok" : "caido",
  });
}
