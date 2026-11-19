// The app's entry, the module the page loads after the render-blocking theme module (theme.ts):
// the foundation's page stylesheet, the one stylesheet a page links, which Vite bundles as CSS; then
// the config, read once; then the router, over the page's <main>, which mounts the home view at the
// base and the not-found view at any other path. The router's base is the document's <base href>;
// links are relative to it. Each view loads on its first visit.
// Adapted from spike-html-apps (2883d78): apps/pipeline/src/main.ts, its config and router alone.

import "@standards-lab/web-foundation/page.css";

import { createRouter } from "@standards-lab/web-app-sdk/router.js";

import { readConfig, useConfig } from "./config.ts";

const outlet = document.querySelector("main");
if (!outlet) throw new Error("the page has no <main> for the router's views");

try {
  useConfig(await readConfig(document));
} catch (error) {
  // The reader's error names the element or the file; the page shows it where a view would be.
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
