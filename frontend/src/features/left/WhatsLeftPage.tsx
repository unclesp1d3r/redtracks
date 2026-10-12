import { Link } from "react-router";
import { QueryStates } from "@/components/QueryStates";
import { buttonVariants } from "@/components/ui/button";
import { hostsOf } from "@/lib/api";
import { useEngagementQuery, useSummariesQuery } from "@/lib/query";
import { HostLeftCard, hasNothingLeft, type UntriedCred } from "./HostLeftCard";

export function WhatsLeftPage() {
  const summaries = useSummariesQuery();
  const engagement = useEngagementQuery();

  const flagLabels = new Map((engagement.data?.Meta.flagTypes ?? []).map((f) => [f.id, f.label]));
  const hosts = engagement.data ? hostsOf(engagement.data) : [];
  const hostName = new Map(hosts.map((h) => [h.id, h.hostname || h.ip]));
  const creds = new Map<string, UntriedCred>(
    (engagement.data?.Creds.credentials ?? []).map((c) => [
      c.id,
      { id: c.id, username: c.username, foundOn: hostName.get(c.foundOnHostId) ?? "" },
    ]),
  );

  const empty = (
    <div className="flex flex-col items-start gap-3 py-10">
      <p className="text-sm text-muted-foreground">No hosts yet.</p>
      <Link to="/board" className={buttonVariants({ variant: "outline" })}>
        Go to Board
      </Link>
    </div>
  );

  return (
    <div className="space-y-4">
      <h1 className="text-xl font-semibold">What's left</h1>
      <QueryStates query={summaries} isEmpty={(s) => s.length === 0} empty={empty}>
        {(rows) => (
          <div className="grid gap-3 md:grid-cols-2">
            {/* Array.prototype.sort is stable, so board order survives within each group. */}
            {rows
              .slice().sort((a, b) => Number(hasNothingLeft(a)) - Number(hasNothingLeft(b)))
              .map((s) => (
                <HostLeftCard key={s.HostID} summary={s} flagLabels={flagLabels} creds={creds} />
              ))}
          </div>
        )}
      </QueryStates>
    </div>
  );
}
