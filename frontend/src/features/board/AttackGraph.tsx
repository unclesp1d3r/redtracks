import { useQuery } from "@tanstack/react-query";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Mermaid } from "@/components/Mermaid";
import { QueryStates } from "@/components/QueryStates";
import { EngagementService } from "@/lib/api";
import { keys } from "@/lib/query";

export function AttackGraph() {
  const query = useQuery({ queryKey: keys.graph(), queryFn: () => EngagementService.AttackGraph() });
  return (
    <Card size="sm">
      <CardHeader>
        <CardTitle className="text-xs font-semibold tracking-wider text-muted-foreground uppercase">
          Attack graph
        </CardTitle>
      </CardHeader>
      <CardContent>
        <QueryStates
          query={query}
          isEmpty={(g) => g.trim() === ""}
          empty={<p className="text-sm text-muted-foreground">No access paths recorded yet.</p>}
        >
          {(graph) => <Mermaid text={graph} />}
        </QueryStates>
      </CardContent>
    </Card>
  );
}
