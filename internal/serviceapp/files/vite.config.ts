// The client's build (mise run build): the entry, theme, and page modules into dist/, with the
// manifest go-web-sdk's NewClient renders the shell from.
// See README.md

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
