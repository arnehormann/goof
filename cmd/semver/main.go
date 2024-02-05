package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"
)

const (
	tagregexp = "tagregexp"

	reNumber     = `0|[1-9]\d*`
	reIdentifier = `0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*`
	reMeta       = `[0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)`

	// https://semver.org/spec/v2.0.0.html
	semverregexp = `^` +
		`v?` + // optional "v" prefix
		`(?P<major>` + reNumber + `)` + // named number "major"
		`\.` +
		`(?P<minor>` + reNumber + `)` + // named number "minor"
		`\.` +
		`(?P<patch>` + reNumber + `)` + // named number "patch"
		`(?:-` + // optionally followed by "-" separated prerelease
		`(?P<prerelease>(?:` + reIdentifier + `)(?:\.(?:` + reIdentifier + `))*)` +
		`)?` +
		`(?:\+` + // optionally followed by "+" separated buildmetadata
		`(?P<buildmetadata>` + reMeta + `*)` +
		`)?` +
		`$`
)

// template prefix to set set various variables when rendering CommitInfo.
// concerning the semantic version format: the regexp is from
//   https://semver.org/#is-there-a-suggested-regular-expression-regex-to-check-a-semver-string
// with an added optional leading "v"
//
// reference for supported environment variables in the default template:
// https://JENKINS_HOST/env-vars.html/
var varPrefix = `
{{- define "` + tagregexp + `"}}` + semverregexp + `{{end}}
{{- $now := Now}}
{{- $buildid := Env "BUILD_ID"}}
{{- $changeid := Env "CHANGE_ID"}}
{{- $rev := "0000000000000000000000000000000000000000"}}{{- if ge (len .Revision) 40}}{{$rev = .Revision}}{{end}}
{{- $shortrev := slice $rev 0 8}}
{{- $timestamp := .Time.UTC.Unix}}
{{- $utc := .Time.UTC.Format "2006-01-02T15:04:05"}}
{{- $utctag := .Time.UTC.Format "20060102150405"}}
{{- $status := "modified"}}{{- if .Clean}}{{$status = "clean"}}{{end}}
{{- $devsuffix := ""}}{{- if eq false .Clean}}{{$devsuffix = printf ".%v" $now.Unix}}{{end}}
{{- $build := printf "%s.%s%s" $utctag (slice .Revision 0 8) $devsuffix}}
{{- $buildtag := $build}}
{{- $semver := .Semver}}{{- if or (not .Clean) (eq .Semver "")}}{{$semver = printf "0.0.0-%s" $buildtag}}{{end}}
{{- if (ne $changeid "")}}{{$semver = printf "change%06s" $changeid}}{{end}}
{{- if eq "v" (slice $semver 0 1)}}{{$semver = slice $semver 1}}{{end}}
{{- $branch := .Branch -}}
`

var formats = map[string]string{
	"bazel": varPrefix + `
STABLE_COMMIT_ID {{$rev}}
STABLE_COMMIT_TS {{$timestamp}}
STABLE_COMMIT_UTC {{$utc}}
STABLE_COMMIT_UTC_TAG {{$utctag}}
STABLE_COMMIT_BUILD {{$build}}
STABLE_COMMIT_SEMVER {{$semver}}
STABLE_COMMIT_BRANCH {{$branch}}
STABLE_COMMIT_STATUS {{$status}}
`,
	"env": varPrefix + `
COMMIT_ID={{$rev}}
COMMIT_TS={{$timestamp}}
COMMIT_UTC={{$utc}}
COMMIT_UTC_TAG={{$utctag}}
COMMIT_BUILD={{$build}}
COMMIT_SEMVER={{$semver}}
COMMIT_BRANCH={{$branch}}
COMMIT_STATUS={{$status}}
`,
	"version": varPrefix + `{{$semver}}
`,
}

const (
	formatUTC    = "2006-01-02T15:04:05"
	formatUTCTag = "20060102150405"
)

const (
	_ = iota // start at 1 below to get non-0 exit codes
	// ExitOnCommand is the exit code for an error running git
	ExitOnCommand
	// ExitOnUsage is the exit code for wrong arguments
	ExitOnUsage
	// ExitOnTemplate is the exit code if the template could not be compiled
	ExitOnTemplate
	// ExitOnRegexp is the exit code if the version regexp could not be compiled
	ExitOnRegexp
	// ExitOnChdir is the exit code if the process could not change directories
	ExitOnChdir
	// ExitOnCreateFile is the exit code if the output file could not be created
	ExitOnCreateFile
)

type discarder struct{}

func (d discarder) Read([]byte) (int, error) { return 0, nil }

func (d discarder) Write([]byte) (int, error) { return 0, nil }

func (d discarder) Printf(string, ...interface{}) {}

