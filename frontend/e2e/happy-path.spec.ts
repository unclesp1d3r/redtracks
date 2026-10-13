import { readFileSync } from "node:fs";
import { join, resolve } from "node:path";
import { expect, test } from "@playwright/test";

const HOME = process.env.REDTRACKS_E2E_HOME ?? "";
// Playwright runs from frontend/, so this resolves to the repo-level fixture.
const NMAP_FIXTURE = resolve("../internal/importers/testdata/nmap-localhost.xml");

function readJson(...parts: string[]) {
  return JSON.parse(readFileSync(join(HOME, "engagements", "e2e", ...parts), "utf8"));
}

test("create, import, check off, add credential", async ({ page }) => {
  await page.goto("/#/");
  await page.getByRole("button", { name: "New engagement" }).first().click();
  await page.getByLabel("Name").fill("e2e");
  await page.getByRole("button", { name: "Create" }).click();
  await expect(page).toHaveURL(/#\/board/);

  await page.goto("/#/import");
  await page.getByLabel("Scan output file").setInputFiles(NMAP_FIXTURE);
  await page.getByRole("button", { name: "Confirm" }).click();
  await expect(page).toHaveURL(/#\/board/);

  await page.getByRole("link", { name: "127.0.0.1" }).click();
  await page.getByRole("navigation", { name: "Services" }).getByRole("button").first().click();
  const row = page.getByTestId(/^row-/).first();
  await row.focus();
  await page.keyboard.press("Space");
  await expect(row.getByRole("checkbox")).toBeChecked();

  await page.goto("/#/tables");
  await page.getByRole("button", { name: /^Add/ }).first().click();
  await page.getByLabel("Username").fill("admin");
  await page.getByLabel("Secret").fill("Winter2026!");
  await page.getByRole("dialog").getByRole("button", { name: "Add", exact: true }).click();
  await expect(page.getByRole("cell", { name: "admin", exact: true })).toBeVisible();

  const host = readJson("hosts", "127.0.0.1", "host.json");
  const items = [host.checklist, ...(host.services ?? []).map((s: { checklist: unknown }) => s.checklist)]
    .flatMap((c) => Object.values((c as { items?: Record<string, { status: string }> } | undefined)?.items ?? {}))
    .filter((i) => i.status === "done");
  expect(items).toHaveLength(1);

  const creds = readJson("creds.json");
  expect(creds.credentials.filter((c: { username: string }) => c.username === "admin")).toHaveLength(1);
});
