// The theme module, which the page loads with blocking="render" ahead of the app's entry, so the
// reader's stored theme choice is on the root element before the browser first paints: the
// foundation's full registration module, every built-in profile page-full.css ships (the color
// palettes zen, github, catppuccin, and everforest; the spaces regular and compact; the types mono
// and sans), then its preferences module, which applies the choice stored under
// web-foundation.preferences and follows another tab's save. Registration comes first:
// preferences.js drops a stored name no module has registered. Its bytes delay first paint, so it
// imports nothing more.

import "@standards-lab/web-foundation/register-full.js";
import "@standards-lab/web-foundation/preferences.js";