// CommitInfo contains information retrieved from git
type CommitInfo struct {
	Revision string
	Semver   string
	Branch   string
	Time     time.Time
	Clean    bool
}

// NewCommitInfo runs various "git" commands to retrieve a CommitInfo
// for the current working directory.
func NewCommitInfo(ref string, reSemver *regexp.Regexp) (*CommitInfo, error) {
	epoch := time.Unix(0, 0).UTC()
	c := &CommitInfo{}
	var rev string
	ts_rev, err := git("rev-list", "-1", "--timestamp", ref)
	if err != nil {
		if ref == "HEAD" {
			bad := &CommitInfo{
				Time: epoch,
				Semver: fmt.Sprintf(
					"v0.0.0-%s-00000000-%s",
					epoch,
					time.Now().UTC().Format(formatUTCTag),
				),
			}
			return bad, fmt.Errorf("detached HEAD: %v", err)
		}
		return nil, fmt.Errorf("could not process rev-list for %q: %v", ref, err)
	}
	idx := strings.IndexAny(ts_rev, " \t")
	if idx < 0 {
		return nil, fmt.Errorf("illegal result format for git rev-list, needs to contain space or tab: %q", ts_rev)
	}
	ts, rev := ts_rev[0:idx], strings.TrimSpace(ts_rev[idx+1:])
	d, err := strconv.ParseInt(ts, 10, 64)
	if err == nil {
		c.Time = time.Unix(d, 0).UTC()
	}
	c.Revision = rev
	tags, err := git("tag", "--points-at", ref)
	if err == nil && tags != "" {
		var semver string
		for _, v := range strings.Split(tags, "\n") {
			v = strings.TrimSpace(v)
			if !reSemver.MatchString(v) {
				continue
			}
			if semver == "" || semver < v {
				semver = v
			}
		}
		c.Semver = semver
	}
	changed, err := git("diff-index", "--quiet", ref)
	if err == nil && changed == "" {
		c.Clean = true
	}
	branch, err := git("symbolic-ref", "--short", ref)
	if err == nil {
		end := strings.IndexAny(branch, " \t\r\n")
		if end >= 0 {
			branch = branch[:end]
		}
		c.Branch = strings.TrimSpace(branch)
	}
	// Possible CommitInfo extensions (but better not to keep error handling manageable):
	// $(git show --format=%XYZ ref) could be used - with these "XYZ" values:
	// with "X" of either "a" for author or "c" for committer:
	// "Xn" - name
	// "Xe" - email address
	// "Xt" - unix timestamp
	// or also
	// "s" subject
	// "b" body
	// "B" raw body (including subject)
	return c, nil
}

func git(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var wout bytes.Buffer
	var werr bytes.Buffer
	cmd.Stdin = bytes.NewReader(nil)
	cmd.Stdout = &wout
	cmd.Stderr = &werr
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("git error for %v: %v\n", args, err)
	}
	if werr.Len() != 0 {
		return "", fmt.Errorf("git error for %v: %v\n", args, werr.String())
	}
	return wout.String(), nil
}

