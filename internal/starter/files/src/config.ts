// The app's config, which it reads once, at start: from the page's #app-config, an inline JSON
// script element, when the page has one, and otherwise from config.json, fetched relative to the
// document's base. A config that is missing, isn't JSON, or isn't an AppConfig fails, naming the
// element or the file.

/** The id of the `<script type="application/json">` element a page may write the config into. */
export const configId = "app-config";

/** The file the config is fetched from, relative to the document's base, when the page has no
 * #app-config. */
export const configFile = "config.json";

/** The app's config. */
export interface AppConfig {
  /** The API the app calls, a URL or a path; the app resolves it against the document's base. */
  apiBase: string;
}

/** What readConfig reads of the page: an element by its id, and the base URL. A Document is
 * one; the unit tier, on node, passes its own. */
export interface ConfigPage {
  readonly baseURI: string;
  getElementById(
    id: string,
  ): { getAttribute(name: string): string | null; readonly textContent: string | null } | null;
}

/** Fetches a URL, as fetch does; the unit tier passes its own. */
export type Fetch = (
  url: string,
) => Promise<{ readonly ok: boolean; readonly status: number; text(): Promise<string> }>;

/** The page's config: #app-config's when the page has the element, otherwise config.json's,
 * fetched relative to the page's base with `load`. Rejects, naming the element or the file's URL,
 * when the element isn't a JSON script element, the file can't be fetched or answers other than
 * 2xx, or either doesn't hold JSON or an AppConfig. */
export async function readConfig(page: ConfigPage, load: Fetch = fetch): Promise<AppConfig> {
  const element = page.getElementById(configId);
  if (element) {
    const name = `#${configId}`;
    if (element.getAttribute("type") !== "application/json") {
      throw new Error(`${name} isn't a <script type="application/json"> element`);
    }
    return parse(name, element.textContent ?? "");
  }
  const url = new URL(configFile, page.baseURI).href;
  const name = `${configFile} (${url})`;
  let response;
  try {
    response = await load(url);
  } catch (error) {
    throw new Error(`${name} couldn't be fetched: ${(error as Error).message}`, { cause: error });
  }
  if (!response.ok) throw new Error(`${name} answered ${response.status}`);
  return parse(name, await response.text());
}

function parse(name: string, text: string): AppConfig {
  let config: unknown;
  try {
    config = JSON.parse(text);
  } catch (error) {
    throw new Error(`${name} isn't JSON: ${(error as Error).message}`, { cause: error });
  }
  if (!isConfig(config)) throw new Error(`${name} isn't an app config: its apiBase isn't a string`);
  return config;
}

function isConfig(value: unknown): value is AppConfig {
  return typeof (value as Partial<AppConfig> | null)?.apiBase === "string";
}

let current: AppConfig | undefined;

/** Sets the config every view reads; main calls it before the router starts. */
export function useConfig(config: AppConfig): void {
  current = config;
}

/** The config main set; throws when a view mounts before useConfig was called. */
export function appConfig(): AppConfig {
  if (!current) throw new Error("useConfig wasn't called before a view mounted");
  return current;
}
