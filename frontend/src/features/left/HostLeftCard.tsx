import { TriangleAlertIcon } from "lucide-react";
import { Link } from "react-router";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { HostSummary } from "@/lib/api";

export interface UntriedCred {
  id: string;
  username: string;
  /** IP or hostname of the host the credential was found on; empty when unknown. */
  foundOn: string;
}

interface Props {
  summary: HostSummary;
  /** flag type id -> label */
  flagLabels: ReadonlyMap<string, string>;
  /** credential id -> display data */
  creds: ReadonlyMap<string, UntriedCred>;
}

export function hasNothingLeft(s: HostSummary): boolean {
  return (
    (s.MissingFlags ?? []).length === 0 &&
    (s.UntouchedServices ?? []).length === 0 &&
    s.OpenItems === 0 &&
    (s.UntriedCreds ?? []).length === 0 &&
    !s.NoAccessPath
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="space-y-1">
      <h3 className="text-xs font-medium uppercase tracking-wide text-muted-foreground">{title}</h3>
      <ul className="flex flex-wrap gap-1">{children}</ul>
    </section>
  );
}

export function HostLeftCard({ summary, flagLabels, creds }: Props) {
  const href = `/host/${encodeURIComponent(summary.IP)}`;
  const missing = summary.MissingFlags ?? [];
  const untouched = summary.UntouchedServices ?? [];
  const untried = summary.UntriedCreds ?? [];
  const chip = (key: string, label: string) => (
    <li key={key}>
      <Link to={href}>
        <Badge variant="outline">{label}</Badge>
      </Link>
    </li>
  );
  return (
    <Card size="sm">
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Link to={href} className="font-mono hover:underline">
            {summary.IP}
          </Link>
          {summary.Hostname && <span className="font-normal text-muted-foreground">{summary.Hostname}</span>}
          <Badge variant="secondary">{summary.Status}</Badge>
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        {hasNothingLeft(summary) && <p className="text-sm text-muted-foreground">Nothing left</p>}
        {summary.NoAccessPath && (
          <Link to={href} className="inline-flex items-center gap-1 text-sm text-accent hover:underline">
            <TriangleAlertIcon className="size-4" aria-hidden />
            No access path
          </Link>
        )}
        {missing.length > 0 && (
          <Section title="Missing flags">{missing.map((id) => chip(id, flagLabels.get(id) ?? id))}</Section>
        )}
        {untouched.length > 0 && (
          <Section title="Untouched services">{untouched.map((s) => chip(s, s))}</Section>
        )}
        {summary.OpenItems > 0 && (
          <Section title="Open items">
            <li>
              <Link to={href} className="text-sm hover:underline">
                {summary.OpenItems} open checklist {summary.OpenItems === 1 ? "item" : "items"}
              </Link>
            </li>
          </Section>
        )}
        {untried.length > 0 && (
          <Section title="Untried credentials">
            {untried.map((id) => {
              const c = creds.get(id);
              const label = c ? (c.foundOn ? `${c.username} (found on ${c.foundOn})` : c.username) : id;
              return chip(id, label);
            })}
          </Section>
        )}
      </CardContent>
    </Card>
  );
}
