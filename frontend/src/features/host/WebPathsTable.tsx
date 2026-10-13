import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import type { WebPath } from "@/lib/api";

export function WebPathsTable({ paths }: { paths: WebPath[] }) {
  if (paths.length === 0) {
    return <p className="text-sm text-muted-foreground">No web paths discovered. Import ffuf, feroxbuster, or gobuster output.</p>;
  }
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Path</TableHead>
          <TableHead>Status</TableHead>
          <TableHead>Size</TableHead>
          <TableHead>Source</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {paths.map((p) => (
          <TableRow key={`${p.source}:${p.path}`}>
            <TableCell className="font-mono text-xs">{p.path}</TableCell>
            <TableCell className="tabular-nums">{p.status}</TableCell>
            <TableCell className="tabular-nums">{p.size}</TableCell>
            <TableCell>{p.source}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
