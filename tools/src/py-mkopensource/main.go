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

	"github.com/datawire/go-mkopensource/pkg/scanningerrors"

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
		{"blinker", "1.6.3", ""}:                       {MIT},
		{"build", "1.0.3", ""}:                         {MIT},
		{"CacheControl", "0.12.6", "UNKNOWN"}:          {Apache2},
		{"CacheControl", "0.12.10", "UNKNOWN"}:         {Apache2},
		{"Click", "7.0", "BSD"}:                        {BSD3},
		{"Flask", "3.0.0", ""}:                         {BSD3},
		{"GitPython", "3.1.11", "UNKNOWN"}:             {BSD3},
		{"Jinja2", "2.10.1", "BSD"}:                    {BSD3},
		{"colorama", "0.4.3", "BSD"}:                   {BSD3},
		{"colorama", "0.4.4", "BSD"}:                   {BSD3},
		{"decorator", "4.4.2", "new BSD License"}:      {BSD2},
		{"gitdb", "4.0.5", "BSD License"}:              {BSD3},
		{"idna", "3.4", ""}:                            {BSD3},
		{"importlib-metadata", "5.1.0", "None"}:        {Apache2},
		{"importlib-resources", "5.4.0", "UNKNOWN"}:    {Apache2},
		{"itsdangerous", "1.1.0", "BSD"}:               {BSD3},
		{"jsonpatch", "1.33", "Modified BSD License"}:  {BSD3},
		{"jsonpointer", "2.4", "Modified BSD License"}: {BSD3},
		{"jsonschema", "3.2.0", "UNKNOWN"}:             {MIT},
		{"lockfile", "0.12.2", "UNKNOWN"}:              {MIT},
		{"oauthlib", "3.1.0", "BSD"}:                   {BSD3},
		{"oauthlib", "3.2.2", "BSD"}:                   {BSD3},
		{"pep517", "0.13.0", ""}:                       {MIT},
		{"pip-tools", "7.3.0", "BSD"}:                  {BSD3},
		{"ptyprocess", "0.6.0", "UNKNOWN"}:             {ISC},
		{"pyasn1", "0.5.0", "BSD"}:                     {BSD2},
		{"pyasn1-modules", "0.3.0", "BSD"}:             {BSD2},
		{"pycparser", "2.20", "BSD"}:                   {BSD3},
		{"pyparsing", "3.0.9", ""}:                     {MIT},
		{"pyproject_hooks", "1.0.0", ""}:               {MIT},
		{"python-dateutil", "2.8.1", "Dual License"}:   {BSD3, Apache2},
		{"python-dateutil", "2.8.2", "Dual License"}:   {BSD3, Apache2},
		{"python-json-logger", "2.0.7", "BSD"}:         {BSD2},
		{"semantic-version", "2.10.0", "BSD"}:          {BSD2},
		{"smmap", "3.0.4", "BSD"}:                      {BSD3},
		{"tomli", "2.0.1", ""}:                         {MIT},
		{"typing_extensions", "4.8.0", ""}:             {PSF},
		{"webencodings", "0.5.1", "BSD"}:               {BSD3},
		{"websocket-client", "0.57.0", "BSD"}:          {BSD3},
		{"websocket-client", "1.2.3", "Apache-2.0"}:    {Apache2},
		{"Werkzeug", "3.0.0", ""}:                      {BSD3},
		{"zipp", "3.11.0", "None"}:                     {MIT},

		// These are packages with non-trivial strings to parse, and
		// it's easier to just hard-code it.
		{"orjson", "3.9.9", "Apache-2.0 OR MIT"}: {Apache2, MIT},
		{"packaging", "23.1", ""}:                {BSD2, Apache2},
		{"packaging", "23.2", ""}:                {BSD2, Apache2},
	}[tuple{name, version, license}]
	if ok {
		ret := make(map[License]struct{}, len(override))
		for _, l := range override {
			ret[l] = struct{}{}
		}
		return ret
	}

	static, ok := map[string][]License{
		"AGPLv3+": {AGPL3OrLater},

		"ASL 2":                       {Apache2},
		"Apache":                      {Apache2},
		"Apache 2":                    {Apache2},
		"Apache 2.0":                  {Apache2},
		"Apache-2.0":                  {Apache2},
		"Apache-2.0 license":          {Apache2},
		"Apache License":              {Apache2},
		"Apache License 2.0":          {Apache2},
		"Apache License Version 2.0":  {Apache2},
		"Apache License, Version 2.0": {Apache2},
		"Apache Software License":     {Apache2},
		"Apache Software License 2.0": {Apache2},

		"BSD-2-Clause": {BSD2},

		"3-Clause BSD License": {BSD3},
		"BSD-3-Clause":         {BSD3},
		"BSD 3 Clause":         {BSD3},

		"GPLv2": {GPL2Only},

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

	dependencyInfo, err := getDependencies(distribNames, distribs)
	if err != nil {
		return err
	}

	switch outputType {
	case jsonOutputType:
		if err := jsonOutput(w, dependencyInfo); err != nil {
			return err
		}
	default:
		if err := markdownOutput(w, dependencyInfo); err != nil {
			return err
		}
	}

	return nil
}

