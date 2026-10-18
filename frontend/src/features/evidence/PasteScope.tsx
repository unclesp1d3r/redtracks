import { useMemo, useState } from "react";
import type { ReactNode } from "react";
import { PasteContext, usePasteListener, usePasteScope } from "./usePaste";
import type { PasteTarget } from "./usePaste";

/** Owns the paste target for one host page and installs its paste listener while mounted. */
export function PasteScope({ ip, children }: { ip: string; children: ReactNode }) {
  const [target, setTarget] = useState<PasteTarget | null>(null);
  const value = useMemo(
    () => ({
      target,
      focusTarget: setTarget,
      blurTarget: (key: string) => setTarget((t) => (t?.key === key ? null : t)),
    }),
    [target],
  );
  usePasteListener(ip, target);
  return <PasteContext.Provider value={value}>{children}</PasteContext.Provider>;
}

export function PasteIndicator() {
  const { target } = usePasteScope();
  return (
    <p role="status" className="text-xs text-muted-foreground">
      Paste target: <strong className="font-medium text-foreground">{target?.name ?? "host"}</strong>
    </p>
  );
}
