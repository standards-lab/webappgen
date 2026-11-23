// The home view, at the base: a heading and the API the config names. It renders into light DOM,
// so the page's foundation sheet styles it. Loaded on its first visit.

import { appConfig } from "../config.ts";

/** The home view's tag, which the router mounts. */
export const homeTag = "app-home";

class AppHome extends HTMLElement {
  connectedCallback() {
    const heading = document.createElement("h2");
    heading.textContent = "Home";
    const api = document.createElement("p");
    const code = document.createElement("code");
    code.textContent = appConfig().apiBase;
    api.append("The app calls its API at ", code, ".");
    this.replaceChildren(heading, api);
  }
}

customElements.define(homeTag, AppHome);
