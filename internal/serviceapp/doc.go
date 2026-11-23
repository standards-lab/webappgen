// Package serviceapp is the service-app kind of project: a Go web service
// copied from go-web-sdk-template, with a web client written into it.
// [Command] is its leaf under new, with the kind's flags, the checks that
// refuse its input before anything is written, the copy of the template,
// made by gonew through go run, and the writing of the client, the starter
// with [Files] laid over it, rendered with [Values]. [TemplatePin] and
// [ToolsPin] are the versions it copies and copies with; [CheckModule] is
// the check of the written service's module path, [ParseClientDir] of the
// directory the client is written into, and [PackageName] names the
// client's Go package. [InjectCommand] is its leaf under inject, which
// writes the client into an existing service, with the same flags, the
// same writing of the client, and the same wiring. [Wire] wires the
// client into the service's composition root: the client file it writes,
// and its edits at the template's anchors, applied all together or left
// all together; [CheckWire] refuses a service Wire would refuse, before
// anything is written. [EditConfig] edits the service's mise.toml and CI
// workflow, each only when it holds the pinned template's text where the
// edit goes, so its tasks and CI build the client, and writes its
// .air.toml for the dev loop, never over one that exists. Both leaves'
// --install runs their install steps with [project.Install].
package serviceapp
