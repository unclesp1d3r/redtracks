export const MAX_IMPORT_BYTES = 20 * 1024 * 1024;

/** Reads a blob as base64 without the data-URL prefix. */
export function toBase64(file: Blob): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onerror = () => reject(reader.error ?? new Error("could not read file"));
    reader.onload = () => {
      const url = String(reader.result);
      resolve(url.slice(url.indexOf(",") + 1));
    };
    reader.readAsDataURL(file);
  });
}

export function tooLargeMessage(name: string): string {
  return `${name} is larger than ${MAX_IMPORT_BYTES / (1024 * 1024)} MB.`;
}

/** Standard base64 of the UTF-8 bytes of text. */
export function textToBase64(text: string): string {
  let binary = "";
  new TextEncoder().encode(text).forEach((b) => {
    binary += String.fromCharCode(b);
  });
  return btoa(binary);
}
