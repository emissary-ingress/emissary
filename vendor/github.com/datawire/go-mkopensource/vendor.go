package main

// This file mimics the behavior of `go mod vendor`.

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/datawire/go-mkopensource/pkg/golist"
)

////////////////////////////////////////////////////////////////////////
// To simplify things, don't try to optimize by avoiding traversing
// the same file repeatedly; instead, just maintain a global cache of
// filesystem access.

var (
	fsFileCache = make(map[string][]byte)
	fsDirCache  = make(map[string][]os.FileInfo)
)

func readFile(filename string) ([]byte, error) {
	if _, done := fsFileCache[filename]; !done {
		body, err := ioutil.ReadFile(filename)
		if err != nil {
			return nil, err
		}
		fsFileCache[filename] = body
	}
	return fsFileCache[filename], nil
}

func readDir(dirname string) ([]os.FileInfo, error) {
	if _, done := fsDirCache[dirname]; !done {
		body, err := ioutil.ReadDir(dirname)
		if err != nil {
			return nil, err
		}
		fsDirCache[dirname] = body
	}
	return fsDirCache[dirname], nil
}

////////////////////////////////////////////////////////////////////////
// Mimicing `go mod vendor`'s structure, one of our core abstractions
// within the file is to collect the files in a directory that fit
// some 'match'er criteria.

// collectDir mimics
// /usr/lib/go/src/cmd/go/internal/modcmd/vendor.go:copyDir(),
// but:
//  1. instead of copying them to a ./vendor/ directory, it adds them
//     to the in-memory 'vendor' map.
//  2. the match() function takes a full filename, instead of a
//     (dirname, os.FileInfo) tuple.
func collectDir(vendor map[string][]byte, dst, src string, match func(filename string) bool) error {
	files, err := readDir(src)
	if err != nil {
		return err
	}
	for _, file := range files {
		filename := filepath.Join(src, file.Name())
		if file.IsDir() || !file.Mode().IsRegular() || !match(filename) {
			continue
		}
		body, err := readFile(filename)
		if err != nil {
			return err
		}
		vendor[filepath.Join(dst, file.Name())] = body
	}
	return nil
}

////////////////////////////////////////////////////////////////////////
// Matchers for collectDir

// metaPrefixes is copied from
// /usr/lib/go/src/cmd/go/internal/modcmd/vendor.go
//
// metaPrefixes is the list of metadata file prefixes.
// Vendoring copies metadata files from parents of copied directories.
// Note that this list could be arbitrarily extended, and it is longer
// in other tools (such as godep or dep). By using this limited set of
// prefixes and also insisting on capitalized file names, we are trying
// to nudge people toward more agreement on the naming
// and also trying to avoid false positives.
var metaPrefixes = []string{
	"AUTHORS",
	"CONTRIBUTORS",
	"COPYLEFT",
	"COPYING",
	"COPYRIGHT",
	"LEGAL",
	"LICENSE",
	"NOTICE",
	"PATENTS",
}

// matchMetadata mimics
// /usr/lib/go/src/cmd/go/internal/modcmd/vendor.go:matchMetadata().
func matchMetadata(filename string) bool {
	name := filepath.Base(filename)
	for _, p := range metaPrefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

// matchSourceFiles is a little bit stricter than
// /usr/lib/go/src/cmd/go/internal/modcmd/vendor.go:matchPotentialSourceFiles().
// I think that's OK.
func matchSourceFiles(pkgInfo golist.Package) func(filename string) bool {
	var sourceFiles []string
	sourceFiles = append(sourceFiles, pkgInfo.GoFiles...)
	sourceFiles = append(sourceFiles, pkgInfo.CgoFiles...)
	sourceFiles = append(sourceFiles, pkgInfo.CFiles...)
	sourceFiles = append(sourceFiles, pkgInfo.CXXFiles...)
	sourceFiles = append(sourceFiles, pkgInfo.MFiles...)
	sourceFiles = append(sourceFiles, pkgInfo.HFiles...)
	sourceFiles = append(sourceFiles, pkgInfo.FFiles...)
	sourceFiles = append(sourceFiles, pkgInfo.SFiles...)
	sourceFiles = append(sourceFiles, pkgInfo.SwigFiles...)
	sourceFiles = append(sourceFiles, pkgInfo.SwigCXXFiles...)
	sourceFiles = append(sourceFiles, pkgInfo.SysoFiles...)
	return func(filename string) bool {
		needle := filepath.Base(filename)
		for _, straw := range sourceFiles {
			if straw == needle {
				return true
			}
		}
		return false
	}
}

func matchAll(string) bool {
	return true
}

////////////////////////////////////////////////////////////////////////
// OK, this is where it gets hard to build sane abstractions, and we
// start having to think about global program operations.

// collectPkg mimics
// /usr/lib/go/src/cmd/go/internal/modcmd/vendor.go:vendorPkg().
func collectPkg(vendor map[string][]byte, pkgInfo golist.Package) error {
	dst := pkgInfo.ImportPath
	src := pkgInfo.Dir
	err := collectDir(vendor, dst, src, matchSourceFiles(pkgInfo))
	if err != nil {
		return err
	}
	return collectMetadata(vendor, pkgInfo.Module.Path, dst, src)
}

// collectMetadata mimics
// /usr/lib/go/src/cmd/go/internal/modcmd/vendor.go:copyMetadata().
func collectMetadata(ret map[string][]byte, modPath, dst, src string) error {
	for {
		err := collectDir(ret, dst, src, matchMetadata)
		if err != nil {
			return err
		}
		if dst == modPath {
			return nil
		}
		dst = filepath.Dir(dst)
		src = filepath.Dir(src)
	}
}

// collectVendoredPkg is like collectPkg, but behaves as if
// `-mod=vendor`; inspecting the `vendor/` directory instead of the
// module cache.  The point of that is that we use matchAll instead of
// matchSourcefiles, because
//  1. we trust `go mod vendor` to have already pruned out files we
//     don't want, and
//  2. VendorList() doesn't populate the `pkgInfo.{Whatever}Files`
//     variables.
func collectVendoredPkg(vendor map[string][]byte, pkgInfo golist.Package) error {
	dst := pkgInfo.ImportPath
	src := pkgInfo.Dir
	err := collectDir(vendor, dst, src, matchAll)
	if err != nil {
		return err
	}
	return collectMetadata(vendor, pkgInfo.Module.Path, dst, src)
}
