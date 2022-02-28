package main

import (
	"context"
	"os"
	"os/exec"
	"path"
	"strings"

	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
)

func cmdOutput(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stderr = os.Stderr
	bs, err := cmd.Output()
	return string(bs), err
}

// Describe looks at the Git `commitish` and Git `vSEMVER` tags and returns a Go-modules-compatible
// version string that refers to that commitish.
//
// If dirPrefix is non-empty, then Describe considers tags named `path.Join(dirPrefix, "vSEMVER")`
// rather than `vSEMVER`.  Unlike Go itself, Describe does not validate that a `{dirPrefix}/go.mod`
// file exist in the tree referred to by `commitish`.
//
// If dirtyMarker is non-empty, then Describe checks if the current Git tree is dirty and if so
// appends dirtyMarker to the returned version string.  Strictly speaking, this makes the version
// string not Go-modules-compatible.
func Describe(ctx context.Context, commitish, dirPrefix, dirtyMarker string) (string, error) {
	if dirPrefix != "" {
		dirPrefix = path.Clean(dirPrefix) + "/"
	}

	commitInfo, err := statLocal(ctx, commitish)
	if err != nil {
		return "", err
	}

	parentTag, err := mostRecentTag(ctx, commitInfo, dirPrefix)
	if err != nil {
		return "", err
	}

	parentTagInfo, err := statLocal(ctx, parentTag)
	if err != nil {
		return "", err
	}

	isDirty := false
	if dirtyMarker != "" {
		out, err := cmdOutput(ctx, "git", "status", "--porcelain")
		if err != nil {
			return "", err
		}
		isDirty = len(out) > 0
	}

	goVersionStr := strings.TrimPrefix(parentTag, dirPrefix)
	// The '|| isDirty' here is important so that the dirtyMarker parses as a *post*-release
	// rather than as a pre-release.
	if parentTagInfo.Hash != commitInfo.Hash || isDirty {
		goVersionStr = module.PseudoVersion(
			semver.Major(goVersionStr),
			goVersionStr,
			commitInfo.Time,
			ShortenSHA1(commitInfo.Hash))
	}

	if isDirty {
		goVersionStr += dirtyMarker
	}

	return goVersionStr, nil
}