func main() {
	formatKeys := make([]string, 0, len(formats))
	for k, _ := range formats {
		formatKeys = append(formatKeys, k)
	}
	sort.Strings(formatKeys)

	var (
		dir        string
		format     string = "bazel"
		tmpl       string
		ref        string = "HEAD"
		out        string
		setversion string
		unixline   bool = true
		debug      bool
		errlog     bool
		help       bool
	)

	defaultTemplate := formats[format]

	dir = os.Getenv("BUILD_WORKSPACE_DIRECTORY")
	if dir == "" {
		dir, _ = os.Getwd()
	}

	flag.StringVar(&dir, "dir", dir, "set execution directory")
	flag.StringVar(&format, "format", format, "output format, overridable by template. Valid values are: "+strings.Join(formatKeys, ", "))
	flag.StringVar(&tmpl, "template", tmpl, "path to a template file (text/template in Go). Empty for predefined formats")
	flag.StringVar(&ref, "ref", ref, "git reference to a commit to operate on. For testing, should not be changed")
	flag.StringVar(&setversion, "use", setversion, "replace 'git tag' based semver with this one and consider the repo clean")
	flag.StringVar(&out, "out", out, "output file, leave it empty for stdout")
	flag.BoolVar(&unixline, "unixline", unixline, "convert all line endings to unix format: newline")
	flag.BoolVar(&errlog, "errlog", errlog, "log failing git call details to stderr")
	flag.BoolVar(&debug, "debug", debug, "print detailed information for arguments and the data from git")
	flag.BoolVar(&help, "help", help, "show this help text")
	flag.Parse()

	helpAndQuit := func(exit int, message string) {
		flag.CommandLine.SetOutput(os.Stderr)
		if message != "" {
			fmt.Fprintf(os.Stderr, "Error: %v\n\n", message)
		}
		fmt.Fprintf(os.Stderr, "Use %s to retrieve versioning information for the repository containing %s\n", os.Args[0], dir)
		fmt.Fprintf(os.Stderr, "Git is used to retrieve the data. It must be available in your PATH.\n")
		fmt.Fprintf(os.Stderr, "Times used in the default template are UTC. Time errors are encoded as unix epoch.\n")
		fmt.Fprintf(os.Stderr, "Uncommitted files result in a version number v0.0.0\n\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "Check https://golang.org/pkg/text/template for a template reference.\n")
		fmt.Fprintf(os.Stderr, "Two functions are supported: Now for the current time and Env to retrieve an environment variable.\n")
		fmt.Fprintf(os.Stderr, "The default template follows these conventions:\n")
		fmt.Fprintf(os.Stderr, "* time is always UTC\n")
		fmt.Fprintf(os.Stderr, "* time errors are encoded as Unix epoch (1970-01-01T00:00:00)\n")
		fmt.Fprintf(os.Stderr, "* unknown revisions consist of a series of \"0\"\n")
		fmt.Fprintf(os.Stderr, "* tracked but uncommitted files are not fit for a release version\n")
		fmt.Fprintf(os.Stderr, "* tracked but not committed files lead to a 0.0.0-... semver as they are not fit for release\n\n")
		fmt.Fprintf(os.Stderr, "The default template looks like this:\n%s\n", defaultTemplate)
		os.Exit(exit)
	}

	if help || len(flag.Args()) > 0 {
		status := 0
		if !help {
			status = ExitOnUsage
		}
		if debug {
			log.Printf("Args: %#v\n", os.Args)
		}
		helpAndQuit(status, "")
	}

	dest := os.Stdout
	if out != "" {
		f, err := os.Create(out)
		if err != nil {
			log.Printf("Could not create output file %q: %v\n", out, err)
			os.Exit(ExitOnCreateFile)
		}
		defer f.Close()
		dest = f
	}

	var (
		tsrc string
		ok   bool
	)

	if tmpl != "" {
		raw, err := ioutil.ReadFile(tmpl)
		if err != nil {
			helpAndQuit(ExitOnTemplate, fmt.Sprintf("template file %q could not be read: %v", tmpl, err))
		}
		tsrc = string(raw)
	} else if tsrc, ok = formats[format]; !ok {
		helpAndQuit(ExitOnTemplate, fmt.Sprintf("template not found for format %q", format))
	}
	t, err := template.New("").Funcs(template.FuncMap{
		"Now": func() time.Time { return time.Now().UTC() },
		"Env": os.Getenv,
		"If": func(cond bool, t, f string) string {
			if cond {
				return t
			}
			return f
		},
	}).Parse(tsrc)
	if err != nil {
		helpAndQuit(ExitOnTemplate, fmt.Sprintf("template could not compile: %v", err))
	}
	buf := bytes.NewBuffer(nil)
	err = t.ExecuteTemplate(buf, tagregexp, nil)
	if err != nil {
		helpAndQuit(ExitOnTemplate, fmt.Sprintf("template lacks sub template %q with semver regexp", tagregexp))
	}
	if dir != "" {
		err := os.Chdir(dir)
		if err != nil {
			helpAndQuit(ExitOnChdir, fmt.Sprintf("could not cd to %q: %v", dir, err))
		}
	}

	var logger interface {
		Printf(string, ...interface{})
	} = discarder{}
	if errlog {
		l := log.Default()
		l.SetOutput(os.Stderr)
		logger = l
	}

	re := buf.String()
	reSemver, err := regexp.Compile(re)
	if err != nil {
		helpAndQuit(ExitOnRegexp, fmt.Sprintf("regexp error for %q: %v", re, err))
	}

	c, err := NewCommitInfo(ref, reSemver)
	if err != nil {
		helpAndQuit(ExitOnCommand, fmt.Sprintf("status retrieval failed: %v", err))
	}

	if setversion != "" {
		if reSemver.MatchString(setversion) {
			c.Semver = setversion
			c.Clean = true
		} else {
			logger.Printf("Version warning: using detected %q, not %q; it did not match %q\n", c.Semver, setversion, re)
		}
	}

	if debug {
		logger.Printf("Regexp: %s\n", re)
		logger.Printf("Git: %#v\n", c)
	}

	buf.Reset()
	err = t.Execute(buf, c)
	if err != nil {
		helpAndQuit(ExitOnTemplate, fmt.Sprintf("template did not render: %v", err))
	}
	rendered := buf.String()
	if unixline {
		rendered = strings.ReplaceAll(rendered, "\r\n", "\n")
	}
	fmt.Fprint(dest, rendered)
}
