// The config reader: #app-config's JSON when the page has the element, otherwise config.json's,
// fetched relative to the page's base; and each failure, naming the element or the file. The page
// and the fetch are the test's own, on node.

import assert from "node:assert/strict";
import { test } from "node:test";

import { appConfig, readConfig, useConfig } from "../../src/config.ts";
import type { ConfigPage, Fetch } from "../../src/config.ts";

const BASE = "http://127.0.0.1:8080/app/";
const FILE = `config.json (${BASE}config.json)`;

function page(element?: { type: string | null; text: string }): ConfigPage {
  return {
    baseURI: BASE,
    getElementById: (id) =>
      element && id === "app-config"
        ? {
            getAttribute: (name) => (name === "type" ? element.type : null),
            textContent: element.text,
          }
        : null,
  };
}

function answering(status: number, body: string): Fetch & { urls: string[] } {
  const urls: string[] = [];
  const load = async (url: string) => {
    urls.push(url);
    return { ok: status >= 200 && status < 300, status, text: async () => body };
  };
  return Object.assign(load, { urls });
}

const escape = (text: string) => text.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

const unreachable: Fetch = async () => {
  throw new TypeError("fetch failed");
};

test("reads #app-config's JSON when the page has the element, and fetches nothing", async () => {
  const load = answering(200, '{"apiBase": "/from-file/"}');
  const config = await readConfig(
    page({ type: "application/json", text: '{"apiBase": "https://api.example.com/v1/"}' }),
    load,
  );
  assert.deepEqual(config, { apiBase: "https://api.example.com/v1/" });
  assert.deepEqual(load.urls, []);
});

test("fetches config.json relative to the page's base when the page has no #app-config", async () => {
  const load = answering(200, '{"apiBase": "/api/"}');
  assert.deepEqual(await readConfig(page(), load), { apiBase: "/api/" });
  assert.deepEqual(load.urls, [`${BASE}config.json`]);
});

test("each failure names the element or the file", async () => {
  for (const [name, read, message] of [
    [
      "an element of another type",
      () => readConfig(page({ type: "text/plain", text: '{"apiBase": "/api/"}' }), unreachable),
      `#app-config isn't a <script type="application/json"> element`,
    ],
    [
      "an element that isn't JSON",
      () => readConfig(page({ type: "application/json", text: "apiBase=/api/" }), unreachable),
      /^#app-config isn't JSON: /,
    ],
    [
      "an element of the wrong shape",
      () => readConfig(page({ type: "application/json", text: '{"apiBase": 1}' }), unreachable),
      "#app-config isn't an app config: its apiBase isn't a string",
    ],
    [
      "a missing file",
      () => readConfig(page(), answering(404, "Not Found")),
      `${FILE} answered 404`,
    ],
    [
      "a file that can't be fetched",
      () => readConfig(page(), unreachable),
      `${FILE} couldn't be fetched: fetch failed`,
    ],
    [
      "a file that isn't JSON",
      () => readConfig(page(), answering(200, "<!doctype html>")),
      new RegExp(`^${escape(FILE)} isn't JSON: `),
    ],
    [
      "a file of the wrong shape",
      () => readConfig(page(), answering(200, '{"api": "/api/"}')),
      `${FILE} isn't an app config: its apiBase isn't a string`,
    ],
    [
      "a file holding null",
      () => readConfig(page(), answering(200, "null")),
      `${FILE} isn't an app config: its apiBase isn't a string`,
    ],
  ] as const) {
    await assert.rejects(read(), { message }, name);
  }
});

test("appConfig returns the config useConfig set", () => {
  useConfig({ apiBase: "/api/" });
  assert.deepEqual(appConfig(), { apiBase: "/api/" });
});
