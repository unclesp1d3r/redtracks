// jsdom has no DataTransfer or ClipboardEvent; these shims build just enough for paste handlers.

export function pngFile(name = "shot.png", size = 8): File {
  return new File([new Uint8Array(size)], name, { type: "image/png", lastModified: 1_700_000_000_000 });
}

interface ClipboardInit {
  files?: File[];
  /** Entries exposed only through clipboardData.items, as some webviews do. */
  itemFiles?: File[];
  text?: string;
}

/** A DataTransfer stand-in with files, items, and text/plain. */
export function makeClipboardData({ files = [], itemFiles = [], text = "" }: ClipboardInit): DataTransfer {
  const items = [
    ...files.map((f) => ({ kind: "file", type: f.type, getAsFile: () => f })),
    ...itemFiles.map((f) => ({ kind: "file", type: f.type, getAsFile: () => f })),
    ...(text ? [{ kind: "string", type: "text/plain", getAsFile: () => null }] : []),
  ];
  return {
    files,
    items,
    getData: (type: string) => (type === "text/plain" ? text : ""),
  } as unknown as DataTransfer;
}

/** A bubbling, cancelable paste event carrying the clipboard data. */
export function pasteEvent(init: ClipboardInit): Event {
  const event = new Event("paste", { bubbles: true, cancelable: true });
  Object.defineProperty(event, "clipboardData", { value: makeClipboardData(init) });
  return event;
}

/** A bubbling, cancelable drop event carrying dropped files. */
export function dropEvent(files: File[]): Event {
  const event = new Event("drop", { bubbles: true, cancelable: true });
  Object.defineProperty(event, "dataTransfer", { value: { files, types: ["Files"] } });
  return event;
}
