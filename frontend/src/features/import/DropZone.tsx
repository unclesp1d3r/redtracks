import { useRef, useState, type DragEvent } from "react";
import { UploadIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { ImportService, errorMessage, type PreviewResponse } from "@/lib/api";
import { cn } from "@/lib/utils";
import { MAX_IMPORT_BYTES, toBase64 } from "./encode";

interface Props {
  onPreview: (response: PreviewResponse) => void;
}

export function DropZone({ onPreview }: Props) {
  const input = useRef<HTMLInputElement>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [over, setOver] = useState(false);

  const handle = async (file: File) => {
    setError(null);
    if (file.size > MAX_IMPORT_BYTES) {
      setError(`${file.name} is larger than ${MAX_IMPORT_BYTES / (1024 * 1024)} MB.`);
      return;
    }
    setBusy(true);
    try {
      onPreview(await ImportService.PreviewData(file.name, await toBase64(file)));
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
      if (input.current) input.current.value = "";
    }
  };

  const onDrop = (e: DragEvent) => {
    e.preventDefault();
    setOver(false);
    const file = e.dataTransfer.files[0];
    if (file) void handle(file);
  };

  return (
    <div className="space-y-2">
      <div
        data-testid="drop-zone"
        onDragOver={(e) => {
          e.preventDefault();
          setOver(true);
        }}
        onDragLeave={() => setOver(false)}
        onDrop={onDrop}
        className={cn(
          "flex flex-col items-center gap-3 rounded-lg border-2 border-dashed px-6 py-12 text-center",
          over && "border-primary bg-muted",
        )}
      >
        <UploadIcon aria-hidden />
        <p className="text-sm text-muted-foreground">
          {busy ? "Reading file..." : "Drop scan output here (nmap, masscan, ffuf, feroxbuster, gobuster, nikto, nuclei, enum4linux-ng, netexec)"}
        </p>
        <Button variant="outline" disabled={busy} onClick={() => input.current?.click()}>
          Browse
        </Button>
        <input
          ref={input}
          type="file"
          hidden
          aria-label="Scan output file"
          onChange={(e) => {
            const file = e.target.files?.[0];
            if (file) void handle(file);
          }}
        />
      </div>
      {error && (
        <p role="alert" className="font-mono text-xs whitespace-pre-wrap text-destructive">
          {error}
        </p>
      )}
    </div>
  );
}
