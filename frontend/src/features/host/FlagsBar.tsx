import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { CheckIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { EvidenceLinkKind, HostService } from "@/lib/api";
import type { Flag, FlagType, Host } from "@/lib/api";
import { ProofStrip } from "@/features/evidence/ProofStrip";
import { plainLink } from "@/features/evidence/proof";
import { useTargetRow } from "@/features/evidence/usePaste";
import { cn } from "@/lib/utils";

export function FlagsBar({ host, flagTypes }: { host: Host; flagTypes: FlagType[] }) {
  return (
    <div className="flex flex-wrap gap-2" aria-label="Flags">
      {flagTypes.map((ft) => (
        <FlagChip key={ft.id} ip={host.ip} flagType={ft} flag={(host.flags ?? []).find((f) => f.typeId === ft.id)} />
      ))}
    </div>
  );
}

interface ChipProps {
  ip: string;
  flagType: FlagType;
  flag: Flag | undefined;
}

function FlagChip({ ip, flagType, flag }: ChipProps) {
  const link = plainLink(EvidenceLinkKind.EvidenceLinkFlag, flagType.id);
  const name = `flag ${flagType.label}`;
  const rowProps = useTargetRow(ip, link, name);
  return (
    <div {...rowProps} className={cn(rowProps.className, "flex flex-wrap items-center gap-1.5 rounded-md p-0.5")}>
      <FlagValueChip ip={ip} flagType={flagType} flag={flag} />
      <ProofStrip ip={ip} link={link} name={name} words={{ yes: "proven", no: "unproven" }} />
    </div>
  );
}

function FlagValueChip({ ip, flagType, flag }: ChipProps) {
  const [open, setOpen] = useState(false);
  const [value, setValue] = useState(flag?.value ?? "");
  const [submitted, setSubmitted] = useState(flag?.submitted ?? false);
  const isCaptured = (flag?.value ?? "") !== "";

  const save = useMutation({
    mutationFn: (f: Flag) => HostService.SetFlag(ip, f),
    onSuccess: () => setOpen(false),
  });
  const send = (v: string, s: boolean) =>
    save.mutate({
      typeId: flagType.id,
      value: v,
      submitted: s,
      capturedAt: v === "" ? "" : (flag?.capturedAt || new Date().toISOString()),
    });

  return (
    <Popover
      open={open}
      onOpenChange={(o) => {
        setOpen(o);
        if (o) {
          setValue(flag?.value ?? "");
          setSubmitted(flag?.submitted ?? false);
        }
      }}
    >
      <PopoverTrigger
        className={cn(
          "inline-flex h-6 items-center gap-1 rounded-full border px-2.5 text-xs font-medium",
          isCaptured ? "bg-secondary text-secondary-foreground" : "text-muted-foreground",
        )}
      >
        {isCaptured && <CheckIcon className="size-3" aria-hidden />}
        {flagType.label}
        {isCaptured && flag?.submitted && <span className="text-[10px]">(submitted)</span>}
      </PopoverTrigger>
      <PopoverContent align="start">
        <label className="space-y-1 text-sm">
          <span>{flagType.label} value</span>
          <Input value={value} onChange={(e) => setValue(e.target.value)} autoFocus />
        </label>
        <label className="flex items-center gap-2 text-sm">
          <Checkbox checked={submitted} onCheckedChange={(c) => setSubmitted(c === true)} />
          Submitted
        </label>
        <div className="flex justify-between">
          <Button size="sm" variant="ghost" disabled={!isCaptured} onClick={() => send("", false)}>
            Clear
          </Button>
          <Button size="sm" onClick={() => send(value.trim(), submitted)}>
            Save
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  );
}
