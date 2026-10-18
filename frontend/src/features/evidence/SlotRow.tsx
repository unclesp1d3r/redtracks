import { ProofStrip } from "./ProofStrip";
import { slotLink } from "./proof";
import { useTargetRow } from "./usePaste";
import { cn } from "@/lib/utils";

interface Slot {
  id: string;
  title: string;
  hint: string;
}

interface Props {
  ip: string;
  /** "" for a host-level slot. */
  serviceId: string;
  slot: Slot;
}

export function SlotRow({ ip, serviceId, slot }: Props) {
  const link = slotLink(slot.id, serviceId);
  const name = `slot ${slot.title}`;
  const rowProps = useTargetRow(ip, link, name);
  return (
    <li>
      <div {...rowProps} className={cn(rowProps.className, "space-y-1 p-2")}>
        <div className="flex items-baseline gap-2">
          <span className="text-sm font-medium">{slot.title}</span>
          {slot.hint && <span className="text-xs text-muted-foreground">{slot.hint}</span>}
        </div>
        <ProofStrip ip={ip} link={link} name={name} words={{ yes: "filled", no: "missing" }} />
      </div>
    </li>
  );
}

/** The proof slots a definition owes, shown after its checklist items. */
export function SlotRows({ ip, serviceId, slots }: { ip: string; serviceId: string; slots: Slot[] }) {
  if (slots.length === 0) return null;
  return (
    <ul aria-label="Proof slots" className="mt-1 divide-y rounded-md border border-dashed">
      {slots.map((s) => (
        <SlotRow key={s.id} ip={ip} serviceId={serviceId} slot={s} />
      ))}
    </ul>
  );
}
