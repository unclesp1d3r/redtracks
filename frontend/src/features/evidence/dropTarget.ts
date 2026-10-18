import { useState } from "react";
import type { DragEvent } from "react";
import type { EvidenceLink } from "@/lib/api";
import { useCapabilities, useCapture } from "./useEvidence";

/** Wails only reports drops on elements carrying data-file-drop-target; the value is not interpreted. */
export const DROP_MARKER = "evidence";

/** Wails adds this class to the hovered drop target; the HTML5 path uses data-drop-active instead. */
export const DROP_ACTIVE_CLASSES =
  "data-[drop-active=true]:bg-muted data-[drop-active=true]:ring-2 data-[drop-active=true]:ring-primary " +
  "[&.file-drop-target-active]:bg-muted [&.file-drop-target-active]:ring-2 [&.file-drop-target-active]:ring-primary";

/** "<kind>:<targetId>[:<serviceId>]", the format Dropper.parseTarget reads. */
export function formatTarget(link: EvidenceLink): string {
  return [link.kind, link.targetId, ...(link.serviceId ? [link.serviceId] : [])].join(":");
}

/** The attributes the desktop window hands to Go with a dropped path. No target means host only. */
export function dropAttrs(ip: string, link: EvidenceLink | undefined): Record<string, string> {
  return {
    "data-file-drop-target": DROP_MARKER,
    "data-evidence-host": ip,
    ...(link ? { "data-evidence-target": formatTarget(link) } : {}),
  };
}

const hasFiles = (e: DragEvent) => Array.from(e.dataTransfer?.types ?? []).includes("Files");

/**
 * Native window: data attributes only, Go does the capture. Server build: HTML5 handlers that send
 * the same capture through AddFromData.
 */
export function useDropTarget(ip: string, link?: EvidenceLink) {
  const { data: caps } = useCapabilities();
  const { captureFiles } = useCapture(ip);
  const [isOver, setIsOver] = useState(false);
  if (caps?.NativeDrop) return { props: dropAttrs(ip, link) };
  return {
    props: {
      "data-drop-active": isOver,
      onDragOver: (e: DragEvent) => {
        if (!hasFiles(e)) return;
        e.preventDefault();
        setIsOver(true);
      },
      onDragLeave: () => setIsOver(false),
      onDrop: (e: DragEvent) => {
        if (!hasFiles(e)) return;
        e.preventDefault();
        e.stopPropagation();
        setIsOver(false);
        captureFiles(Array.from(e.dataTransfer.files), link ? [link] : []);
      },
    },
  };
}
