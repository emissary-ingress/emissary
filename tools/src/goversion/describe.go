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

// Describe looks at the Git `commitish` and Git `vSEMVER` tags and returns a list of
// Go-modules-compatible version strings that refers to that commitish.
//
// If dirPrefix is non-empty, then Describe considers tags named `path.Join(dirPrefix, "vSEMVER")`
// rather than `vSEMVER`.  Unlike Go itself, Describe does not validate that a `{dirPrefix}/go.mod`
// file exist in the tree referred to by `commitish`.
//
// If dirtyMarker is non-empty, then Describe checks if the current Git tree is dirty and if so
// appends dirtyMarker to the returned version strings.  Strictly speaking, this makes the version
// strings not Go-modules-compatible.
//
// maxDescriptions limits how many version strings to return; a value of <=0 is no limit.  For each
// ancestral tag, there is a distinct way to refer to the same commit; usually the highest version
// number (semver comparison) is the one you want; but sometimes it is useful to have the others.
// The strings returned are ordered highest-version-first.
func Describe(ctx context.Context, commitish, dirPrefix, dirtyMarker string, maxDescriptions int) ([]string, error) {
	if dirPrefix != "" {
		dirPrefix = path.Clean(dirPrefix) + "/"
	}

	commitInfo, err := statLocal(ctx, commitish)
	if err != nil {
		return nil, err
	}

	var parentTags []string
	if maxDescriptions == 1 {
		var parentTag string
		parentTag, err = mostRecentTag(ctx, commitInfo, dirPrefix)
		parentTags = []string{parentTag}
	} else {
		parentTags, err = mostRecentTags(ctx, commitInfo, dirPrefix)
	}
	if err != nil {
		return nil, err
	}

	isDirty := false
	if dirtyMarker != "" {
		out, err := cmdOutput(ctx, "git", "status", "--porcelain")
		if err != nil {
			return nil, err
		}
		isDirty = len(out) > 0
	}

	var descriptions []string
	for _, parentTag := range parentTags {
		parentTagInfo, err := statLocal(ctx, parentTag)
		if err != nil {
			return nil, err
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

		descriptions = append(descriptions, goVersionStr)
		if maxDescriptions > 0 && len(descriptions) >= maxDescriptions {
			break
		}
	}

	return descriptions, nil
}
