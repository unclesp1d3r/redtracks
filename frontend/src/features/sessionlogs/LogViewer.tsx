import { useEffect, useRef, useState, type FormEvent, type UIEvent } from "react";
import { QueryStates } from "@/components/QueryStates";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type { SessionLogSource } from "@/lib/api";
import { cn } from "@/lib/utils";
import { ClipDialog } from "./ClipDialog";
import { MAX_CLIP_LINES, lineCount, parseRange } from "./range";
import { SearchBox } from "./SearchBox";
import { usePage } from "./useSessionLog";

/** Lines fetched per window; the viewer never holds more than this in the DOM. */
export const WINDOW_LINES = 200;
const EDGE_PX = 8;

const clampStart = (line: number, total: number) => Math.max(0, Math.min(line, total - WINDOW_LINES));

export function LogViewer({ log }: { log: SessionLogSource }) {
  const total = log.Lines;
  const [start, setStart] = useState(0);
  const [focusLine, setFocusLine] = useState<number | null>(null);
  const [rangeText, setRangeText] = useState("");
  const [isAwaitingEnd, setIsAwaitingEnd] = useState(false);
  const [jumpText, setJumpText] = useState("");
  const [isClipOpen, setIsClipOpen] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);
  const lastTop = useRef(0);
  const settle = useRef<"top" | "bottom" | null>(null);

  const count = Math.min(WINDOW_LINES, total - start);
  const page = usePage(log.ID, start, count);
  const parsed = rangeText.trim() === "" ? null : parseRange(rangeText, total);

  useEffect(() => {
    const el = scrollRef.current;
    if (!el || page.isPlaceholderData) return;
    if (settle.current === "top") el.scrollTop = 0;
    if (settle.current === "bottom") el.scrollTop = el.scrollHeight;
    lastTop.current = el.scrollTop;
    settle.current = null;
    el.querySelector("[data-focused=true]")?.scrollIntoView?.({ block: "center" });
  }, [start, page.isPlaceholderData, focusLine]);

  const canNext = start + WINDOW_LINES < total;
  const canPrev = start > 0;
  function move(delta: number, edge: "top" | "bottom") {
    settle.current = edge;
    setStart(clampStart(start + delta, total));
  }
  function goTo(line: number) {
    setFocusLine(line);
    setStart(clampStart(line - WINDOW_LINES / 2, total));
  }

  function onScroll(e: UIEvent<HTMLDivElement>) {
    const el = e.currentTarget;
    const wasTop = lastTop.current;
    lastTop.current = el.scrollTop;
    if (page.isFetching) return;
    if (el.scrollTop > wasTop && el.scrollTop + el.clientHeight >= el.scrollHeight - EDGE_PX && canNext) {
      move(WINDOW_LINES, "top");
    } else if (el.scrollTop < wasTop && el.scrollTop <= EDGE_PX && canPrev) {
      move(-WINDOW_LINES, "bottom");
    }
  }

  function submitJump(e: FormEvent) {
    e.preventDefault();
    const n = Number(jumpText);
    if (Number.isInteger(n) && n >= 1 && n <= total) goTo(n - 1);
  }

  function pickLine(line: number, shift: boolean) {
    const anchor = parseRange(rangeText, total).range?.from;
    if ((shift || isAwaitingEnd) && anchor !== undefined) {
      const [a, b] = [Math.min(anchor, line), Math.max(anchor, line)];
      setRangeText(`${a + 1}-${b + 1}`);
      setIsAwaitingEnd(false);
      return;
    }
    setRangeText(`${line + 1}`);
    setIsAwaitingEnd(true);
  }

  const isSelected = (line: number) => parsed?.range !== undefined && line >= parsed.range.from && line <= parsed.range.to;

  return (
    <section role="region" aria-label={`Log ${log.Name}`} className="min-w-0 space-y-3">
      <div className="flex flex-wrap items-center gap-2">
        <Button size="sm" variant="outline" disabled={!canPrev} onClick={() => move(-WINDOW_LINES, "top")}>
          Previous
        </Button>
        <Button size="sm" variant="outline" disabled={!canNext} onClick={() => move(WINDOW_LINES, "top")}>
          Next
        </Button>
        <span className="text-xs text-muted-foreground">
          Lines {start + 1}-{start + count} of {total}
        </span>
        <form onSubmit={submitJump}>
          <Input
            aria-label="Jump to line"
            inputMode="numeric"
            placeholder="Go to line"
            className="w-28"
            value={jumpText}
            onChange={(e) => setJumpText(e.target.value)}
          />
        </form>
      </div>
      <SearchBox log={log} onPick={goTo} />
      <QueryStates query={page} isEmpty={(d) => d.length === 0} empty={<p className="text-sm text-muted-foreground">This log is empty.</p>}>
        {(lines) => (
          <div ref={scrollRef} onScroll={onScroll} className="max-h-96 overflow-auto rounded-md border font-mono text-xs">
            {lines.map((text, i) => {
              const line = start + i;
              const isFocused = line === focusLine;
              return (
                <div
                  key={line}
                  data-focused={isFocused}
                  data-selected={isSelected(line)}
                  className={cn("flex", isSelected(line) && "bg-primary/15", isFocused && "outline outline-1 outline-primary")}
                >
                  <button
                    type="button"
                    aria-label={`Select line ${line + 1}`}
                    onClick={(e) => pickLine(line, e.shiftKey)}
                    className="w-20 shrink-0 select-none px-2 text-right text-muted-foreground hover:text-foreground"
                  >
                    <span aria-hidden>{isFocused ? ">" : isSelected(line) ? "|" : " "}</span> {line + 1}
                  </button>
                  <span className="whitespace-pre-wrap break-all px-2">{text}</span>
                </div>
              );
            })}
          </div>
        )}
      </QueryStates>
      <div className="flex flex-wrap items-center gap-2">
        <Input
          aria-label="Clip range"
          placeholder="from-to, e.g. 10-49"
          className="w-44"
          value={rangeText}
          onChange={(e) => {
            setRangeText(e.target.value);
            setIsAwaitingEnd(false);
          }}
        />
        <Button size="sm" disabled={!parsed?.range} onClick={() => setIsClipOpen(true)}>
          Clip to evidence
        </Button>
        <span className="text-xs text-muted-foreground">Limit {MAX_CLIP_LINES.toLocaleString("en-US")} lines per clip.</span>
        <span className="text-xs text-muted-foreground" role="status">
          {parsed?.range
            ? `${lineCount(parsed.range)} lines selected`
            : (parsed?.error ?? "Select lines to clip")}
        </span>
      </div>
      {parsed?.range && (
        <ClipDialog log={log} from={parsed.range.from} to={parsed.range.to} open={isClipOpen} onOpenChange={setIsClipOpen} />
      )}
    </section>
  );
}
