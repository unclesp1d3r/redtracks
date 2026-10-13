import { useState } from "react";
import { useParams } from "react-router";
import { QueryStates } from "@/components/QueryStates";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { hostsOf } from "@/lib/api";
import type { Engagement, Host } from "@/lib/api";
import { useEngagementQuery } from "@/lib/query";
import { AccessPaths } from "./AccessPaths";
import { ChecklistStatus } from "./ChecklistStatus";
import { AddServiceDialog } from "./AddServiceDialog";
import { HostHeader } from "./HostHeader";
import { NotesEditor } from "./NotesEditor";
import { ChecklistPane, ServicePanel } from "./ServicePanel";
import { ServiceList } from "./ServiceList";

/** Domain for command substitution: hostname after the first dot, else the engagement domain. */
export function domainFor(host: Host, engagementDomain: string): string {
  const dot = host.hostname.indexOf(".");
  return dot >= 0 ? host.hostname.slice(dot + 1) : engagementDomain;
}

export function HostPage() {
  const { ip = "" } = useParams();
  const query = useEngagementQuery();
  const notFound = <p className="text-sm text-muted-foreground">No host with IP {ip} in this engagement.</p>;
  return (
    // A missing host is decided inside children so the host is looked up once.
    <QueryStates query={query} isEmpty={() => false} empty={notFound}>
      {(eng) => {
        const host = hostsOf(eng).find((h) => h.ip === ip);
        return host ? <HostView eng={eng} host={host} /> : notFound;
      }}
    </QueryStates>
  );
}

function HostView({ eng, host }: { eng: Engagement; host: Host }) {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [isAdding, setIsAdding] = useState(false);
  const services = host.services ?? [];
  const selected = services.find((s) => s.id === selectedId) ?? services[0];
  const domain = domainFor(host, eng.Meta.domain);

  return (
    <div className="space-y-4">
      <HostHeader host={host} meta={eng.Meta} />
      <AccessPaths
        host={host}
        otherHosts={hostsOf(eng).filter((h) => h.ip !== host.ip)}
        credentials={eng.Creds.credentials ?? []}
      />
      <ChecklistStatus />
      <Tabs defaultValue="services">
        <TabsList>
          <TabsTrigger value="services">Services</TabsTrigger>
          <TabsTrigger value="host">Host checklists</TabsTrigger>
        </TabsList>
        <TabsContent value="services">
          <div className="grid grid-cols-[14rem_1fr] gap-4">
            <ServiceList
              services={services}
              selectedId={selected?.id ?? null}
              onSelect={setSelectedId}
              onAdd={() => setIsAdding(true)}
            />
            {selected ? (
              <ServicePanel ip={host.ip} service={selected} findings={eng.Findings ?? []} domain={domain} />
            ) : (
              <p className="text-sm text-muted-foreground">Add a service to see its checklist.</p>
            )}
          </div>
        </TabsContent>
        <TabsContent value="host">
          <ChecklistPane ip={host.ip} serviceId="" domain={domain} />
        </TabsContent>
      </Tabs>
      <NotesEditor ip={host.ip} />
      <AddServiceDialog ip={host.ip} open={isAdding} onOpenChange={setIsAdding} />
    </div>
  );
}
