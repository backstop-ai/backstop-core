import { expect, Page } from "@playwright/test";

export const canonicalRoutes = [
  "/", "/evaluate/", "/model/", "/adopt/", "/use-cases/",
  "/pack/examples/", "/pack/guide/", "/reference/", "/status/", "/contributing/",
] as const;

export const viewports = [
  { name: "narrow", width: 360, height: 800 },
  { name: "medium", width: 768, height: 1024 },
  { name: "wide", width: 1440, height: 1000 },
] as const;

const focusableSelector = [
  "a[href]", "summary", "[data-overflow-region][tabindex=\"0\"]",
].join(",");

const primaryNavigation = [
  ["Evaluate", "/evaluate/"], ["Model", "/model/"], ["Adopt", "/adopt/"],
  ["Pack", "/pack/"],
] as const;

const utilityNavigation = [
  ["Contributing", "/contributing/"],
] as const;

export async function settleLayout(page: Page): Promise<void> {
  await page.waitForLoadState("load");
  await expect.poll(() => page.evaluate(() => document.fonts.status)).toBe("loaded");
  // JavaScript is disabled in the document, so two 60 Hz frame intervals are
  // observed from the runner after font readiness instead of injecting page code.
  await page.waitForTimeout(34);
}

export async function revealPrimaryNavigation(page: Page): Promise<void> {
  const toggle = page.locator(".site-nav-toggle");
  if (await toggle.count() === 0) return;
  const box = await toggle.boundingBox();
  if (!box || box.width < 1 || box.height < 1) return;
  const primary = page.locator('nav[aria-label="Primary"]');
  if (await primary.isVisible()) return;
  await toggle.click();
  await expect(primary, "mobile navigation opens from the menu control").toBeVisible();
}

async function startKeyboardAtSkipLink(page: Page): Promise<void> {
  const skip = page.locator(".skip-link");
  await expect(skip, "skip link is present for keyboard origin").toHaveCount(1);
  await skip.focus({ force: true });
}

async function assertDocumentDoesNotOverflow(page: Page, route: string): Promise<void> {
  const overflowState = await page.evaluate(() => {
    const viewportWidth = document.documentElement.clientWidth;
    const horizontalOverflow = document.documentElement.scrollWidth - viewportWidth;
    const offenders = [...document.querySelectorAll<HTMLElement>("body *")]
      .map((element) => {
        const rect = element.getBoundingClientRect();
        const style = getComputedStyle(element);
        return {
          tag: element.tagName.toLowerCase(),
          id: element.id,
          classes: element.className,
          left: rect.left,
          right: rect.right,
          width: rect.width,
          overflowX: style.overflowX,
          text: (element.textContent ?? "").replace(/\s+/g, " ").trim().slice(0, 120),
        };
      })
      .filter((item) => item.right > viewportWidth + 1 || item.left < -1)
      .sort((a, b) => Math.max(b.right - viewportWidth, -b.left) - Math.max(a.right - viewportWidth, -a.left))
      .slice(0, 12);
    return { horizontalOverflow, viewportWidth, offenders };
  });
  expect(
    overflowState.horizontalOverflow,
    `${route} document overflow; viewport=${overflowState.viewportWidth}; offenders=${JSON.stringify(overflowState.offenders)}`,
  ).toBeLessThanOrEqual(1);
}

