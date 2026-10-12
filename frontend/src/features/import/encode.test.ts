import { describe, expect, it } from "vitest";
import { toBase64 } from "./encode";

describe("toBase64", () => {
  it("encodes raw bytes without the data URL prefix", async () => {
    const file = new File([new Uint8Array([0, 255, 10])], "x.bin");
    expect(await toBase64(file)).toBe("AP8K");
  });

  it("returns an empty string for an empty file", async () => {
    expect(await toBase64(new File([], "empty"))).toBe("");
  });
});
