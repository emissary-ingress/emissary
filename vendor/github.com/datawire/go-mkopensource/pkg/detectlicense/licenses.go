package detectlicense

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

type License struct {
	Name           string
	NoticeFile     bool // are NOTICE files "a thing" for this license?
	WeakCopyleft   bool // requires that library to be open-source
	StrongCopyleft bool // requires the resulting program to be open-source
}

var (
	Proprietary = License{Name: "proprietary"}

	PublicDomain = License{Name: "public domain"}

	Apache2 = License{Name: "Apache License 2.0", NoticeFile: true}
	BSD1    = License{Name: "1-clause BSD license"}
	BSD2    = License{Name: "2-clause BSD license"}
	BSD3    = License{Name: "3-clause BSD license"}
	ISC     = License{Name: "ISC license"}
	MIT     = License{Name: "MIT license"}
	MPL2    = License{Name: "Mozilla Public License 2.0", NoticeFile: true, WeakCopyleft: true}

	CcBySa40 = License{Name: "Creative Commons Attribution Share Alike 4.0 International", StrongCopyleft: true}
)

// https://spdx.org/licenses/
var (
	// split with "+" to avoid a false-positive on itself
	spdxTag = []byte("SPDX-License" + "-Identifier:")

	spdxIdentifiers = map[string]License{
		"Apache-2.0":   Apache2,
		"BSD-1-Clause": BSD1,
		"BSD-2-Clause": BSD2,
		"BSD-3-Clause": BSD3,
		"ISC":          ISC,
		"MIT":          MIT,
		"MPL-2.0":      MPL2,
		"CC-BY-SA-4.0": CcBySa40,
	}
)

func expectsNotice(licenses map[License]struct{}) bool {
	for license := range licenses {
		if license.NoticeFile {
			return true
		}
	}
	return false
}

func DetectLicenses(files map[string][]byte) (map[License]struct{}, error) {
	licenses := make(map[License]struct{})
	hasNotice := false
	hasLicenseFile := false
	hasNonSPDXSource := false
	hasPatents := false

loop:
	for filename, filebody := range files {

		switch filename {
		case "github.com/miekg/dns/COPYRIGHT":
			// This file identifies copyright holders, but
			// the license info is in the LICENSE file.
			continue loop
		case "sigs.k8s.io/kustomize/kyaml/LICENSE_TEMPLATE":
			// This is a template file for generated code,
			// not an actual license file.
			continue loop
		}

		name := filepath.Base(filename)
		// See ./vendor.go:metaPrefixes
		switch {
		case strings.HasPrefix(name, "AUTHORS") ||
			strings.HasPrefix(name, "CONTRIBUTORS"):
			// Ignore this file; it does not identify a license.
		case strings.HasPrefix(name, "COPYLEFT") ||
			strings.HasPrefix(name, "COPYING") ||
			strings.HasPrefix(name, "COPYRIGHT") ||
			strings.HasPrefix(name, "LEGAL") ||
			strings.HasPrefix(name, "LICENSE"):
			ls := IdentifyLicenses(filebody)
			if len(ls) == 0 {
				return nil, fmt.Errorf("could not identify license in file %q", filename)
			}
			if name == "LICENSE.docs" && len(ls) == 1 {
				if _, isCc := ls[CcBySa40]; isCc {
					// This file describes the license of the
					// docs, which are licensed separately from
					// the code.  We don't care about the docs.
					continue
				}
			}
			for l := range ls {
				licenses[l] = struct{}{}
			}
			hasLicenseFile = true
		case strings.HasPrefix(name, "NOTICE"):
			hasNotice = true
		case strings.HasPrefix(name, "PATENTS"):
			// ignore this file, for now
			hasPatents = true
		default:
			// This is a source file; look for an SPDX
			// identifier.
			ls, err := IdentifySPDXLicenses(filebody)
			if err != nil {
				return nil, err
			}
			if len(ls) == 0 {
				hasNonSPDXSource = true
			}
			for l := range ls {
				licenses[l] = struct{}{}
			}
		}
	}

	if !expectsNotice(licenses) && hasNotice {
		return nil, errors.New("the NOTICE file is really only for the Apache 2.0 and MPL 2.0 licenses; something hokey is going on")
	}
	if _, hasApache := licenses[Apache2]; hasApache && hasPatents {
		// TODO: Check if the MPL has a patent grant.  A quick skimming says "seems to explicitly say no", but I'm too tired to actually read the thing.
		return nil, errors.New("the Apache license contains a patent-grant, but there's a separate PATENTS file; something hokey is going on")
	}
	if !hasLicenseFile && hasNonSPDXSource {
		return nil, errors.New("could not identify a license for all sources (had no global LICENSE file)")
	}

	if len(licenses) == 0 {
		panic(errors.New("should not happen"))
	}
	return licenses, nil
}

