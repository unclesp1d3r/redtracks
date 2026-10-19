import { useState, type FormEvent } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type { SessionLogSource } from "@/lib/api";
import { SEARCH_LIMIT, useLogSearch } from "./useSessionLog";

export function SearchBox({ log, onPick }: { log: SessionLogSource; onPick: (line: number) => void }) {
  const [text, setText] = useState("");
  const [submitted, setSubmitted] = useState("");
  const search = useLogSearch(log.ID, submitted);

  function submit(e: FormEvent) {
    e.preventDefault();
    setSubmitted(text.trim());
  }
  const matches = search.data ?? [];

  return (
    <div className="space-y-2">
      <form onSubmit={submit} className="flex gap-2">
        <Input aria-label="Search in log" value={text} onChange={(e) => setText(e.target.value)} placeholder="Search" />
        <Button type="submit" variant="outline" disabled={search.isFetching}>
          {search.isFetching ? "Searching..." : "Search"}
        </Button>
      </form>
      {search.isError && (
        <p role="alert" className="text-sm text-destructive">
          {search.error.message}
        </p>
      )}
      {search.isSuccess && matches.length === 0 && <p className="text-sm text-muted-foreground">No matches</p>}
      {matches.length > 0 && (
        <ul aria-label="Search results" className="max-h-40 space-y-1 overflow-auto font-mono text-xs">
          {matches.map((m) => (
            <li key={m.Line}>
              <button
                type="button"
                aria-label={`Line ${m.Line + 1}`}
                onClick={() => onPick(m.Line)}
                className="w-full truncate rounded px-1 text-left hover:bg-muted"
              >
                line {m.Line + 1}: {m.Snippet}
              </button>
            </li>
          ))}
          {matches.length >= SEARCH_LIMIT && (
            <li className="text-muted-foreground">More matches not shown; narrow the search.</li>
          )}
        </ul>
      )}
    </div>
  );
}
