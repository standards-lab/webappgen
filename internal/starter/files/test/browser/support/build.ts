// The build, dist/, served as a static host serves the app: under the base its index.html's
// <base href> names, each of the build's files at its path under the base, and index.html for any
// other path with no extension, so a deep link or a reload reaches the router. The server is
// web-tooling's, on a free loopback port; its handler answers under the base, and the server
// answers 404 for the rest.

import { readFileSync } from "node:fs";
import { readFile, stat } from "node:fs/promises";
import { extname, join, normalize } from "node:path";

import { serve } from "@standards-lab/web-tooling/server.js";

/** The build's directory, which mise run build writes. */
export const DIST = "dist";

/** The content type of each extension the build holds; any other is sent as bytes. */
const types: Record<string, string> = {
  ".html": "text/html",
  ".js": "text/javascript",
  ".css": "text/css",
  ".json": "application/json",
  ".svg": "image/svg+xml",
  ".png": "image/png",
  ".ico": "image/x-icon",
  ".txt": "text/plain",
};

/** The build being served, until it is stopped. */
export interface Served {
  /** The server's origin, http://127.0.0.1:<port>. */
  origin: string;
  /** The base the build's index.html names, ending in "/". */
  base: string;
  /** Stops the server. */
  stop(): Promise<void>;
}

/** The <base href> the build's index.html names; throws when it names none. */
export function baseOf(dist = DIST): string {
  const html = readFileSync(join(dist, "index.html"), "utf8");
  const base = /<base href="([^"]*)"/.exec(html)?.[1];
  if (!base) throw new Error(`${dist}/index.html has no <base href>`);
  return base;
}

/** Serves the build; `page` rewrites every HTML file it sends, index.html included. */
export async function serveBuild(
  options: { dist?: string; page?: (html: string) => string } = {},
): Promise<Served> {
  const { dist = DIST, page = (html: string) => html } = options;
  const base = baseOf(dist);
  const server = await serve({
    root: dist,
    handler: async (request, response) => {
      const pathname = new URL(request.url ?? "/", "http://host").pathname;
      if (!pathname.startsWith(base)) return false;
      // normalize() of an absolute path never climbs above its root, so no request leaves dist/.
      let path: string;
      try {
        path = normalize(`/${decodeURIComponent(pathname.slice(base.length))}`);
      } catch {
        return false;
      }
      let file = join(dist, path);
      const found = await stat(file).catch(() => undefined);
      if (!found?.isFile()) {
        if (extname(path) !== "") return false;
        file = join(dist, "index.html");
      }
      const type = types[extname(file)] ?? "application/octet-stream";
      let body: string | Buffer = await readFile(file);
      if (type === "text/html") body = page(body.toString("utf8"));
      response.writeHead(200, { "content-type": type, "cache-control": "no-store" });
      response.end(body);
      return true;
    },
  });
  return { origin: server.origin, base, stop: () => server.stop() };
}