export async function assertRequiredSurface(page: Page, route: string): Promise<void> {
  const wordmark = page.locator("[data-backstop-wordmark]");
  await expect(wordmark).toBeVisible();
  const wordmarkParts = await wordmark.locator(":scope > span").allTextContents();
  expect(wordmarkParts.map((part) => part.trim()), `${route} complete visible wordmark parts`).toEqual(["./b", "backstop", ".sh"]);
  expect(`${wordmarkParts[0].trim()} ${wordmarkParts.slice(1).join("").trim()}`, `${route} normalized visible wordmark`).toBe("./b backstop.sh");
  await assertDocumentDoesNotOverflow(page, route);
  await revealPrimaryNavigation(page);
  const primary = page.locator('nav[aria-label="Primary"] a');
  const utility = page.locator('nav[aria-label="Utility"] a');
  await expect(primary).toHaveCount(primaryNavigation.length);
  await expect(utility).toHaveCount(utilityNavigation.length);
  for (const [index, [label, href]] of primaryNavigation.entries()) {
    await expect(primary.nth(index), `${route} primary ${label}`).toHaveText(label);
    await expect(primary.nth(index), `${route} primary ${label}`).toHaveAttribute("href", href);
    await expect(primary.nth(index), `${route} primary ${label} visible`).toBeVisible();
  }
  for (const [index, [label, href]] of utilityNavigation.entries()) {
    await expect(utility.nth(index), `${route} utility ${label}`).toHaveText(label);
    await expect(utility.nth(index), `${route} utility ${label}`).toHaveAttribute("href", href);
    await expect(utility.nth(index), `${route} utility ${label} visible`).toBeVisible();
  }
  await expect(page.locator("main#main")).toHaveAttribute("data-page-route", route);
  if (route === "/") {
    await expect(page.locator("[data-page-hero] h1")).toContainText("Define the work.");
    await expect(page.locator("[data-home-gate-proof]"), "home gate proof").toContainText("backstop gate");
  } else {
    await expect(page.locator("[data-page-hero] [data-page-question]")).toBeVisible();
  }
  await expect(page.locator("footer")).toBeVisible();
  await assertDocumentDoesNotOverflow(page, route);
}

