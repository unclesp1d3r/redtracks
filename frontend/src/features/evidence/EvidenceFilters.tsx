import { SearchIcon } from "lucide-react";
import { Input } from "@/components/ui/input";
import type { EvidenceItemView } from "@/lib/api";

export type KindFilter = "" | "slot" | "finding" | "flag" | "accessPath" | "nothing";

export interface Filters {
  host: string;
  kind: KindFilter;
  /** yyyy-mm-dd, inclusive; empty means unbounded. */
  from: string;
  to: string;
}

export const EMPTY_FILTERS: Filters = { host: "", kind: "", from: "", to: "" };

const KIND_OPTIONS: { value: KindFilter; label: string }[] = [
  { value: "", label: "All" },
  { value: "slot", label: "Slot" },
  { value: "finding", label: "Finding" },
  { value: "flag", label: "Flag" },
  { value: "accessPath", label: "Access path" },
  { value: "nothing", label: "Nothing yet" },
];

/** Local calendar day of a capture time, as yyyy-mm-dd. */
function dayOf(iso: string): string {
  const d = new Date(iso);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

export function matchesFilters(ip: string, item: EvidenceItemView, f: Filters): boolean {
  if (f.host && f.host !== ip) return false;
  const live = (item.links ?? []).filter((l) => !l.removed);
  if (f.kind === "nothing" && live.length > 0) return false;
  if (f.kind && f.kind !== "nothing" && !live.some((l) => l.kind === f.kind)) return false;
  const day = dayOf(item.capturedAt);
  if (f.from && day < f.from) return false;
  if (f.to && day > f.to) return false;
  return true;
}

const FIELD = "h-8 rounded-lg border border-input bg-transparent px-2 text-sm dark:bg-input/30";

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="flex flex-col gap-1 text-xs text-muted-foreground">
      {label}
      {children}
    </label>
  );
}

interface Props {
  hosts: string[];
  value: Filters;
  onChange: (next: Filters) => void;
  search?: string;
  onSearchChange?: (q: string) => void;
}

export function EvidenceFilters({ hosts, value, onChange, search, onSearchChange }: Props) {
  const set = (patch: Partial<Filters>) => onChange({ ...value, ...patch });
  return (
    <div className="flex flex-wrap items-end gap-3">
      {onSearchChange && (
        <div className="relative min-w-56 flex-1">
          <SearchIcon className="pointer-events-none absolute left-2.5 top-2 size-4 text-muted-foreground" aria-hidden />
          <Input
            aria-label="Search evidence"
            placeholder="Search captions and text"
            className="pl-8"
            value={search ?? ""}
            onChange={(e) => onSearchChange(e.target.value)}
          />
        </div>
      )}
      <Field label="Host">
        <select className={FIELD} value={value.host} onChange={(e) => set({ host: e.target.value })}>
          <option value="">All hosts</option>
          {hosts.map((ip) => (
            <option key={ip} value={ip}>
              {ip}
            </option>
          ))}
        </select>
      </Field>
      <Field label="Target kind">
        <select className={FIELD} value={value.kind} onChange={(e) => set({ kind: e.target.value as KindFilter })}>
          {KIND_OPTIONS.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
      </Field>
      <Field label="Captured from">
        <input type="date" className={FIELD} value={value.from} onChange={(e) => set({ from: e.target.value })} />
      </Field>
      <Field label="Captured to">
        <input type="date" className={FIELD} value={value.to} onChange={(e) => set({ to: e.target.value })} />
      </Field>
    </div>
  );
}
