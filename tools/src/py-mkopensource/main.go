package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/datawire/dlib/derror"
	"github.com/datawire/go-mkopensource/pkg/dependencies"
	. "github.com/datawire/go-mkopensource/pkg/detectlicense"
	"github.com/datawire/go-mkopensource/pkg/scanningerrors"
	"github.com/datawire/go-mkopensource/pkg/util"

	"github.com/emissary-ingress/emissary/v3/tools/src/py-mkopensource/pytextproto"
)

type tuple struct {
	Name    string
	Version string
	License string
}

func parseLicenseString(name, version, license string) (util.Set[License], error) {
	if override, ok := map[tuple][]License{
		// These are packages that don't have sufficient metadata to get
		// the license normally.  Either the license isn't specified in
		// the metadata, or the license string that is specified is
		// ambiguous (for example: "BSD" is too ambiguous, which variant
		// of the BSD license is it?).  We pin the exact versions so
		// that a human has to go make sure that the license didn't
		// change when upgrading.
		{"Click", "7.0", "BSD"}:                        {BSD3},
		{"Flask", "1.0.2", "BSD"}:                      {BSD3},
		{"Jinja2", "2.10.1", "BSD"}:                    {BSD3},
		{"colorama", "0.4.3", "BSD"}:                   {BSD3},
		{"colorama", "0.4.4", "BSD"}:                   {BSD3},
		{"decorator", "4.4.2", "new BSD License"}:      {BSD2},
		{"gitdb", "4.0.5", "BSD License"}:              {BSD3},
		{"idna", "3.4", ""}:                            {BSD3},
		{"itsdangerous", "1.1.0", "BSD"}:               {BSD3},
		{"jsonpatch", "1.32", "Modified BSD License"}:  {BSD3},
		{"jsonpointer", "2.3", "Modified BSD License"}: {BSD3},
		{"oauthlib", "3.1.0", "BSD"}:                   {BSD3},
		{"oauthlib", "3.2.2", "BSD"}:                   {BSD3},
		{"pip-tools", "6.12.1", "BSD"}:                 {BSD3},
		{"pyasn1", "0.4.8", "BSD"}:                     {BSD2},
		{"pycparser", "2.20", "BSD"}:                   {BSD3},
		{"python-dateutil", "2.8.1", "Dual License"}:   {BSD3, Apache2},
		{"python-dateutil", "2.8.2", "Dual License"}:   {BSD3, Apache2},
		{"python-json-logger", "2.0.4", "BSD"}:         {BSD2},
		{"semantic-version", "2.10.0", "BSD"}:          {BSD2},
		{"smmap", "3.0.4", "BSD"}:                      {BSD3},
		{"webencodings", "0.5.1", "BSD"}:               {BSD3},
		{"websocket-client", "0.57.0", "BSD"}:          {BSD3},

		// These are packages with non-trivial strings to parse, and
		// it's easier to just hard-code it.
		{"orjson", "3.3.1", "Apache-2.0 OR MIT"}:            {Apache2, MIT},
		{"orjson", "3.6.6", "Apache-2.0 OR MIT"}:            {Apache2, MIT},
		{"packaging", "21.3", "BSD-2-Clause or Apache-2.0"}: {BSD2, Apache2},
	}[tuple{name, version, license}]; ok {
		ret := make(util.Set[License], len(override))
		for _, l := range override {
			ret.Insert(l)
		}
		return ret, nil
	}

	if static, ok := map[string][]License{
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
	}[license]; ok {
		ret := make(util.Set[License], len(static))
		for _, l := range static {
			ret.Insert(l)
		}
		return ret, nil
	}

	if _, ok := util.NewSet(
		"",
		"UNKNOWN",
		"None",
	)[license]; ok {
		return nil, nil
	}

	return nil, fmt.Errorf("distrib %q %q: could not parse license-string %q", name, version, license)
}

