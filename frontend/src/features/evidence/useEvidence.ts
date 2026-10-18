import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { EvidenceService, errorMessage } from "@/lib/api";
import type { EvidenceLink } from "@/lib/api";
import { reportAdded, reportRefused } from "@/lib/capture";
import { textToBase64, tooLargeMessage, toBase64, MAX_IMPORT_BYTES } from "@/lib/encode";
import { keys } from "@/lib/query";

/** One host's evidence in capture order. Mutations refresh it through the global onSuccess. */
export function useEvidence(ip: string) {
  return useQuery({ queryKey: keys.evidence(ip), queryFn: () => EvidenceService.List(ip) });
}

/** Which desktop integrations this build has; fixed for the process lifetime. */
export function useCapabilities() {
  return useQuery({ queryKey: keys.capabilities(), queryFn: () => EvidenceService.Capabilities(), staleTime: Infinity });
}

interface ItemLink {
  itemId: string;
  link: EvidenceLink;
}

export function useEvidenceActions(ip: string) {
  return {
    link: useMutation({ mutationFn: (v: ItemLink) => EvidenceService.Link(ip, v.itemId, v.link) }),
    unlink: useMutation({ mutationFn: (v: ItemLink) => EvidenceService.Unlink(ip, v.itemId, v.link) }),
    setCaption: useMutation({
      mutationFn: (v: { itemId: string; caption: string }) => EvidenceService.SetCaption(ip, v.itemId, v.caption),
    }),
    drop: useMutation({ mutationFn: (itemId: string) => EvidenceService.Drop(ip, itemId) }),
    adopt: useMutation({ mutationFn: (fileName: string) => EvidenceService.Adopt(ip, fileName) }),
  };
}

interface AddInput {
  name: string;
  data: string;
  lastModified: number;
  links: EvidenceLink[];
}

/** Capture from files or pasted text; every outcome ends in a toast, never silence. */
export function useCapture(ip: string) {
  const qc = useQueryClient();
  const add = useMutation({
    meta: { silent: true },
    mutationFn: (v: AddInput) => EvidenceService.AddFromData(ip, v.name, v.data, v.lastModified, "", v.links),
    onSuccess: (r) => reportAdded(qc, r.Item, r.Duplicate),
    onError: (e, v) => reportRefused(v.name, errorMessage(e)),
  });

  async function captureFile(file: File, links: EvidenceLink[]) {
    if (file.size > MAX_IMPORT_BYTES) {
      toast.error(tooLargeMessage(file.name), { duration: Infinity });
      return;
    }
    try {
      add.mutate({ name: file.name, data: await toBase64(file), lastModified: file.lastModified || 0, links });
    } catch (e) {
      reportRefused(file.name, errorMessage(e));
    }
  }

  return {
    captureFiles: (files: readonly File[], links: EvidenceLink[]) => files.forEach((f) => void captureFile(f, links)),
    captureText: (text: string, links: EvidenceLink[]) =>
      add.mutate({ name: "paste.txt", data: textToBase64(text), lastModified: 0, links }),
  };
}