// IdentifySPDX takes the contents of a source-file and looks for SPDX
// license identifiers.
func IdentifySPDXLicenses(body []byte) (map[License]struct{}, error) {
	licenses := make(map[License]struct{})
	for bytes.Contains(body, spdxTag) {
		tagPos := bytes.Index(body, spdxTag)
		body = body[tagPos+len(spdxTag):]
		idEnd := bytes.IndexByte(body, '\n')
		if idEnd < 0 {
			idEnd = len(body)
		}
		id := string(body[:idEnd])
		body = body[idEnd:]

		id = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(id), "*/"))
		license, licenseOK := spdxIdentifiers[id]
		if !licenseOK {
			return nil, fmt.Errorf("unknown SPDX identifier %q", id)
		}
		licenses[license] = struct{}{}
	}
	return licenses, nil
}

var (
	bsd3funnyAttributionLines = []string{
		`(?:Copyright [^\n]*(?:\s+All rights reserved\.)? *\n)`,
		`As this is fork of the official Go code the same license applies[.:]`,
		reQuote(`Extensions of the original work are copyright (c) 2011 Miek Gieben`),
		reQuote(`Go support for Protocol Buffers - Google's data interchange format`),
		reQuote(`Protocol Buffers for Go with Gadgets`),
		reQuote(`http://github.com/gogo/protobuf`),
		reQuote(`https://github.com/golang/protobuf`),
		reQuote(`Copyright (c) 2012 Péter Surányi. Portions Copyright (c) 2009 The Go
Authors. All rights reserved.`),
	}
)

const (
	rackspaceHeader = `Copyright 2012-2013 Rackspace, Inc.

Licensed under the Apache License, Version 2.0 (the "License"); you may not use
this file except in compliance with the License.  You may obtain a copy of the
License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed
under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
CONDITIONS OF ANY KIND, either express or implied.  See the License for the
specific language governing permissions and limitations under the License.                                ` + /* trailing ws matters */ `

------`

	xzPublicDomain = `Licensing of github.com/xi2/xz
==============================

    This Go package is a modified version of

        XZ Embedded  <http://tukaani.org/xz/embedded.html>

    The contents of the testdata directory are modified versions of
    the test files from

        XZ Utils  <http://tukaani.org/xz/>

    All the files in this package have been written by Michael Cross,
    Lasse Collin and/or Igor PavLov. All these files have been put
    into the public domain. You can do whatever you want with these
    files.

    This software is provided "as is", without any warranty.
`
)

var (
	yamlHeader = reWrap(`The following files were ported to Go from C files of libyaml, and thus
are still covered by their original (copyright and license|MIT license, with the additional
copyright start?ing in 2011 when the project was ported over):

    apic\.go
    emitterc\.go
    parserc\.go
    readerc\.go
    scannerc\.go
    writerc\.go
    yamlh\.go
    yamlprivateh\.go

`)
	reYamlV2 = reCompile(yamlHeader + `\s*` + reMIT.String())

	reYamlV3 = reCompile(`\s*` +
		reQuote(`This project is covered by two different licenses: MIT and Apache.`) + `\s*` +
		`#+ MIT License #+\s*` +
		yamlHeader + `\s*` +
		reMIT.String() + `\s*` +
		`#+ Apache License #+\s*` +
		reQuote(`All the remaining project files are covered by the Apache license:`) + `\s*` +
		reApacheStatement.String())
)

