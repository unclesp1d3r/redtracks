import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { Unresolved } from "@/lib/api";

export interface Choice {
  ip: string;
  port: string;
}
export type Choices = Record<string, Choice>;

const MAX_PORT = 65535;

function validPort(port: string): boolean {
  const n = Number(port);
  return Number.isInteger(n) && n >= 1 && n <= MAX_PORT;
}

/** An entry is resolved once a host is picked and any typed port is valid. */
export function isResolved(c: Choice | undefined): boolean {
  return !!c?.ip && (c.port === "" || validPort(c.port));
}

/** Builds the Apply overrides map: `ip` or `ip:port`. */
export function toOverrides(choices: Choices): Record<string, string> {
  const out: Record<string, string> = {};
  for (const [key, c] of Object.entries(choices)) {
    if (isResolved(c)) out[key] = c.port ? `${c.ip}:${c.port}` : c.ip;
  }
  return out;
}

interface Props {
  unresolved: Unresolved[];
  hostIps: string[];
  choices: Choices;
  onChange: (choices: Choices) => void;
}

export function ResolveHosts({ unresolved, hostIps, choices, onChange }: Props) {
  const set = (key: string, patch: Partial<Choice>) =>
    onChange({ ...choices, [key]: { ...(choices[key] ?? { ip: "", port: "" }), ...patch } });

  return (
    <ul className="space-y-2">
      {unresolved.map((u) => {
        const c = choices[u.Key];
        const bad = !!c?.port && !validPort(c.port);
        return (
          <li key={u.Key} className="flex flex-wrap items-center gap-2 rounded-md border px-3 py-2 text-sm">
            <span className="min-w-40 font-mono">{u.URL || u.Hostname}</span>
            <Select value={c?.ip ?? ""} onValueChange={(v) => set(u.Key, { ip: v ?? "" })}>
              <SelectTrigger aria-label={`Host for ${u.Key}`} className="w-48">
                <SelectValue placeholder="Choose host" />
              </SelectTrigger>
              <SelectContent>
                {hostIps.map((ip) => (
                  <SelectItem key={ip} value={ip}>
                    {ip}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {!u.URL && (
              <Input
                aria-label={`Port for ${u.Key}`}
                aria-invalid={bad}
                className="w-24"
                inputMode="numeric"
                placeholder="Port"
                value={c?.port ?? ""}
                onChange={(e) => set(u.Key, { port: e.target.value.trim() })}
              />
            )}
            {bad && <span className="text-xs text-destructive">Port must be 1-{MAX_PORT}</span>}
          </li>
        );
      })}
    </ul>
  );
}
