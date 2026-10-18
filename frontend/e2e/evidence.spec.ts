import { existsSync, mkdirSync, readdirSync, readFileSync, statSync, utimesSync, writeFileSync } from "node:fs";
import { join, resolve } from "node:path";
import { expect, test, type Locator, type Page } from "@playwright/test";

const HOME = process.env.REDTRACKS_E2E_HOME ?? "";
const NMAP_FIXTURE = resolve("../internal/importers/testdata/nmap-localhost.xml");
const POLL_WAIT_MS = 15_000; // the inbox watcher polls every 2s and needs two agreeing polls
const HOST_IP = "127.0.0.1";
// The bundled exam-proofs definition declares host-level proof slots for every host.
const FOOTHOLD_SLOT_ID = "exam-proofs.foothold";
const FOOTHOLD_TITLE = "Foothold";
const FOOTHOLD_HINT_START = "How the host was reached";
// A real 1x1 PNG (signature + IHDR + IDAT + IEND).
const PNG_BASE64 =
  "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==";
const pngBytes = () => Buffer.from(PNG_BASE64, "base64");

const engagementDir = (name: string) => join(HOME, "engagements", name);
const hostDir = (name: string) => join(engagementDir(name), "hosts", HOST_IP);
const indexPath = (name: string) => join(hostDir(name), "evidence.json");
const readIndex = (name: string) => JSON.parse(readFileSync(indexPath(name), "utf8"));
// The file lands before its index entry, so wait on the index, not the evidence folder.
const indexedCount = (name: string) => (existsSync(indexPath(name)) ? readIndex(name).items.length : 0);
const evidenceFiles = (name: string) => {
  const dir = join(hostDir(name), "evidence");
  return existsSync(dir) ? readdirSync(dir) : [];
};

