import { useEffect, useId, useState } from "react";
import DOMPurify from "dompurify";
import mermaid from "mermaid";

mermaid.initialize({ startOnLoad: false, securityLevel: "strict", theme: "dark" });

type Rendered = { svg: string } | { error: string };

export function Mermaid({ text }: { text: string }) {
  const id = "mmd" + useId().replace(/:/g, "");
  const [result, setResult] = useState<Rendered | null>(null);

  useEffect(() => {
    let cancelled = false;
    mermaid
      .render(id, text)
      .then(({ svg }) => {
        if (cancelled) return;
        const clean = DOMPurify.sanitize(svg, { USE_PROFILES: { svg: true, svgFilters: true } });
        setResult({ svg: clean });
      })
      .catch((e: unknown) => {
        if (!cancelled) setResult({ error: e instanceof Error ? e.message : String(e) });
      });
    return () => {
      cancelled = true;
    };
  }, [id, text]);

  if (result && "error" in result) {
    return (
      <div>
        <p className="mb-1 text-xs text-muted-foreground">Diagram failed to render.</p>
        <pre className="overflow-auto rounded-md bg-muted p-2 text-xs">{text}</pre>
      </div>
    );
  }
  if (!result) return null;
  return <div className="overflow-auto [&_svg]:mx-auto" dangerouslySetInnerHTML={{ __html: result.svg }} />;
}
