// Package standalone is the standalone kind of project: a web client served
// on its own, under an absolute base path, reading its API's address from
// the config.json beside its page. [Command] is its leaf under new, with
// the kind's flags, the checks that refuse its input before anything is
// written, the writing of the starter with [Files] laid over it, and the
// next steps it prints; [ParseBase] is the check of the base path the app
// is served under.
package standalone
