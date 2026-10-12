import type { ComponentProps } from "react";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { Mermaid } from "@/components/Mermaid";

const LANGUAGE_PREFIX = "language-";

/** Fenced mermaid blocks render as diagrams; everything else is a plain code element. */
function Code({ className, children, ...rest }: ComponentProps<"code">) {
  if (className === `${LANGUAGE_PREFIX}mermaid`) {
    return <Mermaid text={String(children).replace(/\n$/, "")} />;
  }
  return (
    <code className={className} {...rest}>
      {children}
    </code>
  );
}

/** Links render as text with the target shown, never as navigable anchors inside the app window. */
function Link({ href, children }: ComponentProps<"a">) {
  return (
    <span>
      {children}
      {href ? <span className="text-muted-foreground"> ({href})</span> : null}
    </span>
  );
}

/**
 * GFM Markdown. Raw HTML is not enabled (react-markdown ignores it without rehype-raw),
 * so notes cannot inject markup.
 */
export function MarkdownView({ text }: { text: string }) {
  return (
    <div className="space-y-2 text-sm [&_h1]:text-lg [&_h1]:font-semibold [&_h2]:font-semibold [&_pre]:overflow-auto [&_pre]:rounded-md [&_pre]:bg-muted [&_pre]:p-2 [&_table]:border [&_td]:border [&_td]:px-2 [&_th]:border [&_th]:px-2 [&_ul]:list-disc [&_ul]:pl-5 [&_ol]:list-decimal [&_ol]:pl-5">
      <Markdown remarkPlugins={[remarkGfm]} components={{ code: Code, a: Link }}>
        {text}
      </Markdown>
    </div>
  );
}
