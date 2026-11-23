// The browser tier's reading of the app: which view a page shows, and the navigations and key
// presses a person makes, through the DevTools Protocol. Each wait is web-tooling's page wait,
// until, which polls an expression inside the page.

import { until } from "@standards-lab/web-tooling/browser.js";
import type { Browser, Page } from "@standards-lab/web-tooling/browser.js";

/** Where a page is: its URL path, and the tag of <main>'s first element, null when it has none. */
export interface Route {
  path: string;
  tag: string | null;
}

/** The page's URL path and the tag of the view its router mounted in <main>. */
export const route = (page: Page): Promise<Route> =>
  page.evaluate(
    "({ path: location.pathname, tag: document.querySelector('main')?.firstElementChild?.localName ?? null })",
  );

/** Resolves once the page's router has mounted `tag`. */
export const mounted = (page: Page, tag: string): Promise<unknown> =>
  until(
    page,
    `document.querySelector('main')?.firstElementChild?.localName === ${JSON.stringify(tag)}`,
  );

/** Clicks the element `selector` finds, as a mouse does, at its center. */
export async function click(page: Page, selector: string): Promise<void> {
  const { x, y }: { x: number; y: number } = await page.evaluate(`(() => {
    const box = document.querySelector(${JSON.stringify(selector)}).getBoundingClientRect();
    return { x: box.x + box.width / 2, y: box.y + box.height / 2 };
  })()`);
  for (const type of ["mousePressed", "mouseReleased"]) {
    await page.cdp("Input.dispatchMouseEvent", { type, x, y, button: "left", clickCount: 1 });
  }
}

const KEYS = {
  s: { key: "s", code: "KeyS", windowsVirtualKeyCode: 83, text: "s" },
  "?": { key: "?", code: "Slash", windowsVirtualKeyCode: 191, modifiers: 8, text: "?" },
  Escape: { key: "Escape", code: "Escape", windowsVirtualKeyCode: 27 },
};

/** Presses and releases `name`, as a keyboard does, on the page's focused element. */
export async function press(page: Page, name: keyof typeof KEYS): Promise<void> {
  const { text, ...key } = { text: undefined, ...KEYS[name] };
  await page.cdp("Input.dispatchKeyEvent", {
    type: text === undefined ? "rawKeyDown" : "keyDown",
    ...key,
    ...(text === undefined ? {} : { text, unmodifiedText: text }),
  });
  await page.cdp("Input.dispatchKeyEvent", { type: "keyUp", ...key });
}

/** Reloads the page, as the browser's button does, resolving once the new document has loaded. */
export async function reload(page: Page): Promise<void> {
  const loaded = new Promise((resolve) => page.on("Page.loadEventFired", resolve));
  await page.cdp("Page.reload");
  await loaded;
}

/** Opens a blank page, runs `prepare` on it (enabling a protocol domain, say), then loads `url`. */
export async function openWith(
  browser: Browser,
  url: string,
  prepare: (page: Page) => Promise<void>,
): Promise<Page> {
  const page = await browser.open("about:blank");
  await prepare(page);
  await page.goto(url);
  return page;
}
