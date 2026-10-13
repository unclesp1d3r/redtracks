import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeftIcon, TriangleAlertIcon } from "lucide-react";
import { NavLink, Outlet } from "react-router";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { TooltipProvider } from "@/components/ui/tooltip";
import { AddHostDialog } from "@/features/board/AddHostDialog";
import { useErrorBanner } from "@/lib/errorBanner";
import { EngagementService } from "@/lib/api";
import { keys, useEngagementQuery } from "@/lib/query";
import { cn } from "@/lib/utils";
import { CommandPalette } from "./CommandPalette";

const NAV = [
  { label: "Board", to: "/board" },
  { label: "Tables", to: "/tables" },
  { label: "Import", to: "/import" },
  { label: "What's left", to: "/left" },
];

export function AppShell() {
  const [addingHost, setAddingHost] = useState(false);
  const banner = useErrorBanner();
  const engagement = useEngagementQuery();
  const root = useQuery({ queryKey: keys.root(), queryFn: () => EngagementService.Root(), staleTime: Infinity });

  return (
    <TooltipProvider>
      <div className="grid h-screen grid-cols-[13rem_1fr] grid-rows-[1fr_auto]">
        <nav aria-label="Main" className="row-span-1 flex flex-col gap-1 border-r border-sidebar-border bg-sidebar p-3">
          <NavLink to="/" className="mb-3 flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground">
            <ArrowLeftIcon className="size-3" /> Engagements
          </NavLink>
          <p className="truncate text-lg font-semibold tracking-tight">{engagement.data?.Meta.name ?? "..."}</p>
          <p className="mb-4 text-xs tracking-wider text-muted-foreground uppercase">{engagement.data?.Meta.type}</p>
          {NAV.map((n) => (
            <NavLink
              key={n.to}
              to={n.to}
              className={({ isActive }) =>
                cn(
                  "rounded-md px-2 py-1.5 text-sm text-muted-foreground hover:bg-sidebar-accent hover:text-foreground",
                  isActive && "bg-sidebar-accent font-medium text-foreground",
                )
              }
            >
              {n.label}
            </NavLink>
          ))}
          <p className="mt-auto text-xs text-muted-foreground">
            Press <kbd className="rounded border px-1 font-mono">Ctrl/Cmd K</kbd>
          </p>
        </nav>
        <main className="overflow-auto p-6">
          {banner.length > 0 && (
            <Alert variant="destructive" className="mb-4">
              <TriangleAlertIcon />
              <AlertTitle>Could not read {banner.length === 1 ? "a file" : `${banner.length} files`} on disk</AlertTitle>
              <AlertDescription>
                <ul className="space-y-1 font-mono text-xs whitespace-pre-wrap">
                  {banner.map((e) => (
                    <li key={`${e.Kind}:${e.Path}`}>
                      {e.Path ? <strong>{e.Path}: </strong> : <strong>{e.Kind}: </strong>}
                      {e.Message}
                    </li>
                  ))}
                </ul>
              </AlertDescription>
            </Alert>
          )}
          <Outlet />
        </main>
        <footer className="col-span-2 border-t border-border px-3 py-1 font-mono text-xs text-muted-foreground">
          root: {root.data ?? "..."}
        </footer>
      </div>
      <CommandPalette onAddHost={() => setAddingHost(true)} />
      <AddHostDialog open={addingHost} onOpenChange={setAddingHost} />
    </TooltipProvider>
  );
}
