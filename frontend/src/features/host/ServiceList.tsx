import { PlusIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import type { Service } from "@/lib/api";
import { cn } from "@/lib/utils";

interface Props {
  services: Service[];
  selectedId: string | null;
  onSelect: (id: string) => void;
  onAdd: () => void;
}

export function ServiceList({ services, selectedId, onSelect, onAdd }: Props) {
  return (
    <nav aria-label="Services" className="space-y-1">
      {services.length === 0 && <p className="text-sm text-muted-foreground">No services yet.</p>}
      {services.map((s) => {
        const isSelected = s.id === selectedId;
        return (
          <button
            key={s.id}
            type="button"
            aria-current={isSelected ? "true" : undefined}
            onClick={() => onSelect(s.id)}
            className={cn(
              "flex w-full items-center gap-2 rounded-md border px-2 py-1 text-left text-sm",
              isSelected ? "border-primary bg-primary/10 font-medium" : "hover:bg-muted",
            )}
          >
            <span className="font-mono tabular-nums">
              {s.port}/{s.protocol}
            </span>
            <span className="text-muted-foreground">{s.type}</span>
          </button>
        );
      })}
      <Button size="sm" variant="outline" onClick={onAdd}>
        <PlusIcon /> Add service
      </Button>
    </nav>
  );
}
