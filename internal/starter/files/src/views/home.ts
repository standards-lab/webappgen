// The home view, at the base: a heading, the API the config names, and the components' status
// badge. It renders into light DOM, so the page's foundation sheet styles it. Loaded on its first
// visit.

import "@standards-lab/web-components/status-badge.js";

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
    const badge = document.createElement("sl-status-badge");
    badge.setAttribute("label", "ready");
    this.replaceChildren(heading, api, badge);
  }
}

customElements.define(homeTag, AppHome);