export async function assertContentCompleteness(page: Page, route: string): Promise<void> {
  const contract = page.locator("[data-required-blocks]");
  await expect(contract, `${route} rendered required-block contract cardinality`).toHaveCount(1);
  const rawRequired = await contract.getAttribute("data-required-blocks");
  expect(rawRequired, `${route} rendered required-block contract`).toBeTruthy();
  const required = rawRequired!.split(",").map((value) => value.trim()).filter(Boolean);
  expect(required.length, `${route} required-block count`).toBeGreaterThan(0);

  for (const id of required) {
    const block = page.locator(`#${id}`);
    await expect(block, `${route} #${id} cardinality`).toHaveCount(1);

    const inCanonicalAnchors = await block.evaluate((element) => element.closest(".canonical-anchors") !== null);
    if (inCanonicalAnchors) {
      const hidden = await block.evaluate((element) => {
        const container = element.closest(".canonical-anchors") as HTMLElement | null;
        if (!container) return false;
        const rect = container.getBoundingClientRect();
        const clip = getComputedStyle(container).clip;
        return rect.width <= 1 && rect.height <= 1 && clip === "rect(0px, 0px, 0px, 0px)";
      });
      expect(hidden, `${route} #${id} not visitor-visible`).toBe(true);
      continue;
    }

    await expect(block, `${route} #${id} visibility`).toBeVisible();

    const state = await block.evaluate((element) => {
      if (!/^H[1-6]$/.test(element.tagName)) {
        const rect = element.getBoundingClientRect();
        const text = (element.textContent ?? "").replace(/\s+/g, " ").trim();
        const visibleBlocks = [...element.children].filter((child) => {
          const style = getComputedStyle(child);
          const childRect = child.getBoundingClientRect();
          return style.display !== "none" && style.visibility !== "hidden" && childRect.width > 0 && childRect.height > 0;
        }).length;
        return { textLength: text.length, visibleBlocks, firstContentGap: rect.height > 0 ? 0 : null };
      }
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

  if (route === "/model/") {
    const container = page.locator(".canonical-anchors");
    await expect(container, `${route} canonical-anchors cardinality`).toHaveCount(1);
    const hiddenAnchorIds = [
      "intent-artifacts", "work-tracks", "bounded-execution", "recipes", "gates-and-policy",
      "waivers", "capabilities-and-journeys", "provenance-and-verification", "harness-integration",
      "product-category", "delivery-lifecycle",
    ];
    for (const anchorId of hiddenAnchorIds) {
      const anchor = container.locator(`#${anchorId}`);
      await expect(anchor, `${route} #${anchorId} cardinality`).toHaveCount(1);
      const hidden = await anchor.evaluate((element) => {
        const host = element.closest(".canonical-anchors") as HTMLElement | null;
        if (!host) return false;
        const rect = host.getBoundingClientRect();
        const clip = getComputedStyle(host).clip;
        return rect.width <= 1 && rect.height <= 1 && clip === "rect(0px, 0px, 0px, 0px)";
      });
      expect(hidden, `${route} #${anchorId} not visitor-visible`).toBe(true);
    }
    expect(required.filter((id) => id === "delivery-lifecycle").length, `${route} hidden required block`).toBe(1);
    expect(required.filter((id) => id !== "delivery-lifecycle").length, `${route} visitor required blocks`).toBe(3);
  }

  if (route === "/") {
    const sections = page.locator("[data-home-system-section]");
    await expect(sections, "home canonical system sections").toHaveCount(3);
    for (const [index, [number, title]] of [["01", "Define the work"], ["02", "Enforce your standards"], ["03", "Detect drift"]].entries()) {
      await expect(sections.nth(index).locator(".section-number"), `home system section ${index + 1} number`).toHaveText(number);
      await expect(sections.nth(index).locator(".section-kicker"), `home system section ${index + 1} title`).toHaveText(title);
    }
    const modes = page.locator("[data-home-modes] > article");
    const expectedModes = ["Full framework", "Artifact workflow", "Standards enforcement", "Deterministic scaffolding"];
    await expect(modes, "home canonical modes").toHaveCount(expectedModes.length);
    for (const [index, expected] of expectedModes.entries()) await expect(modes.nth(index)).toContainText(expected);
    const homeText = await page.locator("[data-page-content]").innerText();
    for (const forbidden of ["What failure does Backstop prevent?", "Why Backstop", "Choose your path", "EvaluateFailure fit", "UnderstandArtifacts", "AdoptOne real standard"]) {
      expect(homeText.replace(/\s+/g, ""), `home forbidden scaffold ${forbidden}`).not.toContain(forbidden.replace(/\s+/g, ""));
    }
    const homeTextLength = await page.locator("[data-page-content]").evaluate((element) => (element.textContent ?? "").replace(/\s+/g, " ").trim().length);
    expect(homeTextLength, "home substantive content length").toBeGreaterThanOrEqual(1200);
  }
}

export async function assertKeyboardOrderAndBounds(page: Page, route: string): Promise<void> {
  await revealPrimaryNavigation(page);
  await startKeyboardAtSkipLink(page);
  const expectedCanonicalFocusOrder = [
    "Home:/",
    ...primaryNavigation.map(([label, href]) => `Primary:${label}:${href}`),
    ...utilityNavigation.map(([label, href]) => `Utility:${label}:${href}`),
  ];
  const expected = await page.locator(focusableSelector).evaluateAll((elements) => elements.filter((element) => {
    const style = getComputedStyle(element);
    const rect = element.getBoundingClientRect();
    return style.visibility !== "hidden" && style.display !== "none" && rect.width > 0 && rect.height > 0;
  }).length);
  expect(expected, `${route} focusable count`).toBeGreaterThan(0);

  await page.waitForTimeout(34);
  const seen: string[] = [];
  const seenCanonical: string[] = [];
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
      const href = active.getAttribute("href") ?? "";
      const navigation = active.closest<HTMLElement>('nav[aria-label="Primary"], nav[aria-label="Utility"]');
      const navigationLabel = navigation?.getAttribute("aria-label");
      const canonicalIdentity = active.matches("[data-backstop-wordmark]")
        ? `Home:${href}`
        : navigation && navigationLabel
          ? `${navigationLabel}:${(active.textContent ?? "").replace(/\s+/g, " ").trim()}:${href}`
          : null;
      return {
        identity: href || active.id || active.tagName,
        canonicalIdentity,
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
    if (state!.canonicalIdentity) seenCanonical.push(state!.canonicalIdentity);
    await page.keyboard.press("Tab");
    await page.waitForTimeout(34);
  }
  expect(seen.length, `${route} traversed focusables`).toBe(expected);
  for (const identity of expectedCanonicalFocusOrder) {
    expect(seenCanonical.filter((candidate) => candidate === identity), `${route} keyboard encounter ${identity}`).toHaveLength(1);
  }
  expect(seenCanonical, `${route} ordered Home + 4 primary + 1 utility keyboard traversal`).toEqual(expectedCanonicalFocusOrder);
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
