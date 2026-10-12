import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router";
import {
  Command,
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { EngagementService } from "@/lib/api";
import { keys } from "@/lib/query";

export const SCREENS = [
  { label: "Board", to: "/board" },
  { label: "Tables", to: "/tables" },
  { label: "Import", to: "/import" },
  { label: "What's left", to: "/left" },
  { label: "Engagements", to: "/" },
] as const;

export function CommandPalette({ onAddHost }: { onAddHost: () => void }) {
  const [open, setOpen] = useState(false);
  const navigate = useNavigate();
  const engagement = useQuery({ queryKey: keys.engagement(), queryFn: () => EngagementService.Current() });

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key.toLowerCase() === "k" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setOpen((o) => !o);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  const go = (to: string) => {
    setOpen(false);
    navigate(to);
  };
  const hosts = engagement.data?.Hosts ?? [];

  return (
    <CommandDialog open={open} onOpenChange={setOpen}>
      <Command>
        <CommandInput placeholder="Jump to a host, service, or screen" />
        <CommandList>
          <CommandEmpty>No matches.</CommandEmpty>
          <CommandGroup heading="Actions">
            <CommandItem
              onSelect={() => {
                setOpen(false);
                onAddHost();
              }}
            >
              Add host
            </CommandItem>
            {SCREENS.map((s) => (
              <CommandItem key={s.to} onSelect={() => go(s.to)}>
                Go to {s.label}
              </CommandItem>
            ))}
          </CommandGroup>
          <CommandGroup heading="Hosts">
            {hosts.map((h) => (
              <CommandItem
                key={h.id}
                value={`${h.ip} ${h.hostname}`}
                onSelect={() => go(`/host/${encodeURIComponent(h.ip)}`)}
              >
                <span className="font-mono">{h.ip}</span>
                <span className="text-muted-foreground">{h.hostname}</span>
              </CommandItem>
            ))}
          </CommandGroup>
          <CommandGroup heading="Services">
            {hosts.flatMap((h) =>
              (h.services ?? []).map((s) => (
                <CommandItem
                  key={`${h.id}:${s.id}`}
                  value={`${h.ip}:${s.port} ${s.type}`}
                  onSelect={() => go(`/host/${encodeURIComponent(h.ip)}`)}
                >
                  <span className="font-mono">
                    {h.ip}:{s.port}
                  </span>
                  <span className="text-muted-foreground">{s.type}</span>
                </CommandItem>
              )),
            )}
          </CommandGroup>
        </CommandList>
      </Command>
    </CommandDialog>
  );
}
