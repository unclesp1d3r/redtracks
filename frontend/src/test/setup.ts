import "@testing-library/jest-dom/vitest";

// jsdom has no matchMedia; sonner's Toaster reads it on mount.
if (!window.matchMedia) {
  window.matchMedia = (query: string) =>
    ({ matches: false, media: query, addEventListener: () => undefined, removeEventListener: () => undefined }) as unknown as MediaQueryList;
}
