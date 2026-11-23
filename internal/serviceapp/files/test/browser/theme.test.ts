// The theme: a color picked in the Settings dialog's sl-settings is on the root element and stored
// under the foundation's key at once, with no Save, and a reload keeps it; and a non-default theme
// choice stored under that key is on the root element, and the page module's header on the page,
// before the browser first paints. The first-paint case holds every script's fetch, and the same
// build with blocking="render" stripped from its shell paints first, before the theme is applied
// and the page drawn: the case fails without the render-blocking modules, the theme module and the
// page module, which the shell loads in that order.
// A recorder, run ahead of the page's own scripts from Page.addScriptToEvaluateOnNewDocument,
// records in window.flash the first paint, from a PerformanceObserver, and the moments data-scheme
// lands on the root and the header in the body, from MutationObservers; every script's fetch is
// held DELAY ms, so a module that doesn't block rendering runs after the browser has painted.

import assert from "node:assert/strict";
import { after, before, describe, test } from "node:test";

import { launch, until } from "@standards-lab/web-tooling/browser.js";
import type { Browser } from "@standards-lab/web-tooling/browser.js";

import { mounted, openWith, press, reload } from "./support/app.ts";
import { serveBuild } from "./support/build.ts";
import type { Served } from "./support/build.ts";

const KEY = "web-foundation.preferences";

const CHOICE = { color: "catppuccin", space: "compact", type: "sans", scheme: "dark" };

const APPLIED = {
  "data-palette": "catppuccin",
  "data-space": "compact",
  "data-type": "sans",
  "data-scheme": "dark",
};

const ROOTED = `Object.fromEntries(["data-palette", "data-space", "data-type", "data-scheme"].map((name) => [name, document.documentElement.getAttribute(name)]))`;

let browser: Browser;
let app: Served;
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

test("a color picked in Settings is on the root and stored at once, and a reload keeps it", async (t) => {
  const page = await browser.open(`${app.origin}${app.base}`);
  t.after(async () => {
    await page.evaluate("localStorage.clear()").catch(() => {});
    await page.close();
  });
  await mounted(page, "app-home");
  await until(
    page,
    "document.querySelector('[data-keys=\"s\"]')?.hasAttribute('aria-keyshortcuts')",
  );
  await press(page, "s");
  await until(page, "document.querySelector('#settings').open");
  await page.evaluate(
    `document.querySelector("sl-settings").shadowRoot.querySelector('input[name="color"][value="everforest"]').click()`,
  );
  assert.equal(
    await page.evaluate("document.documentElement.getAttribute('data-palette')"),
    "everforest",
  );
  assert.equal(
    await page.evaluate(`JSON.parse(localStorage.getItem(${JSON.stringify(KEY)})).color`),
    "everforest",
  );

  await reload(page);
  await mounted(page, "app-home");
  assert.equal(
    await page.evaluate("document.documentElement.getAttribute('data-palette')"),
    "everforest",
  );
});

describe("the stored choice at first paint", () => {
  const DELAY = 300;

  const RECORDER = `
    window.flash = { paint: null, applied: null, drawn: null };
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
    new MutationObserver(() => {
      if (document.querySelector("body > header")) window.flash.drawn ??= performance.now();
    }).observe(document, { childList: true, subtree: true });
  `;

  async function flash(
    served: Served,
    t: { after(fn: () => Promise<void>): void },
  ): Promise<{ paint: number; applied: number | null; drawn: number | null; blocking: number }> {
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
    const { paint, applied, drawn, blocking } = await flash(app, t);
    t.diagnostic(`applied at ${applied} ms, drawn at ${drawn} ms, first paint at ${paint} ms`);
    assert.equal(blocking, 2, "the shell's theme and page modules block rendering");
    assert.notEqual(applied, null);
    assert.ok(applied! <= paint, `applied at ${applied}, after first paint at ${paint}`);
    assert.notEqual(drawn, null);
    assert.ok(drawn! <= paint, `drawn at ${drawn}, after first paint at ${paint}`);
    assert.ok(paint >= DELAY, `first paint at ${paint} didn't wait on the held fetch`);
  });

  test("without blocking=render, the page paints the defaults first: the case above can fail", async (t) => {
    const { paint, applied, drawn, blocking } = await flash(plain, t);
    t.diagnostic(`applied at ${applied} ms, drawn at ${drawn} ms, first paint at ${paint} ms`);
    assert.equal(blocking, 0, "the stripped shell blocks rendering on nothing");
    assert.ok(
      applied !== null && applied > paint,
      `applied at ${applied}, first paint at ${paint}`,
    );
    assert.ok(drawn !== null && drawn > paint, `drawn at ${drawn}, first paint at ${paint}`);
  });
});
