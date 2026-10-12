import { TriangleAlertIcon } from "lucide-react";
import { useChecklistStatusQuery } from "@/lib/query";

/** How many checklist definitions loaded, plus any load or install error. */
export function ChecklistStatus() {
  const { data, isError, error } = useChecklistStatusQuery();
  if (isError) return <p role="alert" className="text-xs text-destructive">Checklist status unavailable: {error.message}</p>;
  if (!data) return null;
  const problems = [data.InstallError && `Install: ${data.InstallError}`, data.Error && `Load: ${data.Error}`].filter(Boolean);
  return (
    <div className="space-y-1 text-xs text-muted-foreground">
      <p>Checklist definitions: {data.Count} loaded</p>
      {problems.map((p) => (
        <p key={p} role="alert" className="flex items-start gap-1 text-destructive">
          <TriangleAlertIcon className="mt-0.5 size-3 shrink-0" aria-hidden />
          <span>{p}</span>
        </p>
      ))}
    </div>
  );
}
