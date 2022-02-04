package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/pflag"

	"github.com/datawire/go-mkopensource/pkg/detectlicense"
	"github.com/datawire/go-mkopensource/pkg/golist"
)

type CLIArgs struct {
	OutputFormat    string
	OutputName      string
	OutputType      string
	ApplicationType string

	GoTarFilename string
	Package       string
}

const (
	// Type of output to generate
	markdownOutputType = "markdown"
	jsonOutputType     = "json"

	// Validations to do on the licenses.
	// The only validation for "internal" is to check chat forbidden licenses are not used
	internalApplication = "internal"
	// "external" applications have additional license requirements as documented in
	//https://www.notion.so/datawire/License-Management-5194ca50c9684ff4b301143806c92157
	externalApplication = "external"
)

func parseArgs() (*CLIArgs, error) {
	args := &CLIArgs{}
	argparser := pflag.NewFlagSet(os.Args[0], pflag.ContinueOnError)
	help := false
	argparser.BoolVarP(&help, "help", "h", false, "Show this message")
	argparser.StringVar(&args.OutputFormat, "output-format", "", "Output format ('tar' or 'txt')")
	argparser.StringVar(&args.OutputName, "output-name", "", "Name of the root directory in the --output-format=tar tarball")
	argparser.StringVar(&args.OutputType, "output-type", markdownOutputType,
		fmt.Sprintf("Format used when printing dependency information. One of: %s, %s", markdownOutputType, jsonOutputType))
	argparser.StringVar(&args.GoTarFilename, "gotar", "", "Tarball of the Go stdlib source code")
	argparser.StringVar(&args.Package, "package", "", "The package(s) to report library usage for")
	argparser.StringVar(&args.ApplicationType, "application-type", externalApplication,
		fmt.Sprintf("Where will the application run. One of: %s, %s\n"+
			"Internal applications are run on Ambassador servers.\n"+
			"External applications run on client-controlled infrastructure", internalApplication, externalApplication))

	if err := argparser.Parse(os.Args[1:]); err != nil {
		return nil, err
	}
	if help {
		fmt.Printf("Usage: %v OPTIONS\n", os.Args[0])
		fmt.Println("Build a .opensource.tar.gz tarball for open source license compliance")
		fmt.Println()
		fmt.Println("OPTIONS:")
		argparser.PrintDefaults()
		return nil, pflag.ErrHelp
	}
	if argparser.NArg() != 0 {
		return nil, fmt.Errorf("expected 0 arguments, got %d: %q", argparser.NArg(), argparser.Args())
	}

	if args.OutputType != markdownOutputType && args.OutputType != jsonOutputType {
		return nil, fmt.Errorf("--output-type must be one of '%s', '%s'", markdownOutputType, jsonOutputType)
	}

	if args.ApplicationType != internalApplication && args.ApplicationType != externalApplication {
		return nil, fmt.Errorf("--application-type must be one of '%s', '%s'", internalApplication, externalApplication)
	}

	switch args.OutputFormat {
	case "txt":
		if args.OutputName != "" {
			return nil, errors.New("--output-name is only valid for --output-mode=tar")
		}
	case "tar":
		if args.OutputName == "" {
			return nil, errors.New("--output-name is required for --output-mode=tar")
		}
		if args.OutputType != markdownOutputType {
			return nil, fmt.Errorf("--output-type should be set to '%s' for --output-mode=tar", markdownOutputType)
		}

	default:
		return nil, errors.New("--output-format must be one of 'tar' or 'txt'")
	}

	if !strings.HasPrefix(filepath.Base(args.GoTarFilename), "go1.") || !strings.HasSuffix(args.GoTarFilename, ".tar.gz") {
		return nil, fmt.Errorf("--gotar (%q) doesn't look like a go1.*.tar.gz file", args.GoTarFilename)
	}
	if args.Package == "" {
		return nil, fmt.Errorf("--package (%q) must be non-empty", args.Package)
	}

	return args, nil
}

func main() {
	args, err := parseArgs()
	if err != nil {
		if err == pflag.ErrHelp {
			os.Exit(int(NoError))
		}
		fmt.Fprintf(os.Stderr, "%s: %v\nTry '%s --help' for more information.\n", os.Args[0], err, os.Args[0])
		os.Exit(int(InvalidArgumentsError))
	}
	if err := Main(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: fatal: %v\n", os.Args[0], err)
		os.Exit(int(DependencyGenerationError))
	}
}

