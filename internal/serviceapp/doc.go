// Package serviceapp is the service-app kind of project: a Go web service
// copied from go-web-sdk-template, with a web client written into it.
// [Command] is its leaf under new, with the kind's flags, the checks that
// refuse its input before anything is written, the copy of the template,
// made by gonew through go run, and the writing of the client, the starter
// with [Files] laid over it, rendered with [Values]. [TemplatePin] and
// [ToolsPin] are the versions it copies and copies with; [CheckModule] is
// the check of the written service's module path, [ParseClientDir] of the
// directory the client is written into, and [PackageName] names the
// client's Go package. [Wire] wires the client into the service's
// composition root: the client file it writes, and its edits at the
// template's anchors, applied all together or left all together.
package serviceapp
