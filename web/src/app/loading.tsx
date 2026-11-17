import { Skeleton } from "@/components/ui/skeleton";

export default function Cargando() {
  return (
    <div className="space-y-4">
      <Skeleton className="h-12 w-2/3" />
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <Skeleton className="h-52" />
        <Skeleton className="h-52" />
        <Skeleton className="h-52" />
      </div>
    </div>
  );
}