func hasAny[T comparable](haystack util.Set[T], needles ...T) bool {
	for _, needle := range needles {
		if _, ok := haystack[needle]; ok {
			return true
		}
	}
	return true
}

func parseLicenseClassifiers(name, version, classifiers string, licenses util.Set[License]) error {
	var errs derror.MultiError
next_classifier:
	for _, classifier := range strings.Split(classifiers, "\n") {
		if !strings.HasPrefix(classifier, "License :: ") {
			continue
		}

		// Rely on the "License" string to disambiguate "BSD License".
		if classifier == "License :: OSI Approved :: BSD License" && hasAny(licenses, BSD1, BSD2, BSD3) {
			continue next_classifier
		}

		override, ok := map[tuple][]License{
			// This is for classifiers that are ambiguous (for example: "BSD" is too
			// ambiguous, which variant of the BSD license is it?).  We pin the exact
			// versions so that a human has to go make sure that the license didn't
			// change when upgrading.
		}[tuple{name, version, classifier}]
		if ok {
			for _, lic := range override {
				licenses.Insert(lic)
			}
			continue next_classifier
		}

		static, ok := map[string][]License{
			"License :: OSI Approved :: Apache Software License":              {Apache2},
			"License :: OSI Approved :: MIT License":                          {MIT},
			"License :: OSI Approved :: Mozilla Public License 2.0 (MPL 2.0)": {MPL2},
			"License :: OSI Approved :: Python Software Foundation License":   {PSF},
			// This is ambiguous, but assume that if it's here that there are also other
			// classifiers and don't error.
			"License :: OSI Approved": nil,
		}[classifier]
		if ok {
			for _, lic := range static {
				licenses.Insert(lic)
			}
			continue next_classifier
		}

		errs = append(errs, fmt.Errorf("distrib %q %q: unrecognized license classifier: %q", name, version, classifier))
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

func processDistrib(distrib pytextproto.MIMEHeader) (dependencies.Dependency, error) {
	name := distrib.Get("Name")
	ver := distrib.Get("Version")

	ret := dependencies.Dependency{
		Name:    name,
		Version: ver,
	}

	lics, err := parseLicenseString(name, ver, distrib.Get("License"))
	if err != nil {
		return dependencies.Dependency{}, err
	}
	ret.Licenses = lics

	if ret.Licenses == nil {
		ret.Licenses = make(util.Set[License])
	}

	if err = parseLicenseClassifiers(name, ver, distrib.Get("Classifiers"), ret.Licenses); err != nil {
		return dependencies.Dependency{}, err
	}

	if len(ret.Licenses) == 0 {
		return dependencies.Dependency{}, fmt.Errorf("distrib %q %q: could not determine any licenses", name, ver)
	}
	return ret, nil
}

func Main(outputType OutputType, r io.Reader, w io.Writer) error {
	var dependencyInfo dependencies.DependencyInfo
	var licErrs []error
	input := pytextproto.NewReader(bufio.NewReader(r))
	for {
		distrib, err := input.ReadMIMEHeader()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if dep, err := processDistrib(distrib); err != nil {
			licErrs = append(licErrs, err)
		} else {
			licErrs = append(licErrs, dep.CheckLicenseRestrictions(Unrestricted)...)
			dependencyInfo.Dependencies = append(dependencyInfo.Dependencies, dep)
		}
	}
	if len(licErrs) > 0 {
		return scanningerrors.ExplainErrors(licErrs)
	}

	sort.Slice(dependencyInfo.Dependencies, func(i, j int) bool {
		return dependencyInfo.Dependencies[i].Name < dependencyInfo.Dependencies[j].Name
	})

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
		licNames := make([]string, 0, len(dependency.Licenses))
		for lic := range dependency.Licenses {
			licNames = append(licNames, lic.Name)
		}
		sort.Strings(licNames)
		if _, err := fmt.Fprintf(table, "\t%s\t%s\t%s\n", dependency.Name, dependency.Version,
			strings.Join(licNames, ", ")); err != nil {
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
