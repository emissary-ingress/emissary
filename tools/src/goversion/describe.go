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
