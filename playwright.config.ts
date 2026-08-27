import { defineConfig } from "@playwright/test";

const siteRoot = process.env.PUBLIC_SITE_ROOT ?? "/tmp/backstop-site";

export default defineConfig({
  testDir: "./tests",
  testMatch: "public-site.spec.ts",
  fullyParallel: false,
  workers: 1,
  retries: 0,
  timeout: 45_000,
  reporter: [["line"]],
  outputDir: process.env.PLAYWRIGHT_OUTPUT_DIR ?? "/tmp/backstop-playwright-results",
  use: {
    baseURL: "http://127.0.0.1:4173",
    browserName: "chromium",
    javaScriptEnabled: false,
    actionTimeout: 5_000,
    navigationTimeout: 10_000,
  },
  webServer: {
    command: `python3 -m http.server 4173 --bind 127.0.0.1 --directory ${JSON.stringify(siteRoot)}`,
    url: "http://127.0.0.1:4173/",
    reuseExistingServer: false,
    timeout: 15_000,
  },
});
