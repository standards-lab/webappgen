// The view for a path no route matches: it says so and links back to the home view, at the base.
// It renders into light DOM, as the home view does. Loaded on its first visit. Its link, `.`, is
// the document's base, the home view.

/** The not-found view's tag, which the router mounts. */
export const notFoundTag = "app-not-found";

class AppNotFound extends HTMLElement {
  connectedCallback() {
    const heading = document.createElement("h2");
    heading.textContent = "Not found";
    const text = document.createElement("p");
    text.textContent = "Nothing in the app lives at this address.";
    const link = document.createElement("a");
    link.href = ".";
    link.textContent = "Go to the home page";
    const back = document.createElement("p");
    back.append(link);
    this.replaceChildren(heading, text, back);
  }
}

customElements.define(notFoundTag, AppNotFound);
