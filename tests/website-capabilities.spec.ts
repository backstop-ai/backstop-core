import { expect, test } from "@playwright/test";

const journeys = [
  { key: "CAP-004/@UJ-001", jlink: "JLINK-001", first: "/" },
  { key: "CAP-004/@UJ-002", jlink: "JLINK-002", first: "/evaluate/" },
  { key: "CAP-005/@UJ-001", jlink: "JLINK-003", first: "/use-cases/" },
  { key: "CAP-005/@UJ-002", jlink: "JLINK-005", first: "/model/" },
  { key: "CAP-006/@UJ-001", jlink: "JLINK-006", first: "/model/" },
  { key: "CAP-006/@UJ-002", jlink: "JLINK-007", first: "/status/" },
  { key: "CAP-007/@UJ-001", jlink: "JLINK-008", first: "/model/" },
  { key: "CAP-007/@UJ-002", jlink: "JLINK-009", first: "/reference/" },
  { key: "CAP-008/@UJ-001", jlink: "JLINK-010", first: "/model/" },
  { key: "CAP-008/@UJ-002", jlink: "JLINK-011", first: "/model/" },
  { key: "CAP-009/@UJ-002", jlink: "JLINK-013", first: "/adopt/" },
  { key: "CAP-010/@UJ-001", jlink: "JLINK-015", first: "/use-cases/" },
  { key: "CAP-010/@UJ-002", jlink: "JLINK-016", first: "/use-cases/" },
  { key: "CAP-011/@UJ-001", jlink: "JLINK-017", first: "/packs/" },
  { key: "CAP-011/@UJ-002", jlink: "JLINK-018", first: "/packs/" },
  { key: "CAP-012/@UJ-001", jlink: "JLINK-019", first: "/extend/" },
  { key: "CAP-012/@UJ-002", jlink: "JLINK-020", first: "/extend/" },
  { key: "CAP-013/@UJ-001", jlink: "JLINK-021", first: "/model/" },
  { key: "CAP-013/@UJ-002", jlink: "JLINK-022", first: "/packs/" },
  { key: "CAP-014/@UJ-001", jlink: "JLINK-024", first: "/status/" },
  { key: "CAP-014/@UJ-002", jlink: "JLINK-009", first: "/reference/" },
];

test.describe("website capability journeys", () => {
  test.use({ javaScriptEnabled: false });

  for (const journey of journeys) {
    test(`${journey.key} follows the rendered JLINK without a click tour`, async ({ page }) => {
      await page.goto(journey.first);
      if (journey.key === "CAP-014/@UJ-001") {
        const dual = page.locator(
          'aside[data-boundary-id="BOUNDARY-005"] a[data-journey-link-id="JLINK-024"][data-boundary-continuation]',
        );
        await expect(dual).toHaveCount(1);
        await expect(dual).toHaveAttribute("href", "/contributing/#external-ownership");
        return;
      }
      const link = page.locator(`a[data-journey-link-id="${journey.jlink}"]`).first();
      const href = await link.getAttribute("href");
      expect(href, `${journey.key} missing rendered href`).toBeTruthy();
      expect(href?.startsWith("/")).toBeTruthy();
      expect(href?.includes("#")).toBeTruthy();
    });
  }

  test("published runtime is absent", async ({ page }) => {
    await page.goto("/");
    await expect(page.locator("script")).toHaveCount(0);
  });
});
