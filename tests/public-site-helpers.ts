import { expect, Page } from "@playwright/test";

export const canonicalRoutes = [
  "/", "/evaluate/", "/model/", "/adopt/", "/use-cases/",
  "/packs/", "/extend/", "/reference/", "/status/", "/contributing/",
] as const;

export const viewports = [
  { name: "narrow", width: 360, height: 800 },
  { name: "medium", width: 768, height: 1024 },
  { name: "wide", width: 1440, height: 1000 },
] as const;

const focusableSelector = [
  "a[href]", "summary", "[data-overflow-region][tabindex=\"0\"]",
].join(",");

export async function settleLayout(page: Page): Promise<void> {
  await page.waitForLoadState("load");
  await expect.poll(() => page.evaluate(() => document.fonts.status)).toBe("loaded");
  // JavaScript is disabled in the document, so two 60 Hz frame intervals are
  // observed from the runner after font readiness instead of injecting page code.
  await page.waitForTimeout(34);
}

export async function assertRequiredSurface(page: Page, route: string): Promise<void> {
  await expect(page.locator("[data-backstop-wordmark]")).toBeVisible();
  await expect(page.locator('nav[aria-label="Primary"] a')).toHaveCount(7);
  await expect(page.locator('nav[aria-label="Utility"] a')).toHaveCount(2);
  await expect(page.locator("main#main")).toHaveAttribute("data-page-route", route);
  await expect(page.locator("[data-page-hero] [data-page-question]")).toBeVisible();
  await expect(page.locator("footer")).toBeVisible();
  const horizontalOverflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
  expect(horizontalOverflow, `${route} document overflow`).toBeLessThanOrEqual(1);
}

export async function assertContentCompleteness(page: Page, route: string): Promise<void> {
  const contract = page.locator("[data-required-blocks]");
  await expect(contract, `${route} rendered required-block contract cardinality`).toHaveCount(1);
  const rawRequired = await contract.getAttribute("data-required-blocks");
  expect(rawRequired, `${route} rendered required-block contract`).toBeTruthy();
  const required = rawRequired!.split(",").map((value) => value.trim()).filter(Boolean);
  expect(required.length, `${route} required-block count`).toBeGreaterThan(0);

  for (const id of required) {
    const heading = page.locator(`#${id}`);
    await expect(heading, `${route} #${id} cardinality`).toHaveCount(1);
    await expect(heading, `${route} #${id} visibility`).toBeVisible();

    const state = await heading.evaluate((element) => {
      let sibling = element.nextElementSibling as HTMLElement | null;
      let text = "";
      let visibleBlocks = 0;
      let firstContentTop: number | null = null;
      const headingRect = element.getBoundingClientRect();
      while (sibling && sibling.tagName !== "H2") {
        const style = getComputedStyle(sibling);
        const rect = sibling.getBoundingClientRect();
        const visible = style.display !== "none" && style.visibility !== "hidden" && rect.width > 0 && rect.height > 0;
        if (visible) {
          visibleBlocks += 1;
          text += ` ${sibling.textContent ?? ""}`;
          if (firstContentTop === null) firstContentTop = rect.top;
        }
        sibling = sibling.nextElementSibling as HTMLElement | null;
      }
      return {
        textLength: text.replace(/\s+/g, " ").trim().length,
        visibleBlocks,
        firstContentGap: firstContentTop === null ? null : firstContentTop - headingRect.bottom,
      };
    });

    expect(state.visibleBlocks, `${route} #${id} visible content blocks`).toBeGreaterThan(0);
    expect(state.textLength, `${route} #${id} substantive text`).toBeGreaterThanOrEqual(80);
    expect(state.firstContentGap, `${route} #${id} first content`).not.toBeNull();
    expect(state.firstContentGap!, `${route} #${id} unexplained vertical gap`).toBeLessThanOrEqual(160);
  }

  if (route === "/") {
    await expect(page.locator("[data-home-capabilities] > article"), "home capability summaries").toHaveCount(3);
    await expect(page.locator("[data-home-paths] > article"), "home decision paths").toHaveCount(3);
    await expect(page.locator("[data-home-gate-proof]"), "home gate proof").toContainText("backstop gate");
    for (const href of ["/evaluate/", "/model/", "/adopt/"]) {
      await expect(page.locator(`[data-page-content] a[href="${href}"]`).first(), `home path ${href}`).toBeVisible();
    }
    const homeTextLength = await page.locator("[data-page-content]").evaluate((element) => (element.textContent ?? "").replace(/\s+/g, " ").trim().length);
    expect(homeTextLength, "home substantive content length").toBeGreaterThanOrEqual(1200);
  }
}