// IdentifyLicense takes the contents of a license-file and attempts
// to identify the license(s) in it.  If it is even a little unsure,
// it returns nil.
func IdentifyLicenses(body []byte) map[License]struct{} {
	licenses := make(map[License]struct{})

	switch {

	case reMatch(reApacheLicense, body) || reMatch(reApacheStatement, body):
		licenses[Apache2] = struct{}{}
	case reMatch(reBSD2, body):
		licenses[BSD2] = struct{}{}
	case reMatch(reBSD3, body):
		licenses[BSD3] = struct{}{}
	case reMatch(reISC, body):
		licenses[ISC] = struct{}{}
	case reMatch(reMIT, body):
		licenses[MIT] = struct{}{}
	case reMatch(reMPL, body):
		licenses[MPL2] = struct{}{}
	case reMatch(reCcBySa40, body):
		licenses[CcBySa40] = struct{}{}

	// special-purpose hacks
	case reMatch(reCompile(fmt.Sprintf(`%s\n-+\n+AVL Tree:\n+%s`, reBSD2, reISC)), body):
		// github.com/emirpasic/gods/LICENSE
		licenses[BSD2] = struct{}{}
		licenses[ISC] = struct{}{}
	case reMatch(reCompile(``+
		`(?:`+strings.Join(bsd3funnyAttributionLines, `\s*|`)+`\s*)*`+
		reWrap(``+
			bsdPrefix+
			bsdClause1+
			bsdClause2+
			bsdClause3+
			bsdSuffix)+
		`(?:\s*`+strings.Join(bsd3funnyAttributionLines, `|\s*`)+`)*\s*`),
		body):
		// github.com/gogo/protobuf/LICENSE
		// github.com/src-d/gcfg/LICENSE
		// github.com/miekg/dns/LICENSE
		licenses[BSD3] = struct{}{}
	case reMatch(reCompile(reQuote(rackspaceHeader)+reApacheLicense.String()), body):
		// github.com/gophercloud/gophercloud/LICENSE
		licenses[Apache2] = struct{}{}
	case reMatch(reCompile(fmt.Sprintf(`%s=*\s*The lexer and parser[^\n]*\n[^\n]*below\.%s`, reMIT, reMIT)), body):
		// github.com/kevinburke/ssh_config/LICENSE
		licenses[MIT] = struct{}{}
	case reMatch(reCompile(`Blackfriday is distributed under the Simplified BSD License:\s*`+reBSD2.String()), regexp.MustCompile(`>\s*`).ReplaceAllLiteral(body, []byte{})):
		// gopkg.in/russross/blackfriday.v2/LICENSE.txt
		licenses[BSD2] = struct{}{}
	case reMatch(reYamlV2, body):
		licenses[MIT] = struct{}{}
	case reMatch(reYamlV3, body):
		licenses[MIT] = struct{}{}
		licenses[Apache2] = struct{}{}
	case reMatch(reCompile(reMIT.String()+`\s*`+reBSD3.String()), body):
		// sigs.k8s.io/yaml/LICENSE
		licenses[MIT] = struct{}{}
		licenses[BSD3] = struct{}{}
	case reMatch(reCompile(reMIT.String()+`\s*- Based on \S*, which has the following license:\n"""\s*`+reMIT.String()+`\s*"""\s*`), body):
		// github.com/shopspring/decimal/LICENSE
		licenses[MIT] = struct{}{}
	case string(body) == xzPublicDomain:
		// github.com/xi2/xz/LICENSE
		licenses[PublicDomain] = struct{}{}
	default:
		return nil
	}

	return licenses
}
