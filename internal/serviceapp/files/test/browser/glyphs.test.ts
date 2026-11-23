// The page's glyphs: the page module draws each from web-icons' module, the very svg it exports, so
// the page shows question-16 on Help, gear-16 on Settings, and x-16 on each dialog's Close, and
// writes no data-icon marker.

import assert from "node:assert/strict";
import { after, before, test } from "node:test";

import gear from "@standards-lab/web-icons/octicon/gear-16";
import question from "@standards-lab/web-icons/octicon/question-16";
import x from "@standards-lab/web-icons/octicon/x-16";
import { launch } from "@standards-lab/web-tooling/browser.js";
import type { Browser } from "@standards-lab/web-tooling/browser.js";

import { mounted } from "./support/app.ts";
import { serveBuild } from "./support/build.ts";
import type { Served } from "./support/build.ts";

const GLYPHS: Record<string, string> = {
  '[commandfor="help"][command="show-modal"]': question,
  '[commandfor="settings"][command="show-modal"]': gear,
  '#help [command="close"]': x,
  '#settings [command="close"]': x,
};

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

const all = Object.fromEntries(Object.keys(GLYPHS).map((selector) => [selector, true]));

test("the page module draws every glyph and writes no marker", async (t) => {
  assert.deepEqual(await glyphs(`${app.origin}${app.base}`, t), { markers: 0, filled: all });
});
