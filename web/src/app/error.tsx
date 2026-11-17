"use client";

import { Button } from "@/components/ui/button";

export default function ErrorPagina({ reset }: { error: Error; reset: () => void }) {
  return (
    <div className="rounded-xl border border-dashed border-border bg-card px-6 py-12">
      <h1 className="font-display text-3xl">Esta página se tropezó</h1>
      <p className="mt-2 text-muted-foreground">Puedes reintentar. Si el servicio sigue caído, el aviso de arriba lo dice.</p>
      <Button className="mt-6" onClick={() => reset()}>
        Reintentar
      </Button>
    </div>
  );
}