func loadGoTar(goTarFilename string) (version string, license []byte, err error) {
	goTarFile, err := os.Open(goTarFilename)
	if err != nil {
		return "", nil, err
	}
	defer goTarFile.Close()
	goTarUncompressed, err := gzip.NewReader(goTarFile)
	if err != nil {
		return "", nil, err
	}
	defer goTarUncompressed.Close()
	goTar := tar.NewReader(goTarUncompressed)
	for {
		header, err := goTar.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", nil, err
		}
		switch header.Name {
		case "go/VERSION":
			fc, err := io.ReadAll(goTar)
			if err != nil {
				return "", nil, err
			}
			version = "v" + strings.TrimPrefix(strings.TrimSpace(string(fc)), "go")
		case "go/LICENSE":
			fc, err := io.ReadAll(goTar)
			if err != nil {
				return "", nil, err
			}
			license = fc
		}
		if version != "" && license != nil {
			break
		}
	}
	if version == "" || license == nil {
		return "", nil, fmt.Errorf("file %q did not contain %q or %q", goTarFilename, "go/VERSION", "go/LICENSE")
	}
	return version, license, nil
}

func isAmbassadorProprietary(licenses map[detectlicense.License]struct{}) bool {
	_, ok := licenses[detectlicense.AmbassadorProprietary]
	return ok
}

func licenseIsWeakCopyleft(licenses map[detectlicense.License]struct{}) bool {
	for license := range licenses {
		if license.WeakCopyleft {
			return true
		}
	}
	return false
}

func licenseIsStrongCopyleft(licenses map[detectlicense.License]struct{}) bool {
	for license := range licenses {
		if license.StrongCopyleft {
			return true
		}
	}
	return false
}

func licenseString(licenseSet map[detectlicense.License]struct{}) string {
	licenseList := make([]string, 0, len(licenseSet))
	for license := range licenseSet {
		licenseList = append(licenseList, license.Name)
	}
	sort.Strings(licenseList)
	return strings.Join(licenseList, ", ")
}

