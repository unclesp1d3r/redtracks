import { useRef } from "react";
import { PaperclipIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import type { EvidenceLink } from "@/lib/api";
import { useCapture } from "./useEvidence";

interface Props {
  ip: string;
  /** Absent means a host-level capture. */
  link?: EvidenceLink;
  label: string;
  inputLabel: string;
  /** Show the label next to the icon instead of icon only. */
  isTextVisible?: boolean;
}

/** A real button that opens a hidden file input, so keyboard users can capture without drag or paste. */
export function AttachFile({ ip, link, label, inputLabel, isTextVisible = false }: Props) {
  const input = useRef<HTMLInputElement>(null);
  const { captureFiles } = useCapture(ip);
  return (
    <>
      <Button
        size={isTextVisible ? "xs" : "icon-xs"}
        variant="outline"
        aria-label={label}
        title={label}
        onClick={() => input.current?.click()}
      >
        <PaperclipIcon aria-hidden />
        {isTextVisible && label}
      </Button>
      <input
        ref={input}
        type="file"
        multiple
        hidden
        aria-label={inputLabel}
        onChange={(e) => {
          captureFiles(Array.from(e.target.files ?? []), link ? [link] : []);
          e.target.value = "";
        }}
      />
    </>
  );
}
