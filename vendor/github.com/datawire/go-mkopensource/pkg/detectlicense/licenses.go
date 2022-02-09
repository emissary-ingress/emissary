package detectlicense

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

type LicenseRestriction int

const (
	Forbidden LicenseRestriction = iota
	AmbassadorServers
	Unrestricted
)

type License struct {
	Name           string
	NoticeFile     bool               // are NOTICE files "a thing" for this license?
	WeakCopyleft   bool               // requires that library to be open-source
	StrongCopyleft bool               // requires the resulting program to be open-source
	URL            string             // Location of the license description
	Restriction    LicenseRestriction // Where is this license allowed
}

//nolint:gochecknoglobals // Would be 'const'.
var (
	AmbassadorProprietary = License{Name: "proprietary Ambassador software"}
	ZeroBSD               = License{Name: "BSD Zero Clause License",
		URL: "https://spdx.org/licenses/0BSD.html", Restriction: Forbidden}
	Apache2 = License{Name: "Apache License 2.0", NoticeFile: true,
		URL: "https://opensource.org/licenses/Apache-2.0", Restriction: Forbidden}
	AFL21 = License{Name: "Academic Free License v2.1", URL: "https://spdx.org/licenses/AFL-2.1.html",
		Restriction: Forbidden}
	AGPL1Only    = License{Name: "Affero General Public License v1.0 only", Restriction: Forbidden}
	AGPL1OrLater = License{Name: "Affero General Public License v1.0 or later", Restriction: Forbidden}
	AGPL3Only    = License{Name: "GNU Affero General Public License v3.0 only", Restriction: Forbidden}
	AGPL3OrLater = License{Name: "GNU Affero General Public License v3.0 or later", Restriction: Forbidden}
	BSD1         = License{Name: "1-clause BSD license", URL: "https://opensource.org/licenses/BSD-1-Clause",
		Restriction: Forbidden}
	BSD2 = License{Name: "2-clause BSD license", URL: "https://opensource.org/licenses/BSD-2-Clause",
		Restriction: Forbidden}
	BSD3 = License{Name: "3-clause BSD license", URL: "https://opensource.org/licenses/BSD-3-Clause",
		Restriction: Forbidden}
	CcBy30 = License{Name: "Creative Commons Attribution 3.0 Unported",
		URL: "https://spdx.org/licenses/CC-BY-3.0.html", Restriction: Forbidden}
	CcBy40 = License{Name: "Creative Commons Attribution 4.0 International",
		URL: "https://spdx.org/licenses/CC-BY-4.0.html", Restriction: Forbidden}
	CcBySa40 = License{Name: "Creative Commons Attribution Share Alike 4.0 International",
		StrongCopyleft: true, URL: "https://spdx.org/licenses/CC-BY-SA-4.0.html", Restriction: Forbidden}
	Cc010 = License{Name: "Creative Commons Zero v1.0 Universal",
		URL: "https://spdx.org/licenses/CC0-1.0.html", Restriction: Forbidden}
	EPL10 = License{Name: "Eclipse Public License 1.0", URL: "https://spdx.org/licenses/EPL-1.0.html",
		Restriction: Forbidden}
	GPL1Only = License{Name: "GNU General Public License v1.0 only",
		URL: "https://spdx.org/licenses/GPL-1.0-only.html", Restriction: Forbidden}
	GPL1OrLater = License{Name: "GNU General Public License v1.0 or later",
		URL: "https://spdx.org/licenses/GPL-1.0-or-later.html", Restriction: Forbidden}
	GPL2Only = License{Name: "GNU General Public License v2.0 only",
		URL: "https://spdx.org/licenses/GPL-2.0-only.html", Restriction: Forbidden}
	GPL2OrLater = License{Name: "GNU General Public License v2.0 or later",
		URL: "https://spdx.org/licenses/GPL-2.0-or-later.html", Restriction: Forbidden}
	GPL3Only = License{Name: "GNU General Public License v3.0 only", StrongCopyleft: true,
		URL: "https://spdx.org/licenses/GPL-3.0.html", Restriction: Forbidden}
	GPL3OrLater = License{Name: "GNU General Public License v3.0 or later",
		URL: "https://spdx.org/licenses/GPL-3.0-or-later.html", Restriction: Forbidden}
	ISC       = License{Name: "ISC license", URL: "https://opensource.org/licenses/ISC", Restriction: Forbidden}
	LGPL2Only = License{Name: "GNU Library General Public License v2 only", WeakCopyleft: true,
		Restriction: Forbidden}
	LGPL2OrLater = License{Name: "GNU Library General Public License v2 or later", WeakCopyleft: true,
		Restriction: Forbidden}
	LGPL21Only = License{Name: "GNU Lesser General Public License v2.1 only", WeakCopyleft: true,
		Restriction: Forbidden}
	LGPL21OrLater = License{Name: "GNU Lesser General Public License v2.1 or later", WeakCopyleft: true,
		URL: "https://spdx.org/licenses/LGPL-2.1-or-later.html", Restriction: Forbidden}
	LGPL3Only = License{Name: "GNU Lesser General Public License v3.0 only", WeakCopyleft: true,
		Restriction: Forbidden}
	LGPL3OrLater = License{Name: "GNU Lesser General Public License v3.0 or later", WeakCopyleft: true,
		Restriction: Forbidden}
	MIT   = License{Name: "MIT license", URL: "https://opensource.org/licenses/MIT", Restriction: Forbidden}
	MPL11 = License{Name: "Mozilla Public License 1.1", NoticeFile: true,
		WeakCopyleft: true, URL: "https://spdx.org/licenses/MPL-1.1.html", Restriction: Forbidden}
	MPL2 = License{Name: "Mozilla Public License 2.0", NoticeFile: true,
		WeakCopyleft: true, URL: "https://opensource.org/licenses/MPL-2.0", Restriction: Forbidden}
	ODCBy10 = License{Name: "Open Data Commons Attribution License v1.0", URL: "https://spdx.org/licenses/ODC-By-1.0.html",
		Restriction: Forbidden}
	OFL11 = License{Name: "SIL Open Font License 1.1", URL: "https://spdx.org/licenses/OFL-1.1.html",
		Restriction: Forbidden}
	Python20 = License{Name: "Python License 2.0", URL: "https://spdx.org/licenses/Python-2.0.html",
		Restriction: Forbidden}
	PSF = License{Name: "Python Software Foundation license", URL: "https://spdx.org/licenses/PSF-2.0.html",
		Restriction: Forbidden}
	PublicDomain = License{Name: "Public domain", Restriction: Forbidden}
	Unicode2015  = License{Name: "Unicode License Agreement for Data Files and Software (2015)",
		URL: "https://spdx.org/licenses/Unicode-DFS-2015.html", Restriction: Forbidden}
	Unlicense = License{Name: "The Unlicense",
		URL: "https://spdx.org/licenses/Unlicense.html", Restriction: Forbidden}
	WTFPL = License{Name: "Do What The F*ck You Want To Public License",
		URL: "https://spdx.org/licenses/WTFPL.html", Restriction: Forbidden}
)

