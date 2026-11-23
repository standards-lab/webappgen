// The client's build (mise run build): the entry, theme, and page modules, each a Vite input, into
// dist/, with the manifest go-web-sdk's NewClient reads to name their scripts and CSS in the shell
// it renders. The service writes the HTML, so there is no index.html and no public directory. The
// base is relative, so the build is served under any mount: the shell's <base href> names it, and
// every asset and chunk resolves against it. The theme and page modules are the ones the shell
// loads render-blocking ahead of the entry, the Go package's RenderBlocking.
// See README.md
// Adapted from spike-html-apps (2883d78): vite.config.ts, its three module inputs, for one client
// app.

import { defineConfig } from "vite";
import { standardCssModules } from "vite-plugin-standard-css-modules";

export default defineConfig({
  base: "./",
  publicDir: false,
  plugins: [
    standardCssModules({
      include: ["**/node_modules/@standards-lab/**/*.js"],
      outputMode: "CSSStyleSheet",
    }),
  ],
  build: {
    outDir: "dist",
    manifest: true,
    target: "es2024",
    rolldownOptions: {
      input: { main: "src/main.ts", theme: "src/theme.ts", page: "src/page.ts" },
      output: { strictExecutionOrder: true },
    },
  },
});
