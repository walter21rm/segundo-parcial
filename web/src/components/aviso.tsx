export function Aviso({ titulo, mensaje }: { titulo: string; mensaje: string }) {
  return (
    <div className="rounded-xl border border-dashed border-border bg-card px-6 py-12">
      <h1 className="font-display text-3xl">{titulo}</h1>
      <p className="mt-2 max-w-xl text-muted-foreground">{mensaje}</p>
    </div>
  );
}
