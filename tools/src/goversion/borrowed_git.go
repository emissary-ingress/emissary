// This file is a lightly modified subset of Go 1.17 cmd/go/internal/modfetch/codehost/git.go

package main

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

type commitInfo struct {
	Hash string
	Time time.Time
	Tags []string
}

// statLocal is based on and closely mimics Go 1.17
// cmd/go/internal/modfetch/codehost/git.go:gitRepo.statLocal().
func statLocal(ctx context.Context, commitish string) (*commitInfo, error) {
	if strings.HasPrefix(commitish, "-") {
		return nil, &UnknownRevisionError{Rev: commitish}
	}
	out, err := cmdOutput(ctx, "git", "-c", "log.showsignature=false", "log", "-n1", "--format=format:%H %ct %D", commitish, "--")
	if err != nil {
		return nil, &UnknownRevisionError{Rev: commitish}
	}
	f := strings.Fields(out)
	if len(f) < 2 {
		return nil, fmt.Errorf("unexpected response from git log: %q", out)
	}
	hash := f[0]
	t, err := strconv.ParseInt(f[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid time from git log: %q", out)
	}

	info := commitInfo{
		Hash: hash,
		Time: time.Unix(t, 0).UTC(),
	}

	// Add tags. Output looks like:
	//	ede458df7cd0fdca520df19a33158086a8a68e81 1523994202 HEAD -> master, tag: v1.2.4-annotated, tag: v1.2.3, origin/master, origin/HEAD
	for i := 2; i < len(f); i++ {
		if f[i] == "tag:" {
			i++
			if i < len(f) {
				info.Tags = append(info.Tags, strings.TrimSuffix(f[i], ","))
			}
		}
	}
	sort.Strings(info.Tags)

	return &info, nil
}

// forEachReachableTag is based on and closely mimics Go 1.17
// cmd/go/internal/modfetch/codehost/git.go:gitRepo.RecentTag().
//
// forEachReachableTag iterates over all Git tags matching "refs/tags/{dirPrefix}v{SEMVER}" that are
// reachable-from (self-or-ancestor-of) the given `commit`, calling `cb` on each "v{SEMVER}" string
// (with any "refs/tags/" or {dirPrefix} prefix trimmed off).  The iteration happens in no
// particular order.
func forEachReachableTag(ctx context.Context, commit *commitInfo, dirPrefix string, cb func(string)) error {
	refPrefix := "refs/tags/" + dirPrefix
	out, err := cmdOutput(ctx, "git", "for-each-ref",
		"--format=%(refname)",
		"--merged="+commit.Hash,
		refPrefix)
	if err != nil {
		return err
	}

	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)

		// Git does support lstrip in for-each-ref format, but it was added in v2.13.0.
		// Stripping here instead gives support for git v2.7.0.
		if !strings.HasPrefix(line, refPrefix) {
			// This check mostly just handles the trailing blank line and could be
			// `if line != ""`, but let's have the full HasPrefix check just to be safe.
			continue
		}
		semtag := line[len(refPrefix):]

		// Consider only tags that are valid and complete (not just major.minor prefixes).
		if c := semver.Canonical(semtag); c == "" || !strings.HasPrefix(semtag, c) {
			continue
		}
		cb(semtag)
	}
	return nil
}

// mostRecentTag is based on and closely mimics Go 1.17
// cmd/go/internal/modfetch/codehost/git.go:gitRepo.RecentTag().
//
// The word "recent" is a little bit of a lie; it's based on semver ordering, not commit timestamps.
func mostRecentTag(ctx context.Context, commit *commitInfo, dirPrefix string) (string, error) {
	var highest string
	err := forEachReachableTag(ctx, commit, dirPrefix, func(semtag string) {
		// NOTE: Do not replace the call to semver.Compare with semver.Max.
		// We want to return the actual tag, not a canonicalized version of it,
		// and semver.Max currently canonicalizes (see golang.org/issue/32700).
		if semver.Compare(semtag, highest) > 0 {
			highest = semtag
		}
	})
	if err != nil {
		return "", err
	}
	if highest == "" {
		return "", fmt.Errorf("no %q tags are reachable from %q",
			dirPrefix+"v{SEMVER}", commit.Hash)
	}
	return dirPrefix + highest, nil
}

// mostRecentTags is like mostRecentTag, but returns a list of tags ordered most-recent-first,
// rather than returning the singular most-recent tag.
//
// Like with mostRecentTag, the word "recent" is a little bit of a lie; it's based on semver
// ordering, not commit timestamps.
func mostRecentTags(ctx context.Context, commit *commitInfo, dirPrefix string) ([]string, error) {
	var semtags []string
	err := forEachReachableTag(ctx, commit, dirPrefix, func(semtag string) {
		semtags = append(semtags, semtag)
	})
	if err != nil {
		return nil, err
	}
	if len(semtags) == 0 {
		return nil, fmt.Errorf("no %q tags are reachable from %q",
			dirPrefix+"v{SEMVER}", commit.Hash)
	}
	sort.SliceStable(semtags, func(i, j int) bool {
		return semver.Compare(semtags[i], semtags[j]) > 0
	})
	return semtags, nil
}
