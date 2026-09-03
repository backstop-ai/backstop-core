import { expect, test } from "@playwright/test";
import {
  assertContentCompleteness,
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
        await assertContentCompleteness(page, route);
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
      await assertContentCompleteness(page, route);
      await assertKeyboardOrderAndBounds(page, route);
      await assertLocalOverflow(page, route);
    });
  }
});

test.describe("Evaluate paper surface", () => {
  test.use({ viewport: { width: 1440, height: 900 }, javaScriptEnabled: false });

  test("/evaluate/ uses paper presentation and approved hero contract", async ({ page }) => {
    await page.goto("/evaluate/");
    await settleLayout(page);

    const main = page.locator('main#main[data-page-kind="evaluation"]');
    await expect(main).toHaveCount(1);

    await expect(page.locator('link[rel="stylesheet"][href="/assets/css/backstop-tokens.css"]')).toHaveCount(1);
    await expect(page.locator('meta[name="theme-color"][content="#0c0d0d"]')).toHaveCount(1);
    await expect(page.locator('meta[name="color-scheme"][content="dark"]')).toHaveCount(0);

    const bodyBackground = await page.evaluate(() => getComputedStyle(document.body).backgroundColor);
    const paperResolved = await page.evaluate((color) => {
      const probe = document.createElement("div");
      probe.style.color = color;
      document.body.appendChild(probe);
      const resolved = getComputedStyle(probe).color;
      probe.remove();
      return resolved;
    }, `var(--color-paper)`);
    expect(bodyBackground).toBe(paperResolved);

    await expect(page.locator("[data-page-question]")).toHaveText("Your agent already writes the code.");
    await expect(page.locator(".page-boundary")).toHaveText("Backstop helps you ship confidently.");

    const requiredBlocks = await page.locator('[data-required-blocks]').getAttribute("data-required-blocks");
    expect(requiredBlocks).toBe("working-state,failure-fit,fit-decision");

    const failedVerdict = page.locator(".failed-verdict");
    await expect(failedVerdict).toBeVisible();
    const verdictBackground = await failedVerdict.evaluate((el) => getComputedStyle(el).backgroundColor);
    expect(verdictBackground).not.toBe(bodyBackground);

    const journeyLinks = page.locator("[data-page-content] a[data-journey-link-id]");
    await expect(journeyLinks).toHaveCount(2);
    await expect(page.locator('a[data-journey-link-id="JLINK-002"]')).toHaveAttribute("href", "/model/#operating-model");
    await expect(page.locator('a[data-journey-link-id="JLINK-002"]')).toHaveText("See the operating model");
    await expect(page.locator('a[data-journey-link-id="JLINK-004"]')).toHaveAttribute("href", "/adopt/#install");
    await expect(page.locator('a[data-journey-link-id="JLINK-004"]')).toHaveText("Install Backstop");
    await expect(page.locator('[data-page-content] a[href*="/status/"]')).toHaveCount(0);
    await expect(page.locator('[data-page-content] a[href*="/reference/"]')).toHaveCount(0);
  });

  test("/model/ keeps dark inner-page presentation", async ({ page }) => {
    await page.goto("/model/");
    await settleLayout(page);
    await expect(page.locator('meta[name="color-scheme"][content="dark"]')).toHaveCount(1);
    await expect(page.locator('link[rel="stylesheet"][href="/assets/css/backstop-tokens.css"]')).toHaveCount(0);
  });
});
