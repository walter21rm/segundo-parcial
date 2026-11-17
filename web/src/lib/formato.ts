export const CATEGORIAS = [
  { id: "frutas", nombre: "Frutas" },
  { id: "verduras", nombre: "Verduras" },
  { id: "lacteos", nombre: "Lácteos" },
  { id: "granos", nombre: "Granos" },
  { id: "miel", nombre: "Miel" },
  { id: "otros", nombre: "Otros" },
] as const;

export const DISTRITOS = [
  "Barranco",
  "Chorrillos",
  "La Molina",
  "Lurín",
  "Magdalena del Mar",
  "Miraflores",
  "Pachacámac",
  "Pueblo Libre",
  "San Borja",
  "San Isidro",
  "Santiago de Surco",
  "Surquillo",
];

const ENTERAS = new Set(["unidad", "atado"]);

export function soles(centavos: number) {
  return `S/ ${(centavos / 100).toLocaleString("es-PE", {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })}`;
}

export function cantidadVisible(valor: string, unidad: string) {
  const numero = Number(valor);
  if (Number.isNaN(numero)) return valor;
  if (ENTERAS.has(unidad)) return String(Math.round(numero));
  return numero.toLocaleString("es-PE", { maximumFractionDigits: 3 });
}

export function unidadVisible(unidad: string) {
  if (unidad === "unidad") return "u.";
  return unidad;
}

export function etiquetaEstado(estado: string) {
  const mapa: Record<string, string> = {
    pendiente: "Pendiente",
    confirmado: "Confirmado",
    cancelado: "Cancelado",
    reservando: "Reservando stock",
    compensacion_incompleta: "Devolución incompleta",
  };
  return mapa[estado] ?? estado;
}