async function createEngagementWithHost(page: Page, name: string) {
  await page.goto("/#/");
  await page.getByRole("button", { name: "New engagement" }).first().click();
  await page.getByLabel("Name").fill(name);
  await page.getByRole("button", { name: "Create" }).click();
  await expect(page).toHaveURL(/#\/board/);
  await page.goto("/#/import");
  await page.getByLabel("Scan output file").setInputFiles(NMAP_FIXTURE);
  await page.getByRole("button", { name: "Confirm" }).click();
  await expect(page).toHaveURL(/#\/board/);
  await expect(page.getByRole("link", { name: HOST_IP })).toBeVisible();
}

async function openEvidenceTab(page: Page) {
  await page.getByRole("link", { name: HOST_IP }).click();
  await page.getByRole("tab", { name: "Host checklists" }).click();
}

/** Opens a base-ui select by its accessible name and picks an option. */
async function pick(scope: Page | Locator, select: string, option: string) {
  await scope.getByRole("combobox", { name: select }).click();
  await scope.page().getByRole("option", { name: option, exact: true }).click();
}

/** Focuses a paste-target row, then fires a synthetic paste carrying one PNG file. */
async function pastePng(page: Page, row: Locator) {
  await row.focus();
  await page.evaluate((b64) => {
    const bytes = Uint8Array.from(atob(b64), (c) => c.charCodeAt(0));
    const data = new DataTransfer();
    data.items.add(new File([bytes], "pasted.png", { type: "image/png" }));
    document.body.dispatchEvent(new ClipboardEvent("paste", { clipboardData: data, bubbles: true, cancelable: true }));
  }, PNG_BASE64);
}

test("an inbox file assigned to a host slot lands on disk and fills the slot", async ({ page }) => {
  const name = "e2e-ev-inbox";
  await createEngagementWithHost(page, name);

  mkdirSync(join(HOME, "inbox"), { recursive: true });
  const dropped = join(HOME, "inbox", "shot.png");
  writeFileSync(dropped, pngBytes());
  const past = new Date(Date.now() - 30_000);
  utimesSync(dropped, past, past);

  await page.goto("/#/inbox");
  const row = page.getByRole("listitem", { name: "shot.png" });
  await expect(row).toBeVisible({ timeout: POLL_WAIT_MS });
  await expect(page.getByLabel("1 file in inbox")).toBeVisible();

  await row.getByRole("button", { name: "Assign shot.png" }).click();
  const dialog = page.getByRole("dialog", { name: "Assign shot.png" });
  await pick(dialog, "Host", HOST_IP);
  await pick(dialog, "Slot", FOOTHOLD_TITLE);
  await dialog.getByRole("button", { name: "Assign to host" }).click();
  await expect(dialog).toBeHidden();
  await expect(page.getByText(/^Nothing waiting\./)).toBeVisible();

  await expect.poll(() => indexedCount(name)).toBe(1);
  const { items } = readIndex(name);
  expect(items).toHaveLength(1);
  expect(evidenceFiles(name)).toHaveLength(1);
  expect(items[0].kind).toBe("image");
  expect(items[0].links).toEqual([expect.objectContaining({ kind: "slot", targetId: FOOTHOLD_SLOT_ID })]);
  expect(existsSync(dropped)).toBe(false);

  await page.goto("/#/board");
  await openEvidenceTab(page);
  const slot = page.getByRole("group", { name: `Paste proof for slot ${FOOTHOLD_TITLE}` });
  await expect(slot.getByText("filled", { exact: true })).toBeVisible();
});

test("a pasted screenshot attaches to the focused finding only", async ({ page }) => {
  const name = "e2e-ev-paste";
  await createEngagementWithHost(page, name);

  await page.goto("/#/tables");
  await page.getByRole("tab", { name: "Findings" }).click();
  await page.getByRole("button", { name: /^Add/ }).first().click();
  const add = page.getByRole("dialog");
  await pick(add, "Host", HOST_IP);
  // Findings render under the service they belong to; the host page selects the first service.
  await add.getByRole("combobox", { name: "Service" }).click();
  await page.getByRole("option").nth(1).click();
  await add.getByLabel("Title").fill("Anonymous FTP");
  await add.getByRole("button", { name: "Add", exact: true }).click();
  await expect(page.getByRole("cell", { name: "Anonymous FTP", exact: true })).toBeVisible();

  await page.getByRole("link", { name: "Board" }).click();
  await page.getByRole("link", { name: HOST_IP }).click();
  await page.getByRole("tab", { name: /^Findings/ }).click();
  const row = page.getByRole("group", { name: "Paste proof for finding Anonymous FTP" });
  await expect(row).toBeVisible();
  await pastePng(page, row);

  await expect.poll(() => indexedCount(name)).toBe(1);
  const { items } = readIndex(name);
  expect(items).toHaveLength(1);
  expect(items[0].links).toHaveLength(1);
  expect(items[0].links[0].kind).toBe("finding");
});

test("a session log clip keeps exactly the chosen lines without colour codes", async ({ page }) => {
  const name = "e2e-ev-clip";
  await createEngagementWithHost(page, name);

  const marker = "zebracrossing";
  const lines = Array.from({ length: 2000 }, (_, i) => {
    const n = i + 1;
    const text = n === 1234 ? `marker ${marker}` : `output line ${n}`;
    return n % 7 === 0 ? `\u001b[31m${text}\u001b[0m` : text;
  });
  const source = join(HOME, "gen.log");
  writeFileSync(source, `${lines.join("\n")}\n`);

  await page.goto("/#/logs");
  await page.getByLabel("Path to a session log").fill(source);
  await page.getByRole("button", { name: "Import" }).click();
  // Importing opens the new log; the server names the copy by id, not by the source name.
  const viewer = page.getByRole("region", { name: /^Log .+\.log$/ });
  await expect(viewer).toBeVisible();

  await viewer.getByLabel("Search in log").fill(marker);
  await viewer.getByLabel("Search in log").press("Enter");
  await expect(viewer.getByRole("button", { name: "Line 1234" })).toBeVisible();

  await viewer.getByLabel("Clip range").fill("10-49");
  await viewer.getByRole("button", { name: "Clip to evidence" }).click();
  const dialog = page.getByRole("dialog");
  await pick(dialog, "Host", HOST_IP);
  await dialog.getByRole("button", { name: "Save clip" }).click();
  await expect(dialog.getByText(`Saved to ${HOST_IP}.`)).toBeVisible();

  const logs = join(engagementDir(name), "logs");
  const copies = readdirSync(logs).filter((f) => f.endsWith(".log"));
  expect(copies).toHaveLength(1);
  expect(statSync(join(logs, copies[0])).size).toBe(statSync(source).size);

  const { items } = readIndex(name);
  expect(items).toHaveLength(1);
  expect(items[0].kind).toBe("text");
  expect(items[0].clipOf).toEqual({ logId: copies[0].replace(/\.log$/, ""), firstLine: 9, lastLine: 48 });
  const clipFile = join(hostDir(name), "evidence", items[0].file);
  const expected = lines.slice(9, 49).map((l) => l.replace(/\u001b\[[0-9;]*m/g, ""));
  expect(readFileSync(clipFile, "utf8").replace(/\n$/, "").split("\n")).toEqual(expected);
});

test("the board lists the host owing more proof first and What's left names the gap", async ({ page }) => {
  const name = "e2e-ev-board";
  await createEngagementWithHost(page, name);

  // 203.0.113.7 sorts after 127.0.0.1 by address, so owed-first ordering is observable.
  await page.getByRole("button", { name: "Add host" }).first().click();
  await page.getByLabel("IP address").fill("203.0.113.7");
  await page.getByRole("button", { name: "Add host", exact: true }).last().click();
  await expect(page.getByRole("link", { name: "203.0.113.7" })).toBeVisible();

  await openEvidenceTab(page);
  await pastePng(page, page.getByRole("group", { name: `Paste proof for slot ${FOOTHOLD_TITLE}` }));
  await expect.poll(() => indexedCount(name)).toBe(1);

  await page.goto("/#/board");
  const owed = page.getByTestId("owed-proof");
  await expect(owed).toHaveCount(2);
  const [first, second] = await owed.allTextContents();
  const count = (t: string) => Number(/^(\d+) proofs? owed/.exec(t)?.[1]);
  expect(count(first)).toBe(count(second) + 1);
  await expect(page.getByRole("row").nth(1)).toContainText("203.0.113.7");

  await page.goto("/#/left");
  await expect(page.getByText(FOOTHOLD_TITLE, { exact: true }).first()).toBeVisible();
  await expect(page.getByText(new RegExp(`^${FOOTHOLD_HINT_START}`)).first()).toBeVisible();
});

test("the bundled exam-proofs slots appear on a fresh host", async ({ page }) => {
  await createEngagementWithHost(page, "e2e-ev-bundled");
  await openEvidenceTab(page);
  await expect(page.getByRole("group", { name: "Paste proof for slot Foothold" })).toBeVisible();
});
