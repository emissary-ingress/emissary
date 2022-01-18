package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/textproto"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/pkg/errors"

	"github.com/datawire/dlib/derror"
	"github.com/datawire/go-mkopensource/pkg/dependencies"
	. "github.com/datawire/go-mkopensource/pkg/detectlicense"
)

type tuple struct {
	Name    string
	Version string
	License string
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
		{"importlib-resources", "5.4.0", "UNKNOWN"}:    {Apache2},
		{"itsdangerous", "1.1.0", "BSD"}:               {BSD3},
		{"jsonpatch", "1.32", "Modified BSD License"}:  {BSD3},
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
		{"python-json-logger", "2.0.2", "BSD"}:         {BSD2},
		{"semantic-version", "2.8.5", "BSD"}:           {BSD2},
		{"smmap", "3.0.4", "BSD"}:                      {BSD3},
		{"webencodings", "0.5.1", "BSD"}:               {BSD3},
		{"websocket-client", "0.57.0", "BSD"}:          {BSD3},
		{"zipp", "3.6.0", "UNKNOWN"}:                   {MIT},

		// These are packages with non-trivial strings to parse, and
		// it's easier to just hard-code it.
		{"docutils", "0.17.1", "public domain, Python, 2-Clause BSD, GPL 3 (see COPYING.txt)"}: {PublicDomain, PSF, BSD2, GPL3},
		{"orjson", "3.3.1", "Apache-2.0 OR MIT"}:                                               {Apache2, MIT},
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
		"Apache-2.0 license":          {Apache2},
		"Apache License":              {Apache2},
		"Apache License 2.0":          {Apache2},
		"Apache License Version 2.0":  {Apache2},
		"Apache License, Version 2.0": {Apache2},
		"Apache Software License":     {Apache2},
		"Apache Software License 2.0": {Apache2},

		"3-Clause BSD License": {BSD3},
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

func Main(outputType OutputType, r io.Reader, w io.Writer) error {
	distribs := make(map[string]textproto.MIMEHeader)

	input := textproto.NewReader(bufio.NewReader(r))
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

	switch outputType {
	case jsonOutputType:
		if err := jsonOutput(w, distribNames, distribs); err != nil {
			return err
		}
	default:
		if err := markdownOutput(w, distribNames, distribs); err != nil {
			return err
		}
	}

	return nil
}

func jsonOutput(w io.Writer, distribNames []string, distribs map[string]textproto.MIMEHeader) error {
	dependencyInfo, err := getDependencies(distribNames, distribs)
	if err != nil {
		return err
	}

	jsonString, marshalErr := json.Marshal(dependencyInfo)
	if marshalErr != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Could not generate JSON output: %v\n", err)
		os.Exit(int(MarshallJsonError))
	}

	if _, err := w.Write(jsonString); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Could not write JSON output: %v\n", err)
		os.Exit(int(WriteError))
	}

	_, _ = fmt.Fprintf(w, "\n")

	return nil
}

func markdownOutput(w io.Writer, distribNames []string, distribs map[string]textproto.MIMEHeader) error {
	table := tabwriter.NewWriter(w, 0, 8, 2, ' ', 0)
	_, _ = io.WriteString(table, "  \tName\tVersion\tLicense(s)\n")
	_, _ = io.WriteString(table, "  \t----\t-------\t----------\n")
	var errs derror.MultiError
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

		if _, err := fmt.Fprintf(table, "\t%s\t%s\t%s\n", distribName, distribVersion, distribLicense); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Could not write Markdown output: %v\n", err)
			os.Exit(int(WriteError))
		}
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

	if _, err := fmt.Fprintf(w, "The Emissary-ingress Python code makes use of the following Free and Open Source\nlibraries:\n\n"); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Could not write Markdown output: %v\n", err)
		os.Exit(int(WriteError))
	}

	if err := table.Flush(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Could not write Markdown output: %v\n", err)
		os.Exit(int(WriteError))
	}

	return nil
}

func getDependencies(distribNames []string, distribs map[string]textproto.MIMEHeader) (dependencies.DependencyInfo, error) {
	allLicenses := map[License]struct{}{}
	jsonOutput := dependencies.NewDependencyInfo()

	var errs derror.MultiError
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
			allLicenses[license] = struct{}{}
		}
		sort.Strings(licenseList)

		dependencyDetails := dependencies.Dependency{
			Name:     distribName,
			Version:  distribVersion,
			Licenses: licenseList,
		}
		jsonOutput.Dependencies = append(jsonOutput.Dependencies, dependencyDetails)
	}

	if len(errs) > 0 {
		err := errs
		return jsonOutput, errors.Errorf(`%v
    This probably means that you added or upgraded a dependency, and the
    automated opensource-license-checker can't confidently detect what
    the license is.  (This is a good thing, because it is reminding you
    to check the license of libraries before using them.)

    You need to update the "github.com/datawire/ambassador/v2/cmd/py-mkopensource/main.go"
    file to correctly detect the license.`,
			err)
	}

	for license := range allLicenses {
		jsonOutput.Licenses[license.Name] = license.Url
	}

	return jsonOutput, nil
}

func main() {
	cliArgs, err := parseArgs()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s: %v\nTry '%s --help' for more information.\n", os.Args[0], err, os.Args[0])
		os.Exit(int(InvalidArgumentsError))
	}

	if err := Main(cliArgs.outputType, os.Stdin, os.Stdout); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(int(DependencyGenerationError))
	}
}
