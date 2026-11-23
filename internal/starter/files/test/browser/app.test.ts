// The build, served as a static host serves it, under the base its <base href> names: the home view
// at the base, with the config's API base; the not-found view at any other path, its link back
// home taken by the router; and a deep link and a reload, which the host answers with the page,
// reach the right view. The page is the foundation's application page, and its header and view sit
// inside the layout sheet's gutters, off the viewport's edges. Every asset loads, under the base,
// which the document's base URI, Vite's base, and the router's agree on.
// Adapted from spike-html-apps (2883d78): test/browser/base.test.ts, on a static host.

import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { after, before, test } from "node:test";

import { launch } from "@standards-lab/web-tooling/browser.js";
import type { Browser, Page } from "@standards-lab/web-tooling/browser.js";

import { click, mounted, openWith, reload, route } from "./support/app.ts";
import { DIST, serveBuild } from "./support/build.ts";
import type { Served } from "./support/build.ts";

/** The API base the build's config.json, from public/, holds. */
const apiBase: string = JSON.parse(readFileSync(`${DIST}/config.json`, "utf8")).apiBase;

let browser: Browser;
let app: Served;

before(async () => {
  browser = await launch();
  app = await serveBuild();
});
after(async () => {
  await browser?.close();
  await app?.stop();
});

/** A page with every response and failed load its network log records, from the first request. */
interface Logged {
  page: Page;
  responses: { url: string; status: number }[];
  failures: string[];
}

/** Opens `path`, under the app's origin, logging its network; closed by the test's `t.after`. */
async function open(path: string, t: { after(fn: () => Promise<void>): void }): Promise<Logged> {
  const responses: Logged["responses"] = [];
  const failures: string[] = [];
  const page = await openWith(browser, "about:blank", async (page) => {
    page.on(
      "Network.responseReceived",
      ({ response }: { response: { url: string; status: number } }) =>
        responses.push({ url: response.url, status: response.status }),
    );
    page.on("Network.loadingFailed", ({ errorText }: { errorText: string }) =>
      failures.push(errorText),
    );
    await page.cdp("Network.enable");
  });
  t.after(() => page.close());
  await page.goto(`${app.origin}${path}`);
  return { page, responses, failures };
}

/** Asserts every load succeeded and was under the base: the page, its assets, and config.json. */
function assertAllUnderBase({ responses, failures }: Logged): void {
  assert.deepEqual(failures, []);
  assert.deepEqual(
    responses.filter(({ status }) => status >= 400),
    [],
  );
  const base = `${app.origin}${app.base}`;
  assert.deepEqual(
    responses.filter(({ url }) => !url.startsWith(base)),
    [],
  );
  assert.ok(
    responses.some(({ url }) => url === `${base}config.json`),
    "config.json loaded",
  );
  assert.ok(
    responses.some(({ url }) => url.startsWith(`${base}assets/`)),
    "the assets loaded under the base",
  );
}

test("the base shows the home view, with the config's API base", async (t) => {
  const logged = await open(app.base, t);
  const { page } = logged;
  await mounted(page, "app-home");
  assert.deepEqual(await route(page), { path: app.base, tag: "app-home" });
  assert.equal(await page.evaluate("document.baseURI"), `${app.origin}${app.base}`);
  assert.equal(await page.evaluate("document.querySelector('app-home code').textContent"), apiBase);
  assertAllUnderBase(logged);
});

test("the application page's header and view sit inside the layout's gutters", async (t) => {
  const { page } = await open(app.base, t);
  await mounted(page, "app-home");
  assert.equal(await page.evaluate("document.body.className"), "page-app");
  const gutters: Record<string, { left: number; right: number }> = await page.evaluate(`
    Object.fromEntries(["header h1", "header .header-actions", "app-home h2"].map((selector) => {
      const box = document.querySelector(selector).getBoundingClientRect();
      return [selector, { left: box.left, right: document.documentElement.clientWidth - box.right }];
    }))`);
  for (const [selector, { left, right }] of Object.entries(gutters)) {
    assert.ok(left > 0, `${selector} is flush against the left edge`);
    assert.ok(right > 0, `${selector} is flush against the right edge`);
  }
});

test("a deep link to a path no route matches shows the not-found view, and a reload keeps it", async (t) => {
  const logged = await open(`${app.base}no/such/page`, t);
  const { page } = logged;
  await mounted(page, "app-not-found");
  assert.deepEqual(await route(page), { path: `${app.base}no/such/page`, tag: "app-not-found" });

  await reload(page);
  await mounted(page, "app-not-found");
  assert.deepEqual(await route(page), { path: `${app.base}no/such/page`, tag: "app-not-found" });
  assertAllUnderBase(logged);
});

test("the not-found view's link goes home through the router, and a reload there keeps home", async (t) => {
  const logged = await open(`${app.base}missing`, t);
  const { page } = logged;
  await mounted(page, "app-not-found");
  // The router intercepts the click, so the page isn't loaded again: the marker survives it.
  await page.evaluate("window.marker = true");
  await click(page, "app-not-found a");
  await mounted(page, "app-home");
  assert.deepEqual(await route(page), { path: app.base, tag: "app-home" });
  assert.equal(await page.evaluate("window.marker === true"), true);

  await reload(page);
  await mounted(page, "app-home");
  assert.deepEqual(await route(page), { path: app.base, tag: "app-home" });
  assertAllUnderBase(logged);
});
