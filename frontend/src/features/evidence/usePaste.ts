import { createContext, useContext, useEffect, useRef } from "react";
import type { FocusEvent } from "react";
import type { EvidenceLink } from "@/lib/api";
import { cn } from "@/lib/utils";
import { DROP_ACTIVE_CLASSES, formatTarget, useDropTarget } from "./dropTarget";
import { useCapture } from "./useEvidence";

/** The row paste will attach to: its link and the name shown in the paste-target indicator. */
export interface PasteTarget {
  key: string;
  link: EvidenceLink;
  name: string;
}

interface PasteScopeValue {
  target: PasteTarget | null;
  focusTarget: (t: PasteTarget) => void;
  blurTarget: (key: string) => void;
}

const NO_SCOPE: PasteScopeValue = { target: null, focusTarget: () => undefined, blurTarget: () => undefined };
export const PasteContext = createContext<PasteScopeValue>(NO_SCOPE);

export function usePasteScope(): PasteScopeValue {
  return useContext(PasteContext);
}

/**
 * Props that make an element a focusable proof target: it registers as the paste target on focus,
 * shows a focus ring, and accepts a drop (native attributes or HTML5 handlers).
 */
export function useTargetRow(ip: string, link: EvidenceLink, name: string) {
  const scope = usePasteScope();
  const drop = useDropTarget(ip, link);
  const key = formatTarget(link);
  return {
    role: "group" as const,
    tabIndex: 0,
    "aria-label": `Paste proof for ${name}`,
    className: cn("outline-none focus-visible:ring-2 focus-visible:ring-ring", DROP_ACTIVE_CLASSES),
    onFocus: () => scope.focusTarget({ key, link, name }),
    onBlur: (e: FocusEvent<HTMLElement>) => {
      if (!e.currentTarget.contains(e.relatedTarget as Node | null)) scope.blurTarget(key);
    },
    ...drop.props,
  };
}

const EDITABLE = "input,textarea,select,[contenteditable='true']";

/** Files from clipboardData.files, else from items (some webviews expose images only there). */
function clipboardFiles(data: DataTransfer): File[] {
  if (data.files.length > 0) return Array.from(data.files);
  return Array.from(data.items ?? [])
    .filter((i) => i.kind === "file")
    .flatMap((i) => i.getAsFile() ?? []);
}

/** Document-level paste handler: files first, then text/plain. Pastes into text fields are left alone. */
export function usePasteListener(ip: string, target: PasteTarget | null) {
  const { captureFiles, captureText } = useCapture(ip);
  const latest = useRef({ target, captureFiles, captureText });
  latest.current = { target, captureFiles, captureText };

  useEffect(() => {
    function onPaste(e: ClipboardEvent) {
      if ((e.target as Element | null)?.closest?.(EDITABLE) || !e.clipboardData) return;
      const { target: t, captureFiles: files, captureText: text } = latest.current;
      const links = t ? [t.link] : [];
      const pasted = clipboardFiles(e.clipboardData);
      if (pasted.length > 0) {
        e.preventDefault();
        files(pasted, links);
        return;
      }
      const plain = e.clipboardData.getData("text/plain");
      if (plain.trim() === "") return;
      e.preventDefault();
      text(plain, links);
    }
    document.addEventListener("paste", onPaste);
    return () => document.removeEventListener("paste", onPaste);
  }, []);
}
