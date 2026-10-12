import { useState, type FormEvent } from "react";
import { useMutation } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { EngagementService, type Host } from "@/lib/api";
import { NONE } from "./shared";

export type RowKind = "credential" | "user" | "share" | "loot" | "finding";
type Draft = Record<string, string>;
type Option = { value: string; label: string };

interface Field {
  key: string;
  label: string;
  type: "text" | "textarea" | "host" | "service" | "options";
  options?: readonly string[];
}

const KINDS = ["password", "hash", "key", "token"] as const;
const SEVERITIES = ["info", "low", "medium", "high", "critical"] as const;
export const FINDING_STATUSES = ["noted", "confirmed", "exploited", "false-positive"] as const;

const FIELDS: Record<RowKind, Field[]> = {
  credential: [
    { key: "username", label: "Username", type: "text" },
    { key: "secret", label: "Secret", type: "text" },
    { key: "kind", label: "Kind", type: "options", options: KINDS },
    { key: "domain", label: "Domain", type: "text" },
    { key: "foundOnHostId", label: "Found on host", type: "host" },
    { key: "notes", label: "Notes", type: "textarea" },
  ],
  user: [
    { key: "name", label: "Name", type: "text" },
    { key: "domain", label: "Domain", type: "text" },
    { key: "hostId", label: "Host", type: "host" },
    { key: "notes", label: "Notes", type: "textarea" },
  ],
  share: [
    { key: "hostId", label: "Host", type: "host" },
    { key: "name", label: "Name", type: "text" },
    { key: "access", label: "Access", type: "text" },
    { key: "notes", label: "Notes", type: "textarea" },
  ],
  loot: [
    { key: "hostId", label: "Host", type: "host" },
    { key: "label", label: "Label", type: "text" },
    { key: "path", label: "Path", type: "text" },
    { key: "notes", label: "Notes", type: "textarea" },
  ],
  finding: [
    { key: "hostId", label: "Host", type: "host" },
    { key: "serviceId", label: "Service", type: "service" },
    { key: "title", label: "Title", type: "text" },
    { key: "severity", label: "Severity", type: "options", options: SEVERITIES },
    { key: "source", label: "Source", type: "text" },
    { key: "evidence", label: "Evidence", type: "textarea" },
    { key: "status", label: "Status", type: "options", options: FINDING_STATUSES },
  ],
};

const DEFAULTS: Record<RowKind, Draft> = {
  credential: { kind: "password" },
  user: {},
  share: {},
  loot: {},
  finding: { severity: "info", status: "noted" },
};

/** Required field per kind: a row without it is meaningless. */
const REQUIRED: Record<RowKind, string> = {
  credential: "username",
  user: "name",
  share: "name",
  loot: "label",
  finding: "title",
};

type Api = Record<string, (row: never) => Promise<void>>;
const api = EngagementService as unknown as Api;
const NAMES: Record<RowKind, string> = {
  credential: "Credential",
  user: "User",
  share: "Share",
  loot: "Loot",
  finding: "Finding",
};

function toDraft(kind: RowKind, row?: object): Draft {
  const base = { ...DEFAULTS[kind] };
  for (const f of FIELDS[kind]) {
    const v = (row as Record<string, unknown> | undefined)?.[f.key];
    if (typeof v === "string") base[f.key] = v;
  }
  return base;
}

function serviceOptions(hosts: Host[], hostId: string): Option[] {
  const svcs = hosts.find((h) => h.id === hostId)?.services ?? [];
  return [
    { value: NONE, label: "(none)" },
    ...svcs.map((s) => ({ value: s.id, label: `${s.port}/${s.protocol} ${s.type}`.trim() })),
  ];
}

function OptionSelect({ label, value, options, onChange }: {
  label: string;
  value: string;
  options: Option[];
  onChange: (v: string) => void;
}) {
  return (
    <Select items={options} value={value || NONE} onValueChange={(v) => onChange(v === NONE || v === null ? "" : v)}>
      <SelectTrigger aria-label={label} className="w-full">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {options.map((o) => (
          <SelectItem key={o.value} value={o.value}>
            {o.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

export function hostOptions(hosts: Host[], key: "id" | "ip" = "id"): Option[] {
  return [{ value: NONE, label: "(none)" }, ...hosts.map((h) => ({ value: h[key], label: h.ip }))];
}

interface Props {
  kind: RowKind;
  /** Existing row for edit mode; absent for add. */
  row?: object;
  hosts: Host[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function AddRowDialog({ kind, row, hosts, open, onOpenChange }: Props) {
  const [draft, setDraft] = useState<Draft>(() => toDraft(kind, row));
  const isEdit = row !== undefined;
  const set = (key: string, value: string) => setDraft((d) => ({ ...d, [key]: value }));

  const save = useMutation({
    mutationFn: () => {
      // Edit merges over the original so fields not in the form (worksOn, triedOn) survive.
      const payload = { ...(row ?? { id: "" }), ...draft };
      return api[`${isEdit ? "Update" : "Add"}${NAMES[kind]}`](payload as never);
    },
    onSuccess: () => onOpenChange(false),
  });

  const missing = !draft[REQUIRED[kind]]?.trim();
  function submit(e: FormEvent) {
    e.preventDefault();
    if (!missing) save.mutate();
  }

  function renderField(f: Field) {
    const value = draft[f.key] ?? "";
    if (f.type === "textarea") return <Textarea value={value} onChange={(e) => set(f.key, e.target.value)} />;
    if (f.type === "text") return <Input value={value} onChange={(e) => set(f.key, e.target.value)} />;
    if (f.type === "host") {
      // Changing host invalidates a service chosen for the previous host.
      const onChange = (v: string) => setDraft((d) => ({ ...d, [f.key]: v, serviceId: "" }));
      return <OptionSelect label={f.label} value={value} options={hostOptions(hosts)} onChange={onChange} />;
    }
    if (f.type === "service") {
      return <OptionSelect label={f.label} value={value} options={serviceOptions(hosts, draft.hostId ?? "")} onChange={(v) => set(f.key, v)} />;
    }
    const opts = (f.options ?? []).map((o) => ({ value: o, label: o }));
    return <OptionSelect label={f.label} value={value} options={opts} onChange={(v) => set(f.key, v)} />;
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <form onSubmit={submit} className="space-y-3">
          <DialogHeader>
            <DialogTitle>
              {isEdit ? "Edit" : "Add"} {NAMES[kind].toLowerCase()}
            </DialogTitle>
          </DialogHeader>
          {FIELDS[kind].map((f) => (
            <div key={f.key} className="space-y-1 text-sm">
              <span className="block">{f.label}</span>
              {renderField(f)}
            </div>
          ))}
          <DialogFooter>
            <Button type="submit" disabled={missing || save.isPending}>
              {isEdit ? "Save" : "Add"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
