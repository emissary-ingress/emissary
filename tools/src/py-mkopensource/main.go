package main

import (
	"bufio"
	"fmt"
	"io"
	"net/textproto"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/pkg/errors"

	. "github.com/datawire/ambassador/v2/pkg/mkopensource/detectlicense"
)

type tuple struct {
	Name    string
	Version string
	License string
}

var (
	PSF         = License{Name: "Python Software Foundation license"}
	Unicode2015 = License{Name: "Unicode License Agreement for Data Files and Software (2015)"}
	LGPL21      = License{Name: "GNU Lesser General Public License Version 2.1", WeakCopyleft: true}
	GPL3        = License{Name: "GNU General Public License Version 3", StrongCopyleft: true}
)

type errset []error

func (e errset) Error() string {
	strs := make([]string, len(e))
	for i, err := range e {
		strs[i] = err.Error()
	}
	return strings.Join(strs, "\n")
}

func parseLicenses(name, version, license string) map[License]struct{} {
	override, ok := map[tuple][]License{
		// These are packages that don't have sufficient metadata to get
		// the license normally.  Either the license isn't specified in
		// the metadata, or the license string that is specified is
		// ambiguous (for example: "BSD" is too ambiguous, which variant
		// of the BSD license is it?).  We pin the exact versions so
		// that a human has to go make sure that the license didn't
		// change when upgrading.
		{"CacheControl", "0.12.6", "UNKNOWN"}:          {Apache2},
		{"Click", "7.0", "BSD"}:                        {BSD3},
		{"Flask", "1.0.2", "BSD"}:                      {BSD3},
		{"GitPython", "3.1.11", "UNKNOWN"}:             {BSD3},
		{"Jinja2", "2.10.1", "BSD"}:                    {BSD3},
		{"chardet", "3.0.4", "LGPL"}:                   {LGPL21},
		{"colorama", "0.4.3", "BSD"}:                   {BSD3},
		{"decorator", "4.4.2", "new BSD License"}:      {BSD2},
		{"gitdb", "4.0.5", "BSD License"}:              {BSD3},
		{"idna", "2.7", "BSD-like"}:                    {BSD3, PSF, Unicode2015},
		{"idna", "2.8", "BSD-like"}:                    {BSD3, PSF, Unicode2015},
		{"itsdangerous", "1.1.0", "BSD"}:               {BSD3},
		{"jsonpatch", "1.30", "Modified BSD License"}:  {BSD3},
		{"jsonpointer", "2.0", "Modified BSD License"}: {BSD3},
		{"jsonschema", "3.2.0", "UNKNOWN"}:             {MIT},
		{"lockfile", "0.12.2", "UNKNOWN"}:              {MIT},
		{"oauthlib", "3.1.0", "BSD"}:                   {BSD3},
		{"pep517", "0.8.2", "UNKNOWN"}:                 {MIT},
		{"pip-tools", "5.3.1", "BSD"}:                  {BSD3},
		{"ptyprocess", "0.6.0", "UNKNOWN"}:             {ISC},
		{"pyasn1", "0.4.8", "BSD"}:                     {BSD2},
		{"pycparser", "2.20", "BSD"}:                   {BSD3},
		{"python-dateutil", "2.8.1", "Dual License"}:   {BSD3, Apache2},
		{"semantic-version", "2.8.5", "BSD"}:           {BSD2},
		{"smmap", "3.0.4", "BSD"}:                      {BSD3},
		{"webencodings", "0.5.1", "BSD"}:               {BSD3},
		{"websocket-client", "0.57.0", "BSD"}:          {BSD3},
		{"zipp", "3.4.0", "UNKNOWN"}:                   {MIT},

		// These are packages with non-trivial strings to parse, and
		// it's easier to just hard-code it.
		{"docutils", "0.15.2", "public domain, Python, 2-Clause BSD, GPL 3 (see COPYING.txt)"}: {PublicDomain, PSF, BSD2, GPL3},
		{"packaging", "20.4", "BSD-2-Clause or Apache-2.0"}:                                    {BSD2, Apache2},
	}[tuple{name, version, license}]
	if ok {
		ret := make(map[License]struct{}, len(override))
		for _, l := range override {
			ret[l] = struct{}{}
		}
		return ret
	}

	static, ok := map[string][]License{
		"ASL 2":                       {Apache2},
		"Apache":                      {Apache2},
		"Apache 2":                    {Apache2},
		"Apache 2.0":                  {Apache2},
		"Apache License":              {Apache2},
		"Apache License 2.0":          {Apache2},
		"Apache License Version 2.0":  {Apache2},
		"Apache License, Version 2.0": {Apache2},
		"Apache Software License":     {Apache2},
		"Apache Software License 2.0": {Apache2},
		"Apache-2.0 OR MIT":           {Apache2},

		"3-Clause BSD License": {BSD3},
		"BSD":                  {BSD2},
		"BSD-2-Clause":         {BSD2},
		"BSD-3-Clause":         {BSD3},

		"ISC license": {ISC},
		"ISC":         {ISC},

		"MIT License": {MIT},
		"MIT license": {MIT},
		"MIT":         {MIT},
		"MIT-LICENSE": {MIT},

		"Mozilla Public License 2.0 (MPL 2.0)": {MPL2},
		"MPL-2.0":                              {MPL2},

		"PSF License":    {PSF},
		"PSF":            {PSF},
		"Python license": {PSF},
	}[license]
	if ok {
		ret := make(map[License]struct{}, len(static))
		for _, l := range static {
			ret[l] = struct{}{}
		}
		return ret
	}

	return nil
}

func Main() error {
	distribs := make(map[string]textproto.MIMEHeader)

	input := textproto.NewReader(bufio.NewReader(os.Stdin))
	for {
		distrib, err := input.ReadMIMEHeader()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		distribs[distrib.Get("Name")] = distrib
	}

	distribNames := make([]string, 0, len(distribs))
	for distribName := range distribs {
		distribNames = append(distribNames, distribName)
	}
	sort.Strings(distribNames)

	table := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	io.WriteString(table, "  \tName\tVersion\tLicense(s)\n")
	io.WriteString(table, "  \t----\t-------\t----------\n")
	var errs errset
	for _, distribName := range distribNames {
		distrib := distribs[distribName]
		distribVersion := distrib.Get("Version")

		licenses := parseLicenses(distribName, distribVersion, distrib.Get("License"))
		if licenses == nil {
			errs = append(errs, fmt.Errorf("distrib %q %q: Could not parse license-string %q", distribName, distribVersion, distrib.Get("License")))
			continue
		}
		licenseList := make([]string, 0, len(licenses))
		for license := range licenses {
			licenseList = append(licenseList, license.Name)
		}
		sort.Strings(licenseList)
		distribLicense := strings.Join(licenseList, ", ")

		fmt.Fprintf(table, "\t%s\t%s\t%s\n", distribName, distribVersion, distribLicense)
	}
	if len(errs) > 0 {
		err := errs
		return errors.Errorf(`%v
    This probably means that you added or upgraded a dependency, and the
    automated opensource-license-checker can't confidently detect what
    the license is.  (This is a good thing, because it is reminding you
    to check the license of libraries before using them.)

    You need to update the "github.com/datawire/ambassador/v2/cmd/py-mkopensource/main.go"
    file to correctly detect the license.`,
			err)
	}
	fmt.Printf("The Ambassador Python code makes use of the following Free and Open Source\nlibraries:\n\n")
	table.Flush()

	return nil
}

func main() {
	if err := Main(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
