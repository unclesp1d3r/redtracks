import { describe, expect, it } from "vitest";
import { isResolved, toOverrides, validPort } from "./ResolveHosts";

describe("validPort", () => {
  it.each([
    ["", false],
    ["0", false],
    ["1", true],
    ["65535", true],
    ["65536", false],
    ["abc", false],
    [" 80 ", true], // Number() trims; the input handler trims too
    ["8.5", false],
  ])("validPort(%j) is %s", (port, want) => {
    expect(validPort(port)).toBe(want);
  });
});

describe("isResolved", () => {
  it.each([
    [undefined, false],
    [{ ip: "", port: "" }, false],
    [{ ip: "10.0.0.1", port: "" }, true],
    [{ ip: "10.0.0.1", port: "80" }, true],
    [{ ip: "10.0.0.1", port: "0" }, false],
    [{ ip: "10.0.0.1", port: "abc" }, false],
    [{ ip: "", port: "80" }, false],
  ])("isResolved(%j) is %s", (choice, want) => {
    expect(isResolved(choice)).toBe(want);
  });
});

describe("toOverrides", () => {
  it("emits ip or ip:port and drops unresolved entries", () => {
    expect(
      toOverrides({
        a: { ip: "10.0.0.1", port: "" },
        b: { ip: "10.0.0.2", port: "8080" },
        c: { ip: "10.0.0.3", port: "70000" },
        d: { ip: "", port: "80" },
      }),
    ).toEqual({ a: "10.0.0.1", b: "10.0.0.2:8080" });
  });
});
