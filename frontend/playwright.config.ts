import { mkdtempSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { defineConfig, devices } from "@playwright/test";

// Set at module top level so the main process, the web server, and every worker share one path.
process.env.REDTRACKS_E2E_HOME ??= mkdtempSync(join(tmpdir(), "redtracks-e2e-"));

const PORT = "8080";
const SERVER_START_TIMEOUT_MS = 120_000;

export default defineConfig({
  testDir: "./e2e",
  workers: 1,
  reporter: "list",
  use: {
    baseURL: process.env.E2E_BASE_URL ?? `http://localhost:${PORT}`,
    trace: "retain-on-failure",
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
  webServer: {
    command: "../bin/redtracks-server",
    url: `http://localhost:${PORT}`,
    reuseExistingServer: false,
    timeout: SERVER_START_TIMEOUT_MS,
    env: { REDTRACKS_HOME: process.env.REDTRACKS_E2E_HOME, WAILS_SERVER_PORT: PORT },
  },
});
