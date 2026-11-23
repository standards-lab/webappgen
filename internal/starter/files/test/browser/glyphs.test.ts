// The page's glyphs: web-icons' iconFill() replaces every data-icon marker the page writes with its
// icon's markup, the very svg the icon's module exports, in the build and in Vite's dev server
// alike, so the page shows question-16 on Help, gear-16 on Settings, and x-16 on each dialog's
// Close, and no marker is left.

import assert from "node:assert/strict";
import { mkdtemp, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { after, before, test } from "node:test";

import gear from "@standards-lab/web-icons/octicon/gear-16";
import question from "@standards-lab/web-icons/octicon/question-16";
import x from "@standards-lab/web-icons/octicon/x-16";
import { launch } from "@standards-lab/web-tooling/browser.js";
import type { Browser } from "@standards-lab/web-tooling/browser.js";
import { createServer } from "vite";
import type { ViteDevServer } from "vite";

import { mounted } from "./support/app.ts";
import { baseOf, serveBuild } from "./support/build.ts";
import type { Served } from "./support/build.ts";

/** Each glyph button, by its selector, and the markup its glyph is. */
const GLYPHS: Record<string, string> = {
  '[commandfor="help"][command="show-modal"]': question,
  '[commandfor="settings"][command="show-modal"]': gear,
  '#help [command="close"]': x,
  '#settings [command="close"]': x,
};

let browser: Browser;
let app: Served;
let dev: ViteDevServer;
let devOrigin: string;
let cacheDir: string;

before(async () => {
  browser = await launch();
  app = await serveBuild();
  cacheDir = await mkdtemp(join(tmpdir(), "vite-dev-"));
  dev = await createServer({
    cacheDir,
    logLevel: "error",
    server: { host: "127.0.0.1", port: 0, hmr: false },
  });
  await dev.listen();
  const address = dev.httpServer?.address();
  if (!address || typeof address === "string") throw new Error("the dev server has no port");
  devOrigin = `http://127.0.0.1:${address.port}`;
});
after(async () => {
  await browser?.close();
  await app?.stop();
  await dev?.close();
  await rm(cacheDir, { recursive: true, force: true });
});

/** Opens `url` and reads, once the home view is mounted, how many markers are left and whether
 * each glyph button holds exactly its icon's svg. */
async function glyphs(
  url: string,
  t: { after(fn: () => Promise<void>): void },
): Promise<{ markers: number; filled: Record<string, boolean> }> {
  const page = await browser.open(url);
  t.after(() => page.close());
  await mounted(page, "app-home");
  return page.evaluate(`(() => {
    const filled = {};
    for (const [selector, markup] of Object.entries(${JSON.stringify(GLYPHS)})) {
      const svgs = document.querySelector(selector).querySelectorAll("svg");
      const template = document.createElement("template");
      template.innerHTML = markup;
      filled[selector] =
        svgs.length === 1 && template.content.firstElementChild.isEqualNode(svgs[0]);
    }
    return { markers: document.querySelectorAll("[data-icon]").length, filled };
  })()`);
}

/** Every glyph button filled. */
const all = Object.fromEntries(Object.keys(GLYPHS).map((selector) => [selector, true]));

test("the build fills every glyph and leaves no marker", async (t) => {
  assert.deepEqual(await glyphs(`${app.origin}${app.base}`, t), { markers: 0, filled: all });
});

test("the dev server fills every glyph and leaves no marker", async (t) => {
  assert.deepEqual(await glyphs(`${devOrigin}${baseOf()}`, t), { markers: 0, filled: all });
});