func Main(args *CLIArgs) error {
	// Let's do the expensive stuff (stuff that isn't entirely
	// in-memory) up-front.

	// `tar xf go{version}.src.tar.gz`
	goVersion, goLicense, err := loadGoTar(args.GoTarFilename)
	if err != nil {
		return err
	}

	// `go list`
	var mainMods map[string]struct{}
	var listPkgs []golist.Package
	if args.Package == "mod" {
		// `go list`
		listPkgs, err = VendorList()
		if err != nil {
			return err
		}
		listPkgs = append(listPkgs, golist.Package{}) // stdlib

		// `go list -m`
		cmd := exec.Command("go", "list", "-m")
		cmd.Stderr = os.Stderr
		modname, err := cmd.Output()
		if err != nil {
			return err
		}
		mainMods = make(map[string]struct{}, 1)
		mainMods[strings.TrimSpace(string(modname))] = struct{}{}
	} else {
		// `go list`
		listPkgs, err = golist.GoListPackages([]string{"-deps"}, []string{args.Package})
		if err != nil {
			return err
		}
		// `go list -m` (fast: in-memory)
		mainMods = make(map[string]struct{})
		for _, pkg := range listPkgs {
			if !pkg.DepOnly && pkg.Module != nil {
				mainMods[pkg.Module.Path] = struct{}{}
			}
		}
	}

	// `go mod vendor`
	fs := newFSCache()
	pkgFiles := make(map[string]map[string][]byte)
	for _, pkg := range listPkgs {
		vendor := make(map[string][]byte)
		if pkg.Module == nil {
			// standard library
			vendor["std/LICENSE"] = goLicense
		} else {
			// module
			if _, isMainMod := mainMods[pkg.Module.Path]; isMainMod {
				continue
			}
			if args.Package == "mod" {
				if err := fs.collectVendoredPkg(vendor, pkg); err != nil {
					return err
				}
			} else {
				if err := fs.collectPkg(vendor, pkg); err != nil {
					return err
				}
			}
		}
		pkgFiles[pkg.ImportPath] = vendor
	}

	// From this point on, everything should be entirely in-memory
	// (besides writing the output file, I guess).

	// Figure out the license(s) that apply to each package.  We
	// sort the packages first so that if there's an error, which
	// error the user sees is deterministic.
	pkgNames := make([]string, 0, len(pkgFiles))
	for pkgName := range pkgFiles {
		pkgNames = append(pkgNames, pkgName)
	}
	sort.Strings(pkgNames)
	pkgLicenses := make(map[string]map[detectlicense.License]struct{})
	licErrs := []error(nil)
	for _, pkgName := range pkgNames {
		pkgLicenses[pkgName], err = detectlicense.DetectLicenses(pkgName, pkgFiles[pkgName])
		if err == nil && licenseIsStrongCopyleft(pkgLicenses[pkgName]) {
			err = fmt.Errorf("has an unacceptable license for use by Ambassador Labs (%s)",
				licenseString(pkgLicenses[pkgName]))
		}
		if err != nil {
			err = fmt.Errorf(`package %q: %w`, pkgName, err)
			licErrs = append(licErrs, err)
		}
	}
	if len(licErrs) > 0 {
		return ExplainErrors(licErrs)
	}

	// Group packages by module & collect module info
	modPkgs := make(map[string][]string)
	modInfos := make(map[string]*golist.Module)
	modLicenses := make(map[string]map[detectlicense.License]struct{})
	modNames := make([]string, 0, len(modPkgs))
	for _, pkg := range listPkgs {
		key := "<nil>"
		if pkg.Module != nil {
			key = pkg.Module.Path
		}
		if _, isMainMod := mainMods[key]; isMainMod {
			continue
		}
		modPkgs[key] = append(modPkgs[key], pkg.ImportPath)
		if _, done := modInfos[key]; !done {
			modInfos[key] = pkg.Module
			modLicenses[key] = make(map[detectlicense.License]struct{})
			modNames = append(modNames, key)
		}
		for license := range pkgLicenses[pkg.ImportPath] {
			modLicenses[key][license] = struct{}{}
		}
	}
	sort.Strings(modNames)

	// Figure out how to pronounce "X" in "X incorporates Free and
	// Open Source software".
	var mainCmdPkgs []string
	var mainLibPkgs []string
	for _, pkg := range listPkgs {
		if pkg.Module == nil {
			continue
		}
		if _, isMainMod := mainMods[pkg.Module.Path]; !isMainMod {
			continue
		}
		if pkg.DepOnly {
			continue
		}
		if pkg.Name == "main" {
			mainCmdPkgs = append(mainCmdPkgs, pkg.ImportPath)
		} else {
			mainLibPkgs = append(mainLibPkgs, pkg.ImportPath)
		}
	}
	sort.Strings(mainCmdPkgs)
	sort.Strings(mainLibPkgs)

	// Generate the readme file.
	licenseUsage := getAllowedLicenseUsage(args.ApplicationType)

	switch args.OutputFormat {
	case "txt":
		readme, generationErr := generateOutput(args.Package, args.OutputFormat, args.OutputType, licenseUsage, mainMods, mainLibPkgs, mainCmdPkgs, modNames, modLicenses, modInfos, goVersion)
		if generationErr != nil {
			return generationErr
		}

		if _, err := readme.WriteTo(os.Stdout); err != nil {
			return err
		}
	case "tar":
		// Build a listing of all files to go in to the tarball
		readme, generationErr := generateOutput(args.Package, args.OutputFormat, markdownOutputType, licenseUsage, mainMods, mainLibPkgs, mainCmdPkgs, modNames, modLicenses, modInfos, goVersion)
		if generationErr != nil {
			return generationErr
		}

		tarFiles := make(map[string][]byte)
		tarFiles["OPENSOURCE.md"] = readme.Bytes()
		for pkgName := range pkgFiles {
			ambassadorProprietary := isAmbassadorProprietary(pkgLicenses[pkgName])
			switch {
			case ambassadorProprietary:
				// don't include anything
			case licenseIsWeakCopyleft(pkgLicenses[pkgName]):
				// include everything
				for filename, fileBody := range pkgFiles[pkgName] {
					tarFiles[filename] = fileBody
				}
			default:
				// just include metadata
				for filename, fileBody := range pkgFiles[pkgName] {
					if matchMetadata(filename) {
						tarFiles[filename] = fileBody
					}
				}
			}
		}

		// Write output
		outputFile := os.Stdout
		defer outputFile.Close()
		outputCompressed := gzip.NewWriter(outputFile)
		defer outputCompressed.Close()
		outputTar := tar.NewWriter(outputCompressed)
		defer outputTar.Close()

		filenames := make([]string, 0, len(tarFiles))
		for filename := range tarFiles {
			filenames = append(filenames, filename)
		}
		sort.Strings(filenames)
		for _, filename := range filenames {
			body := tarFiles[filename]
			err := outputTar.WriteHeader(&tar.Header{
				Typeflag: tar.TypeReg,
				Name:     args.OutputName + "/" + filename,
				Size:     int64(len(body)),
				Mode:     0644,
			})
			if err != nil {
				return err
			}
			if _, err := outputTar.Write(body); err != nil {
				return err
			}
		}
	}

	return nil
}

func getAllowedLicenseUsage(applicationType string) detectlicense.AllowedLicenseUse {
	var licenseUsage detectlicense.AllowedLicenseUse
	switch applicationType {
	case internalApplication:
		licenseUsage = detectlicense.OnAmbassadorServers
	default:
		licenseUsage = detectlicense.Unrestricted
	}
	return licenseUsage
}

