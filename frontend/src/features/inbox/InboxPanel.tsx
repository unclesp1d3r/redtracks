import { useQuery } from "@tanstack/react-query";
import { QueryStates } from "@/components/QueryStates";
import { EngagementService } from "@/lib/api";
import { keys } from "@/lib/query";
import { InboxRow } from "./InboxRow";
import { useInboxCount, useInboxQuery } from "./useInbox";

/** Nav badge: nothing at zero. */
export function InboxCountBadge() {
  const count = useInboxCount();
  if (count === 0) return null;
  return (
    <span
      aria-label={`${count} ${count === 1 ? "file" : "files"} in inbox`}
      className="rounded-full bg-primary px-1.5 text-xs text-primary-foreground"
    >
      {count}
    </span>
  );
}

export function InboxPanel() {
  const query = useInboxQuery();
  const root = useQuery({ queryKey: keys.root(), queryFn: () => EngagementService.Root(), staleTime: Infinity });
  const folder = `${root.data ?? "<engagement>"}/inbox`;

  return (
    <section aria-labelledby="inbox-heading" className="space-y-3">
      <h1 id="inbox-heading" className="text-lg font-semibold">
        Inbox
      </h1>
      <p className="text-sm text-muted-foreground">
        Files dropped into <code className="font-mono">{folder}</code> appear here, ready to assign to a host.
      </p>
      <QueryStates
        query={query}
        isEmpty={(d) => (d.Files ?? []).length === 0}
        empty={
          <p className="text-sm text-muted-foreground">
            Nothing waiting. Drop screenshots into {folder} and they appear here.
          </p>
        }
      >
        {(d) => (
          <ul className="space-y-2">
            {(d.Files ?? []).map((f) => (
              <InboxRow key={f.Name} file={f} token={d.Token} />
            ))}
          </ul>
        )}
      </QueryStates>
    </section>
  );
}