// https://spdx.org/licenses/
//
//nolint:gochecknoglobals // Would be 'const'.
var (
	// split with "+" to avoid a false-positive on itself
	spdxTag = []byte("SPDX-License" + "-Identifier:")

	SpdxIdentifiers = map[string]License{
		"0BSD":              ZeroBSD,
		"Apache-2.0":        Apache2,
		"AFL-2.1":           AFL21,
		"AGPL-1.0-only":     AGPL1Only,
		"AGPL-1.0-or-later": AGPL1OrLater,
		"AGPL-3.0-only":     AGPL3Only,
		"AGPL-3.0-or-later": AGPL3OrLater,
		"BSD-1-Clause":      BSD1,
		"BSD-2-Clause":      BSD2,
		"BSD-3-Clause":      BSD3,
		"CC-BY-3.0":         CcBy30,
		"CC-BY-4.0":         CcBy40,
		"CC-BY-SA-4.0":      CcBySa40,
		"CC0-1.0":           Cc010,
		"EPL-1.0":           EPL10,
		"GPL-1.0-only":      GPL1Only,
		"GPL-1.0-or-later":  GPL1OrLater,
		"GPL-2.0-only":      GPL2Only,
		"GPL-2.0-or-later":  GPL2OrLater,
		"GPL-3.0-only":      GPL3Only,
		"GPL-3.0-or-later":  GPL3OrLater,
		"ISC":               ISC,
		"LGPL-2.0-only":     LGPL2Only,
		"LGPL-2.0-or-later": LGPL2OrLater,
		"LGPL-2.1-only":     LGPL21Only,
		"LGPL-2.1-or-later": LGPL21OrLater,
		"LGPL-3.0-only":     LGPL3Only,
		"LGPL-3.0-or-later": LGPL3OrLater,
		"MIT":               MIT,
		"MPL-1.1":           MPL11,
		"MPL-2.0":           MPL2,
		"ODC-By-1.0":        ODCBy10,
		"OFL-1.1":           OFL11,
		"PSF-2.0":           PSF,
		"Python-2.0":        Python20,
		"Unicode-DFS-2015":  Unicode2015,
		"Unlicense":         Unlicense,
		"WTFPL":             WTFPL,
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

func DetectLicenses(packageName string, packageVersion string, files map[string][]byte) (map[License]struct{}, error) {

	if isAmbassadorProprietarySoftware(packageName) {
		// Ambassador's proprietary software has a proprietary license
		softwareLicenses := map[License]struct{}{AmbassadorProprietary: {}}
		return softwareLicenses, nil
	}

	if knownDependencies, isKnown := knownDependencies(packageName, packageVersion); isKnown {
		licenses := map[License]struct{}{}
		for _, license := range knownDependencies {
			licenses[license] = struct{}{}
		}
		return licenses, nil
	}

	licenses := make(map[License][]string)
	hasNotice := false
	hasLicenseFile := false
	hasNonSPDXSource := false
	patents := []string(nil)

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
		case "github.com/telepresenceio/telepresence/v2/LICENSES.md":
			// Licenses for telepresence are in LICENSE and not in LICENSES.md
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
				licenses[l] = append(licenses[l], filename)
			}
			hasLicenseFile = true
		case strings.HasPrefix(name, "NOTICE"):
			hasNotice = true
		case strings.HasPrefix(name, "PATENTS"):
			// ignore this file, for now
			patents = append(patents, filename)
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
				licenses[l] = append(licenses[l], filename)
			}
		}
	}

	bareLicenses := make(map[License]struct{}, len(licenses))
	for license := range licenses {
		bareLicenses[license] = struct{}{}
	}

	if !expectsNotice(bareLicenses) && hasNotice {
		return nil, errors.New("the NOTICE file is really only for the Apache 2.0 and MPL 2.0 licenses; something hokey is going on")
	}
	for _, patentFile := range patents {
		// TODO: Check if the MPL has a patent grant.  A quick skimming says "seems to explicitly say no", but
		// I'm too tired to actually read the thing.
		if _, hasApache := licenses[Apache2]; hasApache {
			dir := filepath.Dir(patentFile)
			// We want to blow up if an Apache-licensed thing has a PATENTS file.  But let it through if a
			// subdirectory of an otherwise-Apache-licensed thing has a different license and includes a
			// PATENTS file.
			hasOther := false
			for license, licenseFiles := range licenses {
				if license == Apache2 {
					continue
				}
				for _, licenseFile := range licenseFiles {
					if strings.HasPrefix(licenseFile, dir+"/") {
						hasOther = true
					}
				}
			}
			if !hasOther {
				return nil, errors.New("the Apache license contains a patent-grant, but there's a separate PATENTS file; something hokey is going on")
			}
		}
	}

	if !hasLicenseFile && hasNonSPDXSource {
		return nil, errors.New("could not identify a license for all sources (had no global LICENSE file)")
	}

	if len(licenses) == 0 {
		panic(errors.New("should not happen"))
	}
	return bareLicenses, nil
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
		license, licenseOK := SpdxIdentifiers[id]
		if !licenseOK {
			return nil, fmt.Errorf("unknown SPDX identifier %q", id)
		}
		licenses[license] = struct{}{}
	}
	return licenses, nil
}

