import { readFileSync, utimesSync, writeFileSync } from "node:fs";
import { join, resolve } from "node:path";
import { expect, test } from "@playwright/test";

const HOME = process.env.REDTRACKS_E2E_HOME ?? "";
const NMAP_FIXTURE = resolve("../internal/importers/testdata/nmap-localhost.xml");
const ENGAGEMENT = "e2e-edit";
const HOST_REL = "hosts/127.0.0.1/host.json";
const POLL_WAIT_MS = 15_000; // the backend polls every 2s

const engagementFile = (rel: string) => join(HOME, "engagements", ENGAGEMENT, rel);

let tick = 0;
/** Writes a file and moves its mtime forward so the mtime poll always sees a change. */
function writeExternally(rel: string, content: string) {
  const path = engagementFile(rel);
  writeFileSync(path, content);
  tick += 10;
  const when = new Date(Date.now() + tick * 1000);
  utimesSync(path, when, when);
}

test("edits made outside the app show up, bad JSON is reported, notes conflict", async ({ page }) => {
  await page.goto("/#/");
  await page.getByRole("button", { name: "New engagement" }).first().click();
  await page.getByLabel("Name").fill(ENGAGEMENT);
  await page.getByRole("button", { name: "Create" }).click();
  await expect(page).toHaveURL(/#\/board/);

  await page.goto("/#/import");
  await page.getByLabel("Scan output file").setInputFiles(NMAP_FIXTURE);
  await page.getByRole("button", { name: "Confirm" }).click();
  await expect(page).toHaveURL(/#\/board/);
  await expect(page.getByRole("link", { name: "127.0.0.1" })).toBeVisible();

  // A valid edit appears on the board without a reload.
  const original = readFileSync(engagementFile(HOST_REL), "utf8");
  const edited = JSON.stringify({ ...JSON.parse(original), hostname: "edited-outside" }, null, 2);
  writeExternally(HOST_REL, edited);
  await expect(page.getByText("edited-outside", { exact: true })).toBeVisible({ timeout: POLL_WAIT_MS });

  // Invalid JSON raises the banner naming the file.
  writeExternally(HOST_REL, "{ not json");
  const banner = page.getByRole("alert").filter({ hasText: HOST_REL });
  await expect(banner).toBeVisible({ timeout: POLL_WAIT_MS });

  // Restoring a valid file clears it.
  writeExternally(HOST_REL, edited);
  await expect(banner).toBeHidden({ timeout: POLL_WAIT_MS });

  // Notes edited on disk while the editor holds unsaved text conflict on save.
  await page.getByRole("link", { name: "127.0.0.1" }).click();
  await page.getByLabel("Notes text").fill("typed in the app");
  writeExternally("hosts/127.0.0.1/notes.md", "typed on disk\n");
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await expect(page.getByRole("dialog").getByText("Notes changed on disk")).toBeVisible();
});