func jsonOutput(w io.Writer, dependencyInfo dependencies.DependencyInfo) error {
	jsonString, marshalErr := json.Marshal(dependencyInfo)
	if marshalErr != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Could not generate JSON output: %v\n", marshalErr)
		os.Exit(int(MarshallJsonError))
	}

	if _, err := w.Write(jsonString); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Could not write JSON output: %v\n", err)
		os.Exit(int(WriteError))
	}

	_, _ = fmt.Fprintf(w, "\n")

	return nil
}

func markdownOutput(w io.Writer, dependencyInfo dependencies.DependencyInfo) error {
	table := tabwriter.NewWriter(w, 0, 8, 2, ' ', 0)
	_, _ = io.WriteString(table, "  \tName\tVersion\tLicense(s)\n")
	_, _ = io.WriteString(table, "  \t----\t-------\t----------\n")
	for _, dependency := range dependencyInfo.Dependencies {
		if _, err := fmt.Fprintf(table, "\t%s\t%s\t%s\n", dependency.Name, dependency.Version,
			strings.Join(dependency.Licenses, ", ")); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Could not write Markdown output: %v\n", err)
			os.Exit(int(WriteError))
		}
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
	dependencyInfo := dependencies.NewDependencyInfo()

	var errs derror.MultiError
	for _, distribName := range distribNames {
		distrib := distribs[distribName]
		distribVersion := distrib.Get("Version")

		dependencyDetails := dependencies.Dependency{
			Name:    distribName,
			Version: distribVersion,
		}
		licenses := parseLicenses(distribName, distribVersion, distrib.Get("License"))
		if licenses == nil {
			errs = append(errs, fmt.Errorf("distrib %q %q: Could not parse license-string %q", distribName, distribVersion, distrib.Get("License")))
			continue
		}

		licenseList := make([]string, 0, len(licenses))
		for license := range licenses {
			licenseList = append(licenseList, license.Name)
			if err := dependencies.CheckLicenseRestrictions(dependencyDetails, license.Name, Unrestricted); err != nil {
				errs = append(errs, err)
			}
		}
		sort.Strings(licenseList)

		dependencyDetails.Licenses = licenseList
		dependencyInfo.Dependencies = append(dependencyInfo.Dependencies, dependencyDetails)
	}

	if len(errs) > 0 {
		return dependencyInfo, scanningerrors.ExplainErrors(errs)
	}

	if err := dependencyInfo.UpdateLicenseList(); err != nil {
		return dependencyInfo, fmt.Errorf("Could not generate list of license URLs: %v\n", err)
	}

	return dependencyInfo, nil
}

func main() {
	cliArgs, err := parseArgs()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s: %v\nTry '%s --help' for more information.\n", os.Args[0], err, os.Args[0])
		os.Exit(int(InvalidArgumentsError))
	}

	if err := Main(cliArgs.outputType, os.Stdin, os.Stdout); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s: fatal: %v\n", os.Args[0], err)
		os.Exit(int(DependencyGenerationError))
	}
}
