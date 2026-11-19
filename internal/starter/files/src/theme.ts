// The theme module, which the page loads with blocking="render" ahead of the app's entry, so the
// reader's stored theme choice is on the root element before the browser first paints: the
// foundation's lean registration module, the default profiles page.css ships, then its
// preferences module, which applies the choice stored under web-foundation.preferences, follows
// another tab's save, and defines <sl-theme-menu>. Its bytes delay first paint, so it imports
// nothing more.
// Adapted from spike-html-apps (2883d78): apps/pipeline/src/register.ts and
// apps/pipeline/src/preferences.ts, one module on the lean entry.

import "@standards-lab/web-foundation/register.js";
import "@standards-lab/web-foundation/preferences.js";
