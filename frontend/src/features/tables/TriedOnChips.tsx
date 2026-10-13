import { CheckIcon, MinusIcon } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import type { Credential, Host } from "@/lib/api";
import { hostIp } from "./shared";

interface Props {
  credential: Credential;
  hosts: Host[];
}

/** "works" (green, check icon) and "tried" (grey, dash icon): text and icon, never colour alone. */
export function TriedOnChips({ credential, hosts }: Props) {
  const works = credential.worksOn ?? [];
  const tried = (credential.triedOn ?? []).filter((id) => !works.includes(id));
  return (
    <div className="flex flex-wrap gap-1">
      {works.map((id) => (
        <Badge key={`w-${id}`} className="bg-green-600/15 text-green-800 dark:text-green-300">
          <CheckIcon aria-hidden /> works {hostIp(hosts, id)}
        </Badge>
      ))}
      {tried.map((id) => (
        <Badge key={`t-${id}`} variant="secondary">
          <MinusIcon aria-hidden /> tried {hostIp(hosts, id)}
        </Badge>
      ))}
    </div>
  );
}
