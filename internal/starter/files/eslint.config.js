// The lint the check runs: the recommended rules over the code, a description on every ESLint and
// TypeScript directive, the browser's globals in src/, and node's in the config, the scripts, and
// the tests.

import comments from "@eslint-community/eslint-plugin-eslint-comments";
import js from "@eslint/js";
import { defineConfig, globalIgnores } from "eslint/config";
import globals from "globals";
import tseslint from "typescript-eslint";

const code = ["**/*.js", "**/*.ts"];

export default defineConfig([
  globalIgnores(["dist/"]),
  { files: code, extends: [js.configs.recommended, tseslint.configs.recommended] },
  {
    files: code,
    plugins: { "@eslint-community/eslint-comments": comments },
    rules: {
      "@eslint-community/eslint-comments/require-description": "error",
      "@typescript-eslint/ban-ts-comment": [
        "error",
        {
          "ts-check": false,
          "ts-expect-error": "allow-with-description",
          "ts-ignore": "allow-with-description",
          "ts-nocheck": "allow-with-description",
        },
      ],
    },
  },
  { files: code, languageOptions: { globals: globals.browser } },
  {
    files: ["test/**", "scripts/**", "*.config.js", "*.config.ts"],
    languageOptions: { globals: globals.node },
  },
]);
