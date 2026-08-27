import { expect, test } from "@playwright/test";
import {
  assertKeyboardOrderAndBounds,
  assertLocalOverflow,
  assertRequiredSurface,
  canonicalRoutes,
  settleLayout,
  viewports,
} from "./public-site-helpers";

for (const viewport of viewports) {
  test.describe(viewport.name, () => {
    test.use({ viewport: { width: viewport.width, height: viewport.height }, javaScriptEnabled: false });

    for (const route of canonicalRoutes) {
      test(`${route} is complete without JavaScript`, async ({ page }) => {
        await page.goto(route);
        await settleLayout(page);
        await assertRequiredSurface(page, route);
        await assertKeyboardOrderAndBounds(page, route);
        await assertLocalOverflow(page, route);
      });
    }
  });
}

test.describe("200 percent text relayout", () => {
  test.use({ viewport: { width: 360, height: 800 }, javaScriptEnabled: false });

  for (const route of canonicalRoutes) {
    test(`${route} reflows at actual 200 percent root text`, async ({ page }) => {
      await page.goto(route);
      await settleLayout(page);
      const baseline = await page.evaluate(() => Number.parseFloat(getComputedStyle(document.documentElement).fontSize));
      await page.route("**/*", async (requestRoute) => {
        if (requestRoute.request().resourceType() !== "document") {
          await requestRoute.continue();
          return;
        }
        const response = await requestRoute.fetch();
        const body = (await response.text()).replace(
          "</head>",
          "<style data-test-root-font>html { font-size: 200% !important; }</style></head>",
        );
        await requestRoute.fulfill({ response, body });
      });
      await page.reload();
      await settleLayout(page);
      const enlarged = await page.evaluate(() => Number.parseFloat(getComputedStyle(document.documentElement).fontSize));
      expect(Math.abs(enlarged - baseline * 2)).toBeLessThanOrEqual(0.01);
      await assertRequiredSurface(page, route);
      await assertKeyboardOrderAndBounds(page, route);
      await assertLocalOverflow(page, route);
    });
  }
});