//nolint:gochecknoglobals // Would be 'const'.
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

	yamlHeader = `The following files were ported to Go from C files of libyaml, and thus
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

`
)

var (
	reYamlV2 = regexp.MustCompile(reWrap(yamlHeader) + `\s*` + reMIT.String())

	reYamlV3 = regexp.MustCompile(`\s*` +
		reQuote(`This project is covered by two different licenses: MIT and Apache.`) + `\s*` +
		`#+ MIT License #+\s*` +
		reWrap(yamlHeader) + `\s*` +
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
	case reMatch(regexp.MustCompile(fmt.Sprintf(`%s\n-+\n+AVL Tree:\n+%s`, reBSD2, reISC)), body):
		// github.com/emirpasic/gods/LICENSE
		licenses[BSD2] = struct{}{}
		licenses[ISC] = struct{}{}
	case reMatch(regexp.MustCompile(``+
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
	case reMatch(regexp.MustCompile(reQuote(rackspaceHeader)+reApacheLicense.String()), body):
		// github.com/gophercloud/gophercloud/LICENSE
		licenses[Apache2] = struct{}{}
	case reMatch(regexp.MustCompile(fmt.Sprintf(`%s=*\s*The lexer and parser[^\n]*\n[^\n]*below\.%s`, reMIT, reMIT)), body):
		// github.com/kevinburke/ssh_config/LICENSE
		licenses[MIT] = struct{}{}
	case reMatch(regexp.MustCompile(`Blackfriday is distributed under the Simplified BSD License:\s*`+reBSD2.String()), regexp.MustCompile(`>\s*`).ReplaceAllLiteral(body, []byte{})):
		// gopkg.in/russross/blackfriday.v2/LICENSE.txt
		licenses[BSD2] = struct{}{}
	case reMatch(reYamlV2, body):
		licenses[MIT] = struct{}{}
	case reMatch(reYamlV3, body):
		licenses[MIT] = struct{}{}
		licenses[Apache2] = struct{}{}
	case reMatch(regexp.MustCompile(reMIT.String()+`\s*`+reBSD3.String()), body):
		// sigs.k8s.io/yaml/LICENSE
		licenses[MIT] = struct{}{}
		licenses[BSD3] = struct{}{}
	case reMatch(regexp.MustCompile(reMIT.String()+`\s*- Based on \S*, which has the following license:\n"""\s*`+reMIT.String()+`\s*"""\s*`), body):
		// github.com/shopspring/decimal/LICENSE
		licenses[MIT] = struct{}{}
	case reMatch(regexp.MustCompile(reBSD3.String()+
		`-+\n+(Files: \S+\n+)+`+reApacheLicense.String()+
		`-+\n+(Files: \S+\n+)+`+reMIT.String()+
		`-+\n+(Files: \S+\n+)+`+reBSD3.String()+
		`-+\n+(Files: \S+\n+)+`+reMIT.String()),
		body):
		// github.com/klauspost/compress/LICENSE
		licenses[Apache2] = struct{}{}
		licenses[BSD3] = struct{}{}
		licenses[MIT] = struct{}{}
	case string(body) == xzPublicDomain:
		// github.com/xi2/xz/LICENSE
		licenses[PublicDomain] = struct{}{}
	default:
		return nil
	}

	return licenses
}
