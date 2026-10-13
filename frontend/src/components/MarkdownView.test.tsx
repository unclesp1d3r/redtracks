import { render, screen } from "@testing-library/react";
import { expect, test, vi } from "vitest";

vi.mock("@/components/Mermaid", () => ({ Mermaid: () => null }));

import { MarkdownView } from "./MarkdownView";

test("renders remote images as alt text, never an img", () => {
  const { container } = render(<MarkdownView text={"![logo](https://evil.example/x.png)"} />);
  expect(container.querySelector("img")).toBeNull();
  expect(screen.getByText("logo")).toBeInTheDocument();
});

test("renders the URL as text when a remote image has no alt", () => {
  const { container } = render(<MarkdownView text={"![](http://evil.example/x.png)"} />);
  expect(container.querySelector("img")).toBeNull();
  expect(screen.getByText("http://evil.example/x.png")).toBeInTheDocument();
});

test("keeps local images", () => {
  const { container } = render(<MarkdownView text={"![shot](data:image/png;base64,AAAA)"} />);
  expect(container.querySelector("img")).not.toBeNull();
});