func generateOutput(packages string, outputFormat string, outputType string, licenseUsage detectlicense.AllowedLicenseUse,
	mainMods map[string]struct{}, mainLibPkgs []string, mainCmdPkgs []string, modNames []string,
	modLicenses map[string]map[detectlicense.License]struct{}, modInfos map[string]*golist.Module,
	goVersion string) (*bytes.Buffer, error) {
	output := new(bytes.Buffer)

	switch outputType {
	case jsonOutputType:
		err := jsonOutput(output, modNames, modLicenses, modInfos, goVersion, licenseUsage)
		if err != nil {
			return nil, err
		}
	default:
		markdownHeader(packages, mainMods, output, mainLibPkgs, mainCmdPkgs)
		output.WriteString("\n")
		err := markdownOutput(output, modNames, modLicenses, modInfos, goVersion, licenseUsage)
		if err != nil {
			return nil, err
		}
	}

	if outputFormat == "tar" {
		output.WriteString("\n")
		output.WriteString(wordwrap(0, 75, "The appropriate license notices and source code are in correspondingly named directories.") + "\n")
	}
	return output, nil
}

func markdownHeader(packages string, mainMods map[string]struct{}, readme *bytes.Buffer, mainLibPkgs []string, mainCmdPkgs []string) {
	if packages == "mod" {
		modnames := make([]string, 0, len(mainMods))
		for modname := range mainMods {
			modnames = append(modnames, modname)
		}
		if len(mainMods) == 1 {
			readme.WriteString(wordwrap(0, 75, fmt.Sprintf("The Go module %q incorporates the following Free and Open Source software:", modnames[0])) + "\n")
		} else {
			sort.Strings(modnames)
			readme.WriteString(wordwrap(0, 75, fmt.Sprintf("The Go modules %q incorporate the following Free and Open Source software:", modnames)) + "\n")
		}
		return
	}

	if len(mainLibPkgs) == 0 {
		if len(mainCmdPkgs) == 1 {
			readme.WriteString(wordwrap(0, 75, fmt.Sprintf("The program %q incorporates the following Free and Open Source software:", path.Base(mainCmdPkgs[0]))) + "\n")
		} else {
			readme.WriteString(wordwrap(0, 75, fmt.Sprintf("The programs %q incorporate the following Free and Open Source software:", packages)) + "\n")
		}
		return
	}

	if len(mainLibPkgs) == 1 {
		readme.WriteString(wordwrap(0, 75, fmt.Sprintf("The Go package %q incorporates the following Free and Open Source software:", mainLibPkgs[0])) + "\n")
	} else {
		readme.WriteString(wordwrap(0, 75, fmt.Sprintf("The Go packages %q incorporate the following Free and Open Source software:", packages)) + "\n")
	}
}

func markdownOutput(readme *bytes.Buffer, modNames []string, modLicenses map[string]map[detectlicense.License]struct{},
	modInfos map[string]*golist.Module, goVersion string, usage detectlicense.AllowedLicenseUse) error {
	dependencyList, generationErr := GenerateDependencyList(modNames, modLicenses, modInfos, goVersion, usage)
	if generationErr != nil {
		return generationErr
	}

	table := tabwriter.NewWriter(readme, 0, 8, 2, ' ', 0)
	io.WriteString(table, "  \tName\tVersion\tLicense(s)\n")
	io.WriteString(table, "  \t----\t-------\t----------\n")

	for _, dependency := range dependencyList.Dependencies {
		depLicenses := strings.Join(dependency.Licenses, ", ")
		if depLicenses == "" {
			panic(fmt.Errorf("this should not happen: empty license string for %q", dependency.Name))
		}

		fmt.Fprintf(table, "\t%s\t%s\t%s\n", dependency.Name, dependency.Version, depLicenses)
	}
	table.Flush()
	return nil
}

func jsonOutput(readme *bytes.Buffer, modNames []string, modLicenses map[string]map[detectlicense.License]struct{},
	modInfos map[string]*golist.Module, goVersion string, usage detectlicense.AllowedLicenseUse) error {
	dependencyList, generationErr := GenerateDependencyList(modNames, modLicenses, modInfos, goVersion, usage)
	if generationErr != nil {
		return generationErr
	}

	jsonString, marshallErr := json.Marshal(dependencyList)
	if marshallErr != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Could not generate JSON output: %v\n", marshallErr)
		os.Exit(int(MarshallJsonError))
	}

	readme.Write(jsonString)
	return nil
}
