// Package serviceapp is the service-app kind of project: a Go web service
// copied from go-web-sdk-template, which a web client is later installed
// into. [Command] is its leaf under new, with the kind's flags, the checks
// that refuse its input before anything is written, and the copy of the
// template, made by gonew through go run. [TemplatePin] and [ToolsPin] are
// the versions it copies and copies with; [CheckModule] is the check of the
// written service's module path, and [ParseClientDir] of the directory the
// client is installed into.
package serviceapp
