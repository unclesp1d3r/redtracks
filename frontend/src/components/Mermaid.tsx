import { useEffect, useId, useState } from "react";
import DOMPurify from "dompurify";
import { DiagnosticsService } from "@/lib/api";

type MermaidApi = (typeof import("mermaid"))["default"];

// Loaded on first render so the large mermaid bundle stays out of the initial chunk.
let mermaidReady: Promise<MermaidApi> | undefined;

function loadMermaid(): Promise<MermaidApi> {
  mermaidReady ??= import("mermaid").then(({ default: mermaid }) => {
    // DOMPurify's svg profile drops <foreignObject>, which is where Mermaid's default HTML
    // labels live, so force SVG <text> labels or every node renders blank.
    mermaid.initialize({
      startOnLoad: false,
      securityLevel: "strict",
      theme: "dark",
      htmlLabels: false,
      flowchart: { htmlLabels: false },
    });
    return mermaid;
  });
  return mermaidReady;
}

type Rendered = { svg: string } | { error: string };

export function Mermaid({ text }: { text: string }) {
  const id = "mmd" + useId().replace(/:/g, "");
  const [result, setResult] = useState<Rendered | null>(null);

  useEffect(() => {
    let cancelled = false;
    loadMermaid()
      .then((mermaid) => mermaid.render(id, text))
      .then(({ svg }) => {
        if (cancelled) return;
        const clean = DOMPurify.sanitize(svg, { USE_PROFILES: { svg: true, svgFilters: true } });
        setResult({ svg: clean });
      })
      .catch((e: unknown) => {
        if (cancelled) return;
        const message = e instanceof Error ? e.message : String(e);
        setResult({ error: message });
        DiagnosticsService.LogError("Mermaid", message).catch((err: unknown) => console.error("LogError failed:", err));
      });
    return () => {
      cancelled = true;
    };
  }, [id, text]);

  if (result && "error" in result) {
    return (
      <div>
        <p className="mb-1 text-xs text-muted-foreground">Diagram failed to render.</p>
        <p role="alert" className="mb-1 font-mono text-xs whitespace-pre-wrap text-destructive">
          {result.error}
        </p>
        <pre className="overflow-auto rounded-md bg-muted p-2 text-xs">{text}</pre>
      </div>
    );
  }
  if (!result) return null;
  return <div className="overflow-auto [&_svg]:mx-auto" dangerouslySetInnerHTML={{ __html: result.svg }} />;
}
