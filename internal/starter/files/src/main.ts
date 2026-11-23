// The app's entry, the module the page loads after the render-blocking theme module (theme.ts):
// the foundation's full page stylesheet, page-full.css, whose profiles theme.ts registers, then its
// glyph-button, dialog, and layout sheets, which Vite bundles as CSS; the foundation's keys.js,
// which binds the header's Help and Settings buttons to ? and S by their data-keys; the
// components' sl-settings and sl-shortcuts, which the page's dialogs hold; then the config, read
// once; then the router, over the page's <main>, which mounts the home view at the base and the
// not-found view at any other path. The router's base is the document's <base href>; links are
// relative to it. Each view loads on its first visit. A config the reader refuses shows, in place
// of a view, as an alert naming the element or the file.

import "@standards-lab/web-foundation/page-full.css";
import "@standards-lab/web-foundation/glyph-button.css";
import "@standards-lab/web-foundation/dialog.css";
import "@standards-lab/web-foundation/layout.css";
import "@standards-lab/web-foundation/keys.js";
import "@standards-lab/web-components/settings.js";
import "@standards-lab/web-components/shortcuts.js";

import { createRouter } from "@standards-lab/web-app-sdk/router.js";

import { readConfig, useConfig } from "./config.ts";

const outlet = document.querySelector("main");
if (!outlet) throw new Error("the page has no <main> for the router's views");

try {
  useConfig(await readConfig(document));
} catch (error) {
  const alert = document.createElement("p");
  alert.setAttribute("role", "alert");
  alert.textContent = `The app couldn't start: ${(error as Error).message}`;
  outlet.replaceChildren(alert);
  throw error;
}

const router = createRouter(
  { "/": async () => (await import("./views/home.ts")).homeTag },
  { outlet, notFound: async () => (await import("./views/not-found.ts")).notFoundTag },
);
await router.start();
