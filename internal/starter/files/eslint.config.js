// The lint the check runs: the recommended rules over the code, on the browser's globals in src/
// and node's in the config, the scripts, and the tests.
// Adapted from spike-html-apps (2883d78): eslint.config.js, for one client app, without the app
// tier's table, which a project of one app has no other app to hold apart from.

import js from "@eslint/js";
import { defineConfig, globalIgnores } from "eslint/config";
import globals from "globals";
import tseslint from "typescript-eslint";

const code = ["**/*.js", "**/*.ts"];

export default defineConfig([
  // The build (mise run build), which .gitignore lists and ESLint doesn't read.
  globalIgnores(["dist/"]),
  { files: code, extends: [js.configs.recommended, tseslint.configs.recommended] },
  { files: code, languageOptions: { globals: globals.browser } },
  {
    files: ["test/**", "scripts/**", "*.config.js", "*.config.ts"],
    languageOptions: { globals: globals.node },
  },
]);
