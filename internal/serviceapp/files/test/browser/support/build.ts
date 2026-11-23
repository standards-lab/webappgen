// The build, dist/, served as the service serves it with go-web-sdk's NewClient: under a base, each
// of the build's files at its path under the base, and for any other path, outside the build's
// hashed directories and with no extension, the shell NewClient renders from the build's manifest,
// so a deep link or a reload reaches the router. The shell names the Go package's render-blocking
// modules, the theme and then the page, with blocking="render", ahead of the entry, each with its
// CSS and a modulepreload for each chunk it imports, and carries the config inline, as #app-config.
// The server is web-tooling's, on a free loopback port; its handler answers under the base, and the
// server answers 404 for the rest.

import { readFileSync } from "node:fs";
import { readFile } from "node:fs/promises";
import { extname, join, posix } from "node:path";

import { serve } from "@standards-lab/web-tooling/server.js";

import type { AppConfig } from "../../../src/config.ts";

/** The build's directory, which mise run build writes. */
export const DIST = "dist";

/** The manifest key of the entry, the Go package's Entry. */
export const ENTRY = "src/main.ts";

/** The manifest keys of the modules the shell loads render-blocking, the Go package's
 * RenderBlocking, in order. */
export const RENDER_BLOCKING = ["src/theme.ts", "src/page.ts"];

/** The base the build is served under unless a case names its own. */
export const BASE = "/app/";

/** The config the shell writes as #app-config unless a case names its own. */
export const CONFIG: AppConfig = { apiBase: "/api/" };

const types: Record<string, string> = {
  ".html": "text/html",
  ".js": "text/javascript",
  ".css": "text/css",
  ".json": "application/json",
  ".svg": "image/svg+xml",
  ".png": "image/png",
  ".ico": "image/x-icon",
  ".txt": "text/plain",
  ".woff2": "font/woff2",
};

interface Chunk {
  file: string;
  css?: string[];
  imports?: string[];
}

type Manifest = Record<string, Chunk>;

/** The build's manifest, dist/.vite/manifest.json. */
export function manifestOf(dist = DIST): Manifest {
  return JSON.parse(readFileSync(join(dist, ".vite", "manifest.json"), "utf8"));
}

function resolve(manifest: Manifest, keys: string[]) {
  const scripts: string[] = [];
  const preload: string[] = [];
  const css: string[] = [];
  for (const key of keys) {
    const root = manifest[key];
    if (!root?.file) throw new Error(`the build's manifest has no entry ${JSON.stringify(key)}`);
    scripts.push(root.file);
  }
  const seen = new Set<string>(keys);
  const walk = (key: string): void => {
    const chunk = manifest[key];
    for (const file of chunk.css ?? []) if (!css.includes(file)) css.push(file);
    for (const imported of chunk.imports ?? []) {
      if (seen.has(imported)) continue;
      seen.add(imported);
      const file = manifest[imported]?.file;
      if (!file) {
        throw new Error(
          `the build's manifest has no entry ${JSON.stringify(imported)}, which ${JSON.stringify(key)} imports`,
        );
      }
      if (!scripts.includes(file) && !preload.includes(file)) preload.push(file);
      walk(imported);
    }
  };
  for (const key of keys) walk(key);
  return { scripts, preload, css };
}

const escapeHTML = (text: string): string =>
  text.replace(
    /[&<>"']/g,
    (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&#34;", "'": "&#39;" })[c] ?? c,
  );

const escapeJSON = (value: unknown): string =>
  JSON.stringify(value).replace(
    /[<>&]/g,
    (c) => `\\u${c.charCodeAt(0).toString(16).padStart(4, "0")}`,
  );

/** The shell NewClient renders for the build under base, with title and config. */
export function renderShell(
  manifest: Manifest,
  options: { base: string; title: string; config: unknown },
): string {
  const { scripts, preload, css } = resolve(manifest, [...RENDER_BLOCKING, ENTRY]);
  const entry = scripts.pop();
  const lines = [
    "<!doctype html>",
    '<html lang="en">',
    "<head>",
    '<meta charset="utf-8">',
    '<meta name="viewport" content="width=device-width, initial-scale=1">',
    `<base href="${escapeHTML(options.base)}">`,
    `<title>${escapeHTML(options.title)}</title>`,
    ...css.map((file) => `<link rel="stylesheet" href="${escapeHTML(file)}">`),
    ...preload.map((file) => `<link rel="modulepreload" href="${escapeHTML(file)}">`),
    `<script type="application/json" id="app-config">${escapeJSON(options.config)}</script>`,
    ...scripts.map(
      (file) => `<script type="module" blocking="render" src="${escapeHTML(file)}"></script>`,
    ),
    `<script type="module" src="${escapeHTML(entry ?? "")}"></script>`,
    "</head>",
    "<body></body>",
    "</html>",
    "",
  ];
  return lines.join("\n");
}

/** The build being served, until it is stopped. */
export interface Served {
  /** The server's origin, http://127.0.0.1:<port>. */
  origin: string;
  /** The base the build is served under, ending in "/". */
  base: string;
  /** Stops the server. */
  stop(): Promise<void>;
}

/** Serves the build under `base`, behind the shell rendered with `title` and `config`; `page`
 * rewrites the shell. A request's path under the base is normalized as an absolute path, so no
 * request reaches outside dist/, and a path with a dot element is never a file. */
export async function serveBuild(
  options: {
    dist?: string;
    base?: string;
    title?: string;
    config?: unknown;
    page?: (html: string) => string;
  } = {},
): Promise<Served> {
  const {
    dist = DIST,
    base = BASE,
    title = "app",
    config = CONFIG,
    page = (html: string) => html,
  } = options;
  const manifest = manifestOf(dist);
  const shell = page(renderShell(manifest, { base, title, config }));
  const hashed = new Set(
    Object.values(manifest).flatMap(({ file, css = [] }) =>
      [file, ...css].map((name) => posix.dirname(name)).filter((dir) => dir !== "."),
    ),
  );
  const server = await serve({
    root: dist,
    handler: async (request, response) => {
      const pathname = new URL(request.url ?? "/", "http://host").pathname;
      if (!pathname.startsWith(base)) return false;
      let name: string;
      try {
        name = posix.normalize(`/${decodeURIComponent(pathname.slice(base.length))}`).slice(1);
      } catch {
        return false;
      }
      const hidden = name.split("/").some((element) => element.startsWith("."));
      const body =
        hidden || name === "" ? undefined : await readFile(join(dist, name)).catch(() => undefined);
      if (body) {
        response.writeHead(200, {
          "content-type": types[extname(name)] ?? "application/octet-stream",
          "cache-control": "no-store",
        });
        response.end(body);
        return true;
      }
      const underHashed = [...hashed].some((dir) => name.startsWith(`${dir}/`));
      if (name !== "" && (hidden || underHashed || extname(name) !== "")) return false;
      response.writeHead(200, { "content-type": "text/html", "cache-control": "no-store" });
      response.end(shell);
      return true;
    },
  });
  return { origin: server.origin, base, stop: () => server.stop() };
}
