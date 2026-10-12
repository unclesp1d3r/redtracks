import { useState, type ReactNode } from "react";
import { useMutation } from "@tanstack/react-query";
import { PencilIcon, PlusIcon, Trash2Icon } from "lucide-react";
import { QueryStates } from "@/components/QueryStates";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { hostsOf, type Engagement, type Host } from "@/lib/api";
import { useEngagementQuery } from "@/lib/query";
import { AddRowDialog } from "./AddRowDialog";
import { ROW_API, type RowKind } from "./shared";

function DeleteButton({ kind, id, label }: { kind: RowKind; id: string; label: string }) {
  const [confirming, setConfirming] = useState(false);
  const del = useMutation({ mutationFn: () => ROW_API[kind].remove(id) });
  if (!confirming) {
    return (
      <Button variant="ghost" size="icon-sm" aria-label={`Delete ${label}`} onClick={() => setConfirming(true)}>
        <Trash2Icon />
      </Button>
    );
  }
  return (
    <span className="inline-flex gap-1">
      <Button variant="destructive" size="sm" disabled={del.isPending} onClick={() => del.mutate()}>
        Confirm delete
      </Button>
      <Button variant="ghost" size="sm" onClick={() => setConfirming(false)}>
        Cancel
      </Button>
    </span>
  );
}

function EditButton({ kind, row, hosts, label }: { kind: RowKind; row: object; hosts: Host[]; label: string }) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button variant="ghost" size="icon-sm" aria-label={`Edit ${label}`} onClick={() => setOpen(true)}>
        <PencilIcon />
      </Button>
      {/* Mounted only while open so the draft re-initialises from the latest row each time. */}
      {open && <AddRowDialog kind={kind} row={row} hosts={hosts} open onOpenChange={setOpen} />}
    </>
  );
}

interface Props<T extends { id: string }> {
  kind: RowKind;
  noun: string;
  select: (e: Engagement) => T[];
  headers: string[];
  /** Cells for one row, aligned with `headers`; the actions column is appended. */
  cells: (row: T, hosts: Host[]) => ReactNode[];
  /** Short text naming a row in aria-labels. */
  name: (row: T) => string;
}

export function RowTable<T extends { id: string }>({ kind, noun, select, headers, cells, name }: Props<T>) {
  const query = useEngagementQuery();
  const [adding, setAdding] = useState(false);
  const addButton = (
    <Button size="sm" onClick={() => setAdding(true)}>
      <PlusIcon /> Add {noun}
    </Button>
  );
  return (
    <QueryStates
      query={query}
      isEmpty={(e) => select(e).length === 0}
      empty={
        <div className="space-y-2 py-6 text-sm text-muted-foreground">
          <p>No {noun} rows yet.</p>
          {addButton}
          {adding && <AddRowDialog kind={kind} hosts={query.data ? hostsOf(query.data) : []} open onOpenChange={setAdding} />}
        </div>
      }
    >
      {(e) => {
        const hosts = hostsOf(e);
        return (
          <div className="space-y-2">
            {addButton}
            <Table>
              <TableHeader>
                <TableRow>
                  {headers.map((h) => (
                    <TableHead key={h}>{h}</TableHead>
                  ))}
                  <TableHead>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {select(e).map((row) => (
                  <TableRow key={row.id}>
                    {cells(row, hosts).map((c, i) => (
                      <TableCell key={headers[i]}>{c}</TableCell>
                    ))}
                    <TableCell className="whitespace-nowrap">
                      <EditButton kind={kind} row={row} hosts={hosts} label={name(row)} />
                      <DeleteButton kind={kind} id={row.id} label={name(row)} />
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            {adding && <AddRowDialog kind={kind} hosts={hosts} open onOpenChange={setAdding} />}
          </div>
        );
      }}
    </QueryStates>
  );
}
