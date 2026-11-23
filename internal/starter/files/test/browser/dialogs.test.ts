// The header's Help and Settings: ? and S, bound through the foundation's keys.js by each glyph
// button's data-keys, open the Help and Settings dialogs as modals; Help, fit to its content,
// lists both keys in its sl-shortcuts, and Settings, wide, lists every built-in profile in its
// sl-settings. Escape, and each dialog's Close, close it with no script of the app's, and focus
// returns to the button that opened it, whether focused before its key or clicked.

import assert from "node:assert/strict";
import { after, before, test } from "node:test";

import { launch, until } from "@standards-lab/web-tooling/browser.js";
import type { Browser, Page } from "@standards-lab/web-tooling/browser.js";

import { click, mounted, press } from "./support/app.ts";
import { serveBuild } from "./support/build.ts";
import type { Served } from "./support/build.ts";

const DIALOGS = [
  { id: "help", key: "?", name: "Help", width: "fit" },
  { id: "settings", key: "s", name: "Settings", width: "wide" },
] as const;

type Dialog = (typeof DIALOGS)[number];

const opener = (dialog: Dialog): string =>
  `document.querySelector('body > header .header-actions [commandfor="${dialog.id}"][command="show-modal"]')`;

const closer = (dialog: Dialog): string =>
  `document.querySelector('#${dialog.id} > .dialog-header [commandfor="${dialog.id}"][command="close"]')`;

const shown = (
  page: Page,
  dialog: Dialog,
): Promise<{ open: boolean; modal: boolean; focus: string }> =>
  page.evaluate(`(() => {
    const dialog = document.querySelector("#${dialog.id}");
    const active = document.activeElement;
    return {
      open: dialog.open,
      modal: dialog.matches(":modal"),
      focus: active === ${opener(dialog)} ? "opener" : active.id || active.localName,
    };
  })()`);

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

async function open(t: { after(fn: () => Promise<void>): void }): Promise<Page> {
  const page = await browser.open(`${app.origin}${app.base}`);
  t.after(() => page.close());
  await page.cdp("Emulation.setFocusEmulationEnabled", { enabled: true });
  await mounted(page, "app-home");
  for (const dialog of DIALOGS) {
    await until(page, `${opener(dialog)}?.hasAttribute("aria-keyshortcuts")`);
  }
  await page.evaluate(`Promise.all(["sl-settings", "sl-shortcuts"].map((tag) =>
    customElements.whenDefined(tag)))`);
  return page;
}

test("each header button shows its key, and opens Help fit to its content and Settings filling the cap", async (t) => {
  const page = await open(t);
  for (const dialog of DIALOGS) {
    assert.equal(
      await page.evaluate(`${opener(dialog)}.innerText.trim()`),
      dialog.key.toUpperCase(),
      dialog.name,
    );
    await press(page, dialog.key);
    await until(page, `document.querySelector("#${dialog.id}").open`);
    const { width, fit, cap }: { width: number; fit: number; cap: number } =
      await page.evaluate(`(() => {
      const dialog = document.querySelector("#${dialog.id}");
      const width = dialog.getBoundingClientRect().width;
      dialog.style.inlineSize = "fit-content";
      const fit = dialog.getBoundingClientRect().width;
      dialog.style.inlineSize = "100vw";
      const cap = dialog.getBoundingClientRect().width;
      dialog.style.inlineSize = "";
      return { width, fit, cap };
    })()`);
    if (dialog.width === "fit") {
      assert.equal(width, fit, `${dialog.name} fits its content`);
      assert.ok(width < cap, `${dialog.name}, ${width}px, is narrower than the cap, ${cap}px`);
    } else {
      assert.equal(width, cap, `${dialog.name}, ${width}px, fills the cap, ${cap}px`);
    }
    await press(page, "Escape");
    await until(page, `!document.querySelector("#${dialog.id}").open`);
  }
});

test("? and S open their dialogs as modals, Escape closes each, and focus returns to the opener", async (t) => {
  const page = await open(t);
  for (const dialog of DIALOGS) {
    await page.evaluate(`${opener(dialog)}.focus()`);
    await press(page, dialog.key);
    await until(page, `document.querySelector("#${dialog.id}").open`);
    const opened = await shown(page, dialog);
    assert.equal(opened.open && opened.modal, true, `${dialog.key} opens ${dialog.name}`);
    assert.notEqual(opened.focus, "opener", `focus moves into ${dialog.name}`);
    await press(page, "Escape");
    await until(page, `!document.querySelector("#${dialog.id}").open`);
    assert.deepEqual(
      await shown(page, dialog),
      { open: false, modal: false, focus: "opener" },
      `Escape closes ${dialog.name}`,
    );
  }
});

test("each opener clicked opens its dialog, whose Close shows Esc, closes it, and returns focus to the opener", async (t) => {
  const page = await open(t);
  for (const dialog of DIALOGS) {
    await click(page, `body > header [commandfor="${dialog.id}"]`);
    await until(page, `document.querySelector("#${dialog.id}").open`);
    assert.equal(await page.evaluate(`${closer(dialog)}.innerText.trim()`), "Esc");
    await page.evaluate(`${closer(dialog)}.click()`);
    await until(page, `!document.querySelector("#${dialog.id}").open`);
    assert.equal((await shown(page, dialog)).focus, "opener", dialog.name);
  }
});

test("Help lists ? and S, each with its button's name", async (t) => {
  const page = await open(t);
  await press(page, "?");
  await until(page, `document.querySelector("#help").open`);
  await until(page, `document.querySelector("sl-shortcuts").shadowRoot?.querySelector("tbody tr")`);
  const rows = await page.evaluate(`[...document.querySelector("sl-shortcuts").shadowRoot
    .querySelectorAll("tbody tr")].map((row) => [...row.cells].map((cell) =>
      cell.textContent.replace(/\\s+/g, " ").trim()))`);
  assert.deepEqual(rows, [
    ["?", "Help"],
    ["S", "Settings"],
  ]);
});

test("Settings lists every built-in profile: four colors, two spaces, two types", async (t) => {
  const page = await open(t);
  await press(page, "s");
  await until(page, `document.querySelector("#settings").open`);
  const options = await page.evaluate(`Object.fromEntries([...document.querySelector("sl-settings")
    .shadowRoot.querySelectorAll("[data-section]")].map((row) => [row.dataset.section,
    [...row.querySelectorAll("input")].map((input) => input.value)]))`);
  assert.deepEqual(options, {
    color: ["zen", "github", "catppuccin", "everforest"],
    scheme: ["system", "light", "dark"],
    space: ["regular", "compact"],
    type: ["mono", "sans"],
  });
});
