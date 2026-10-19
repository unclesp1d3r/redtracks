import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router";
import { QueryStates } from "@/components/QueryStates";
import { EvidenceService, hostPath } from "@/lib/api";
import type { HostEvidence } from "@/lib/api";
import { keys } from "@/lib/query";
import { EMPTY_FILTERS, EvidenceFilters, matchesFilters, type Filters } from "./EvidenceFilters";
import { EvidenceItemRow } from "./EvidenceItemRow";

const SEARCH_DEBOUNCE_MS = 300;

function useDebounced(value: string): string {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const t = setTimeout(() => setDebounced(value), SEARCH_DEBOUNCE_MS);
    return () => clearTimeout(t);
  }, [value]);
  return debounced;
}

const hitKey = (ip: string, id: string) => `${ip}\u0000${id}`;

/** Every host's evidence in one place: filter, search, and jump to the host. */
export function EvidencePage() {
  const all = useQuery({ queryKey: keys.evidenceAll(), queryFn: async () => (await EvidenceService.ListAll()) ?? [] });
  const [filters, setFilters] = useState<Filters>(EMPTY_FILTERS);
  const [search, setSearch] = useState("");
  const query = useDebounced(search.trim());
  const hits = useQuery({
    queryKey: [...keys.evidenceAll(), "search", query],
    queryFn: async () => (await EvidenceService.Search(query)) ?? [],
    enabled: query !== "",
  });
  const snippets = new Map((hits.data ?? []).map((h) => [hitKey(h.ip, h.itemId), h.snippet]));
  const isSearching = query !== "";
  // Filter by hits only once they have arrived; until then (or if the search fails) every item stays listed.
  const isSearchApplied = isSearching && hits.isSuccess;
  const isSearchPending = isSearching && hits.isPending;

  const hostsWithItems = (all.data ?? []).filter((h) => (h.View.items ?? []).length > 0);
  const visible = hostsWithItems
    .map((h) => ({
      ip: h.IP,
      items: (h.View.items ?? []).filter(
        (it) => matchesFilters(h.IP, it, filters) && (!isSearchApplied || snippets.has(hitKey(h.IP, it.id))),
      ),
    }))
    .filter((g) => g.items.length > 0);
  const total = hostsWithItems.reduce((n, h) => n + (h.View.items ?? []).length, 0);
  const shown = visible.reduce((n, g) => n + g.items.length, 0);

  const empty = (
    <p className="py-10 text-sm text-muted-foreground">
      No evidence yet. Capture from a host page, the inbox, or a session log.
    </p>
  );

  return (
    <div className="space-y-4" aria-busy={isSearchPending}>
      <h1 className="text-xl font-semibold">Evidence</h1>
      <QueryStates query={all} isEmpty={(d: HostEvidence[]) => !d.some((h) => (h.View.items ?? []).length > 0)} empty={empty}>
        {() => (
          <>
            <EvidenceFilters
              hosts={hostsWithItems.map((h) => h.IP)}
              value={filters}
              onChange={setFilters}
              search={search}
              onSearchChange={setSearch}
            />
            <p className="text-sm text-muted-foreground" role="status" aria-live="polite">
              {isSearchPending
                ? "Searching…"
                : shown === total
                  ? `${total} ${total === 1 ? "item" : "items"}`
                  : `${shown} of ${total} items`}
            </p>
            {isSearching && hits.isError && (
              <p role="alert" className="text-sm text-destructive">
                Search failed: {hits.error.message}
              </p>
            )}
            {!isSearchPending && visible.length === 0 && (
              <p className="py-6 text-sm text-muted-foreground">No evidence matches these filters.</p>
            )}
            {visible.map((g) => (
              <section key={g.ip} className="space-y-2">
                <h2 className="font-mono text-sm font-medium">
                  <Link to={hostPath(g.ip)} className="hover:underline">
                    {g.ip}
                  </Link>
                </h2>
                <ul className="space-y-2">
                  {g.items.flatMap((it) => [
                    <EvidenceItemRow key={it.id} ip={g.ip} item={it} />,
                    snippets.has(hitKey(g.ip, it.id)) && (
                      <li key={`${it.id}-snippet`} className="px-2 font-mono text-xs text-muted-foreground">
                        {snippets.get(hitKey(g.ip, it.id))}
                      </li>
                    ),
                  ])}
                </ul>
              </section>
            ))}
          </>
        )}
      </QueryStates>
    </div>
  );
}
