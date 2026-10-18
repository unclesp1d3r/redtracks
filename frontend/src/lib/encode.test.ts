import { describe, expect, it } from "vitest";
import { textToBase64, tooLargeMessage, toBase64 } from "./encode";

describe("toBase64", () => {
  it("encodes raw bytes without the data URL prefix", async () => {
    const file = new File([new Uint8Array([0, 255, 10])], "x.bin");
    expect(await toBase64(file)).toBe("AP8K");
  });

  it("returns an empty string for an empty file", async () => {
    expect(await toBase64(new File([], "empty"))).toBe("");
  });
});

describe("textToBase64", () => {
  it("encodes UTF-8 text", () => {
    expect(textToBase64("héllo")).toBe("aMOpbGxv");
  });
});

describe("tooLargeMessage", () => {
  it("names the file and the limit", () => {
    expect(tooLargeMessage("big.png")).toBe("big.png is larger than 20 MB.");
  });
});
