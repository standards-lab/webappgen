// The theme: the header's Settings button, and the S key through the foundation's keys.js, open
// the theme menu, which the render-blocking theme module defines and which lists every profile the
// full entry registers; and a non-default theme choice stored under the foundation's key is on the
// root element before the browser first paints. The first-paint case holds every script's fetch,
// and the same build with blocking="render" stripped from its page paints the defaults first: the
// case fails without the render-blocking module.
// Adapted from spike-html-apps (2883d78): test/browser/theme.test.ts, on the full entry.

import assert from "node:assert/strict";
import { after, before, describe, test } from "node:test";

import { launch, until } from "@standards-lab/web-tooling/browser.js";
import type { Browser } from "@standards-lab/web-tooling/browser.js";

import { click, mounted, openWith } from "./support/app.ts";
import { serveBuild } from "./support/build.ts";
import type { Served } from "./support/build.ts";

/** The foundation's localStorage key for the choice (its preferences.js KEY). */
const KEY = "web-foundation.preferences";

/** A choice off every facet's default, which only the full entry registers, as the theme menu
 * stores it. */
const CHOICE = { color: "catppuccin", space: "compact", type: "sans", scheme: "dark" };

/** CHOICE as the root element carries it. */
const APPLIED = {
  "data-palette": "catppuccin",
  "data-space": "compact",
  "data-type": "sans",
  "data-scheme": "dark",
};

/** The open menu's options, each section's radio values in order. */
const OPTIONS = `Object.fromEntries([...document.querySelectorAll('sl-theme-menu [data-section]')].map((section) => [section.dataset.section, [...section.querySelectorAll('input')].map((input) => input.value)]))`;

/** Whether the theme menu's dialog is open. */
const OPEN = "document.querySelector('sl-theme-menu dialog')?.open === true";

/** The root element's theme attributes. */
const ROOTED = `Object.fromEntries(["data-palette", "data-space", "data-type", "data-scheme"].map((name) => [name, document.documentElement.getAttribute(name)]))`;

let browser: Browser;
let app: Served;
/** The build with every blocking="render" stripped from its page. */
let plain: Served;

before(async () => {
  browser = await launch();
  app = await serveBuild();
  plain = await serveBuild({ page: (html) => html.replaceAll(' blocking="render"', "") });
});
after(async () => {
  await browser?.close();
  await app?.stop();
  await plain?.stop();
});

test("the header's Settings button opens the theme menu, listing every built-in profile", async (t) => {
  const page = await browser.open(`${app.origin}${app.base}`);
  t.after(() => page.close());
  await mounted(page, "app-home");
  await click(page, "header button");
  await until(page, OPEN);
  assert.deepEqual(await page.evaluate(OPTIONS), {
    color: ["zen", "github", "catppuccin", "everforest"],
    scheme: ["system", "light", "dark"],
    space: ["regular", "compact"],
    type: ["mono", "sans"],
  });
});

test("the S key opens the Settings menu", async (t) => {
  const page = await browser.open(`${app.origin}${app.base}`);
  t.after(() => page.close());
  await mounted(page, "app-home");
  // keys.js marks each element it binds, so the binding is live before the key is pressed.
  await until(page, "document.querySelector('header button')?.hasAttribute('aria-keyshortcuts')");
  assert.equal(await page.evaluate(OPEN), false);
  for (const type of ["keyDown", "keyUp"]) {
    await page.cdp("Input.dispatchKeyEvent", { type, key: "s", code: "KeyS", text: "s" });
  }
  await until(page, OPEN);
});

// The first paint, against the moment the stored choice reaches the root. Every script's fetch is
// held DELAY ms, so a module that doesn't block rendering runs after the browser has painted.
// The recorder runs ahead of the page's own scripts, from Page.addScriptToEvaluateOnNewDocument:
// window.flash records, in ms since navigation, first-paint from a PerformanceObserver and the
// moment data-scheme lands from a MutationObserver.
describe("the stored choice at first paint", () => {
  const DELAY = 300;

  const RECORDER = `
    window.flash = { paint: null, applied: null };
    new PerformanceObserver((list) => {
      for (const entry of list.getEntries()) {
        if (entry.name === "first-paint") window.flash.paint ??= entry.startTime;
      }
    }).observe({ type: "paint", buffered: true });
    new MutationObserver(() => {
      if (document.documentElement.hasAttribute("data-scheme")) {
        window.flash.applied ??= performance.now();
      }
    }).observe(document.documentElement, { attributes: true });
  `;

  /** Stores CHOICE on the served build's origin, then loads its base afresh with the recorder
   * installed and every script's fetch held, resolving to what the recorder saw and how many of
   * the page's scripts block rendering. */
  async function flash(
    served: Served,
    t: { after(fn: () => Promise<void>): void },
  ): Promise<{ paint: number; applied: number | null; blocking: number }> {
    const url = `${served.origin}${served.base}`;
    const seed = await browser.open(url);
    t.after(() => seed.close());
    await seed.evaluate(
      `localStorage.setItem(${JSON.stringify(KEY)}, ${JSON.stringify(JSON.stringify(CHOICE))})`,
    );
    const page = await openWith(browser, "about:blank", async (page) => {
      await page.cdp("Page.addScriptToEvaluateOnNewDocument", { source: RECORDER });
      page.on("Fetch.requestPaused", ({ requestId }: { requestId: string }) => {
        setTimeout(() => page.cdp("Fetch.continueRequest", { requestId }).catch(() => {}), DELAY);
      });
      await page.cdp("Fetch.enable", { patterns: [{ urlPattern: "*/assets/*.js" }] });
    });
    t.after(async () => {
      await page.evaluate("localStorage.clear()").catch(() => {});
      await page.close();
    });
    await page.goto(url);
    await until(page, "window.flash.paint !== null", 5000);
    await mounted(page, "app-home");
    assert.deepEqual(await page.evaluate(ROOTED), APPLIED);
    return page.evaluate(
      "({ ...window.flash, blocking: document.querySelectorAll('script[blocking~=render]').length })",
    );
  }

  test("is on the root element before the browser paints", async (t) => {
    const { paint, applied, blocking } = await flash(app, t);
    t.diagnostic(`applied at ${applied} ms, first paint at ${paint} ms`);
    assert.equal(blocking, 1, "the page's theme module blocks rendering");
    assert.notEqual(applied, null);
    assert.ok(applied! <= paint, `applied at ${applied}, after first paint at ${paint}`);
    assert.ok(paint >= DELAY, `first paint at ${paint} didn't wait on the held fetch`);
  });

  test("without blocking=render, the page paints the defaults first: the case above can fail", async (t) => {
    const { paint, applied, blocking } = await flash(plain, t);
    t.diagnostic(`applied at ${applied} ms, first paint at ${paint} ms`);
    assert.equal(blocking, 0, "the stripped page blocks rendering on nothing");
    assert.ok(
      applied !== null && applied > paint,
      `applied at ${applied}, first paint at ${paint}`,
    );
  });
});