export async function assertKeyboardOrderAndBounds(page: Page, route: string): Promise<void> {
  const expected = await page.locator(focusableSelector).evaluateAll((elements) => elements.filter((element) => {
    const style = getComputedStyle(element);
    const rect = element.getBoundingClientRect();
    return style.visibility !== "hidden" && style.display !== "none" && rect.width > 0 && rect.height > 0;
  }).length);
  expect(expected, `${route} focusable count`).toBeGreaterThan(0);

  await page.locator("body").press("Tab");
  await page.waitForTimeout(34);
  const seen: string[] = [];
  for (let index = 0; index < expected; index += 1) {
    await page.locator(":focus").scrollIntoViewIfNeeded();
    const state = await page.evaluate(() => {
      const active = document.activeElement as HTMLElement | null;
      if (!active) return null;
      const rect = active.getBoundingClientRect();
      const visibleTop = Math.max(0, rect.top);
      const visibleBottom = Math.min(innerHeight, rect.bottom);
      const fragment = [...active.getClientRects()].find((candidate) => (
        candidate.right > 0 && candidate.left < innerWidth && candidate.bottom > 0 && candidate.top < innerHeight
      )) ?? rect;
      const fragmentVisibleTop = Math.max(0, fragment.top);
      const fragmentVisibleBottom = Math.min(innerHeight, fragment.bottom);
      const centerX = Math.max(0, Math.min(innerWidth - 1, fragment.left + fragment.width / 2));
      const centerY = Math.max(0, Math.min(innerHeight - 1, fragmentVisibleTop + (fragmentVisibleBottom - fragmentVisibleTop) / 2));
      const top = document.elementFromPoint(centerX, centerY);
      return {
        identity: active.getAttribute("href") ?? active.id ?? active.tagName,
        left: rect.left, top: rect.top, right: rect.right, bottom: rect.bottom,
        visibleHeight: Math.max(0, visibleBottom - visibleTop),
        topmost: top === active || active.contains(top),
        topIdentity: (top as HTMLElement | null)?.getAttribute("href") ?? (top as HTMLElement | null)?.id ?? top?.tagName ?? "none",
      };
    });
    expect(state, `${route} active element`).not.toBeNull();
    expect(state!.left, `${route} focus left`).toBeGreaterThanOrEqual(-1);
    expect(state!.right, `${route} focus right`).toBeLessThanOrEqual(page.viewportSize()!.width + 1);
    expect(state!.visibleHeight, `${route} ${state!.identity} visible focus area`).toBeGreaterThan(1);
    expect(state!.topmost, `${route} ${state!.identity} occluded by ${state!.topIdentity}`).toBeTruthy();
    seen.push(state!.identity);
    await page.keyboard.press("Tab");
    await page.waitForTimeout(34);
  }
  expect(seen.length, `${route} traversed focusables`).toBe(expected);
}

export async function assertLocalOverflow(page: Page, route: string): Promise<void> {
  const regions = page.locator('[data-overflow-region][role="region"][aria-labelledby][tabindex="0"]');
  for (let index = 0; index < await regions.count(); index += 1) {
    const region = regions.nth(index);
    const before = await region.evaluate((element) => element.scrollLeft);
    await region.focus();
    await page.keyboard.press("ArrowRight");
    await page.keyboard.press("End");
    const state = await region.evaluate((element) => ({
      scrollLeft: element.scrollLeft,
      max: element.scrollWidth - element.clientWidth,
    }));
    expect(state.scrollLeft, `${route} overflow region ${index}`).toBeGreaterThanOrEqual(before);
    expect(state.max, `${route} overflow max ${index}`).toBeGreaterThanOrEqual(0);
  }
}
