import { beforeEach, expect, it } from "vitest";
import { errorBanner, type ErrorEvent } from "./errorBanner";

const ev = (Kind: ErrorEvent["Kind"], Path: string, Message = "bad"): ErrorEvent => ({ Kind, Path, Message });

beforeEach(() => {
  errorBanner.clearKind("engagement");
  errorBanner.clearKind("checklists");
});

it("keeps one entry per Kind+Path and replaces repeats", () => {
  errorBanner.add(ev("engagement", "a.json"));
  errorBanner.add(ev("engagement", "b.json"));
  errorBanner.add(ev("engagement", "a.json", "newer"));
  expect(errorBanner.get().map((e) => `${e.Path}:${e.Message}`)).toEqual(["b.json:bad", "a.json:newer"]);
});

it("clears only engagement entries whose path changed; path-less ones wait for a reopen", () => {
  errorBanner.add(ev("engagement", "a.json"));
  errorBanner.add(ev("engagement", "b.json"));
  errorBanner.add(ev("engagement", ""));
  errorBanner.add(ev("checklists", ""));
  errorBanner.clearChanged(["a.json"]);
  expect(errorBanner.get().map((e) => `${e.Kind}:${e.Path}`)).toEqual([
    "engagement:b.json",
    "engagement:",
    "checklists:",
  ]);
  errorBanner.clearKind("engagement");
  expect(errorBanner.get().map((e) => `${e.Kind}:${e.Path}`)).toEqual(["checklists:"]);
});
