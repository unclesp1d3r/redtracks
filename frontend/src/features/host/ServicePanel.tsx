import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { QueryStates } from "@/components/QueryStates";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { HostService, list } from "@/lib/api";
import type { Finding, ItemStatus, Service } from "@/lib/api";
import { keys } from "@/lib/query";
import { Checklist } from "./Checklist";
import { FindingsList } from "./FindingsList";
import { WebPathsTable } from "./WebPathsTable";

interface PaneProps {
  ip: string;
  /** "" targets the host-level checklist. */
  serviceId: string;
  domain: string;
}

/** Loads and edits the checklists for one service (or the host when serviceId is ""). */
export function ChecklistPane({ ip, serviceId, domain }: PaneProps) {
  const qc = useQueryClient();
  const query = useQuery({
    queryKey: keys.checklists(ip, serviceId),
    queryFn: () => list(HostService.Checklists(ip, serviceId)),
  });
  const setItem = useMutation({
    mutationFn: (v: { itemId: string; status: ItemStatus; note: string }) =>
      HostService.SetItem(ip, serviceId, v.itemId, v.status, v.note),
    // onSettled also covers the error path; the global mutation onSuccess covers success.
    onSettled: () => qc.invalidateQueries({ queryKey: keys.checklists(ip, serviceId) }),
  });
  return (
    <QueryStates
      query={query}
      isEmpty={(views) => views.length === 0}
      empty={<p className="text-sm text-muted-foreground">No checklist matches this {serviceId ? "service" : "host"}.</p>}
    >
      {(views) => (
        <Checklist
          views={views}
          targetIp={ip}
          domain={domain}
          serviceId={serviceId}
          onSet={(itemId, status, note) => setItem.mutate({ itemId, status, note })}
        />
      )}
    </QueryStates>
  );
}

interface Props {
  ip: string;
  service: Service;
  findings: Finding[];
  domain: string;
}

export function ServicePanel({ ip, service, findings, domain }: Props) {
  const serviceFindings = findings.filter((f) => f.serviceId === service.id);
  return (
    <Tabs defaultValue="checklist">
      <TabsList>
        <TabsTrigger value="checklist">Checklist</TabsTrigger>
        <TabsTrigger value="web">Web paths</TabsTrigger>
        <TabsTrigger value="findings">Findings ({serviceFindings.length})</TabsTrigger>
      </TabsList>
      <TabsContent value="checklist">
        <ChecklistPane ip={ip} serviceId={service.id} domain={domain} />
      </TabsContent>
      <TabsContent value="web">
        <WebPathsTable paths={service.webPaths ?? []} />
      </TabsContent>
      <TabsContent value="findings">
        <FindingsList ip={ip} findings={serviceFindings} />
      </TabsContent>
    </Tabs>
  );
}
