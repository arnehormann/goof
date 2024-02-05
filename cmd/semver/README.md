# semver

`semver` reports details for a git repository as variables.
It retrieves non-changing details for the current build from git
and prints them.

These are

* HEAD commit revision
* HEAD commit timestamp (UTC)
* current branch
* semantic version based on tag
* are there uncommitted tracked files

Note: the branch is not necessarily deterministic, better avoid using it.
Everyone can name it locally however they want. Might be useful on CI though.

Tags are only used for the semantic version.
They could also be moved - don't do that. Or if you are prone to do it, don't use
the `Semver` field in the template.

We do not currently provide author or committer information or the commit message.
That would grow the possible error cases and there was no clear use case yet.

## Semantic versioning

It mainly helps with semantic versioning and it will show tags following a
subset of the specification at https://semver.org/.
The regular expression used is the one from https://semver.org/#is-there-a-suggested-regular-expression-regex-to-check-a-semver-string

## Installation

You can install it with `go get` or using Bazel: `bazel build @iq_buildtools//cmd/semver`.

## Templating

The default template is usable as output for a Bazel workspace_status command.
Call `semver -help` to get details concerning its operation.

## Path

If it is set (run from Bazel), it will change directories into the path referenced in
`BUILD_WORKSPACE_DIRECTORY` before git is run. This is done to make sure the right repository
is used. The target directory can be changed using `-dir`, see the  help output below.

## Default result

The output looks like this:
```
STABLE_COMMIT_ID 5833e2847a3ced66f119a79c84faa4f6e0c943fd
STABLE_COMMIT_TS 1586368369
STABLE_COMMIT_UTC 2020-04-08T17:52:49
STABLE_COMMIT_UTC_TAG 20200408175249
STABLE_COMMIT_BUILD 20200408175249.5833e284.1586455851
STABLE_COMMIT_SEMVER 0.0.0-20200408175249.5833e284.1586455851
STABLE_COMMIT_BRANCH master
STABLE_COMMIT_STATUS modified
```

## Default help text

The help text currently looks like this:
```
Use semver to retrieve versioning information for the repository containing /mnt/space/src/bitbucket.org/vauwede/rulestack/cmd/semver
Git is used to retrieve the data. It must be available in your PATH.
Times used in the default template are UTC. Time errors are encoded as unix epoch.
Uncommitted files result in a version number v0.0.0

  -debug
        print detailed information for arguments and the data from git
  -dir string
        set execution directory (default "<DIRECTORY>")
  -errlog
        log failing git call details to stderr
  -help
        show this help text
  -out string
        output file, leave it empty for stdout
  -ref string
        git reference to a commit to operate on. For testing, should not be changed (default "HEAD")
  -template string
        path to a template file (text/template in Go). Empty for the default below
Check https://golang.org/pkg/text/template for a template reference.
Two functions are supported: Now for the current time and Env to retrieve an environment variable.
The default template follows these conventions:
* time is always UTC
* time errors are encoded as Unix epoch (1970-01-01T00:00:00)
* unknown revisions consist of a series of "0"
* tracked but uncommitted files are not fit for a release version
* tracked but not committed files lead to a 0.0.0-... semver as they are not fit for release

The default template looks like this:

{{- define "tagregexp"}}
{{- /* regexp from https://semver.org/#is-there-a-suggested-regular-expression-regex-to-check-a-semver-string plus optional leading "v"*/ -}}
^v?(?P<major>0|[1-9]\d*)\.(?P<minor>0|[1-9]\d*)\.(?P<patch>0|[1-9]\d*)(?:-(?P<prerelease>(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+(?P<buildmetadata>[0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$
{{- end}}
{{- $now := Now}}
{{- $rev := "0000000000000000000000000000000000000000"}}{{- if ge (len .Revision) 40}}{{$rev = .Revision}}{{end}}
{{- $shortrev := slice $rev 0 8}}
{{- $timestamp := .Time.UTC.Unix}}
{{- $utc := .Time.UTC.Format "2006-01-02T15:04:05"}}
{{- $utctag := .Time.UTC.Format "20060102150405"}}
{{- $status := "modified"}}{{- if .Clean}}{{$status = "clean"}}{{end}}
{{- $devsuffix := ""}}{{- if eq false .Clean}}{{$devsuffix = printf ".%v" $now.Unix}}{{end}}
{{- $build := printf "%s.%s%s" $utctag (slice .Revision 0 8) $devsuffix}}
{{- $semver := .Semver}}{{- if or (not .Clean) (eq .Semver "")}}{{$semver = printf "0.0.0-%s" $build}}{{end}}
{{- if eq "v" (slice $semver 0 1)}}{{$semver = slice $semver 1}}{{end}}
{{- $branch := .Branch -}}
STABLE_COMMIT_ID {{$rev}}
STABLE_COMMIT_TS {{$timestamp}}
STABLE_COMMIT_UTC {{$utc}}
STABLE_COMMIT_UTC_TAG {{$utctag}}
STABLE_COMMIT_BUILD {{$build}}
STABLE_COMMIT_SEMVER {{$semver}}
STABLE_COMMIT_BRANCH {{$branch}}
STABLE_COMMIT_STATUS {{$status}}
```
