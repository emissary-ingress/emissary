package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"

	"github.com/datawire/ambassador/v2/pkg/mkopensource/detectlicense"
	"github.com/datawire/ambassador/v2/pkg/mkopensource/golist"
)

type CLIArgs struct {
	OutputFormat string
	OutputName   string

	GoTarFilename string
	Package       string
}

func parseArgs() (*CLIArgs, error) {
	args := &CLIArgs{}
	argparser := pflag.NewFlagSet(os.Args[0], pflag.ContinueOnError)
	help := false
	argparser.BoolVarP(&help, "help", "h", false, "Show this message")
	argparser.StringVar(&args.OutputFormat, "output-format", "", "Output format ('tar' or 'txt')")
	argparser.StringVar(&args.OutputName, "output-name", "", "Name of the root directory in the --output-format=tar tarball")
	argparser.StringVar(&args.GoTarFilename, "gotar", "", "Tarball of the Go stdlib source code")
	argparser.StringVar(&args.Package, "package", "", "The package(s) to report library usage for")
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
		return nil, errors.Errorf("expected 0 arguments, got %d: %q", argparser.NArg(), argparser.Args())
	}
	switch args.OutputFormat {
	case "txt":
		if args.OutputName != "" {
			return nil, errors.Errorf("--output-name is only valid for --output-mode=tar")
		}
	case "tar":
		if args.OutputName == "" {
			return nil, errors.Errorf("--output-name is required for --output-mode=tar")
		}
	default:
		return nil, errors.Errorf("--output-mode must be one of 'tar' or 'txt'")
	}
	if !strings.HasPrefix(filepath.Base(args.GoTarFilename), "go1.") || !strings.HasSuffix(args.GoTarFilename, ".tar.gz") {
		return nil, errors.Errorf("--gotar (%q) doesn't look like a go1.*.tar.gz file", args.GoTarFilename)
	}
	if args.Package == "" {
		return nil, errors.Errorf("--package (%q) must be non-empty", args.Package)
	}
	return args, nil
}

func main() {
	args, err := parseArgs()
	if err != nil {
		if err == pflag.ErrHelp {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "%s: %v\nTry '%s --help' for more information.\n", os.Args[0], err, os.Args[0])
		os.Exit(2)
	}
	if err := Main(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: fatal: %v\n", os.Args[0], err)
		os.Exit(1)
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
			fc, err := ioutil.ReadAll(goTar)
			if err != nil {
				return "", nil, err
			}
			version = "v" + strings.TrimPrefix(strings.TrimSpace(string(fc)), "go")
		case "go/LICENSE":
			fc, err := ioutil.ReadAll(goTar)
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
		return "", nil, errors.Errorf("file %q did not contain %q or %q", goTarFilename, "go/VERSION", "go/LICENSE")
	}
	return version, license, nil
}

func licenseIsProprietary(licenses map[detectlicense.License]struct{}) (bool, error) {
	_, proprietary := licenses[detectlicense.Proprietary]
	if proprietary && len(licenses) != 1 {
		return false, errors.New("mixed proprietary and open-source licenses")
	}
	return proprietary, nil
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
				if err := collectVendoredPkg(vendor, pkg); err != nil {
					return err
				}
			} else {
				if err := collectPkg(vendor, pkg); err != nil {
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
	for _, pkgName := range pkgNames {
		pkgLicenses[pkgName], err = detectlicense.DetectLicenses(pkgFiles[pkgName])
		if err != nil {
			return errors.Errorf(`%v
    This probably means that you added or upgraded a dependency, and the
    automated opensource-license-checker can't confidently detect what
    the license is.  (This is a good thing, because it is reminding you
    to check the license of libraries before using them.)

    You need to update the "github.com/datawire/ambassador/v2/pkg/mkopensource/detectlicense/licenses.go"
    file to correctly detect the license.`,
				errors.Wrapf(err, "package %q", pkgName))
		}
		if licenseIsStrongCopyleft(pkgLicenses[pkgName]) {
			return errors.Wrapf(errors.New("has an unacceptable license for use by Ambassador"),
				`package %q`, pkgName)
		}
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
	readme := new(bytes.Buffer)
	if args.Package == "mod" {
		modnames := make([]string, 0, len(mainMods))
		for modname := range mainMods {
			modnames = append(modnames, modname)
		}
		if len(mainMods) == 1 {
			readme.WriteString(wordwrap(75, fmt.Sprintf("The Go module %q incorporates the following Free and Open Source software:", modnames[0])))
		} else {
			sort.Strings(modnames)
			readme.WriteString(wordwrap(75, fmt.Sprintf("The Go modules %q incorporate the following Free and Open Source software:", modnames)))
		}
	} else if len(mainLibPkgs) == 0 {
		if len(mainCmdPkgs) == 1 {
			readme.WriteString(wordwrap(75, fmt.Sprintf("The program %q incorporates the following Free and Open Source software:", path.Base(mainCmdPkgs[0]))))
		} else {
			readme.WriteString(wordwrap(75, fmt.Sprintf("The programs %q incorporate the following Free and Open Source software:", args.Package)))
		}
	} else {
		if len(mainLibPkgs) == 1 {
			readme.WriteString(wordwrap(75, fmt.Sprintf("The Go package %q incorporates the following Free and Open Source software:", mainLibPkgs[0])))
		} else {
			readme.WriteString(wordwrap(75, fmt.Sprintf("The Go packages %q incorporate the following Free and Open Source software:", args.Package)))
		}
	}
	readme.WriteString("\n")
	table := tabwriter.NewWriter(readme, 0, 8, 2, ' ', 0)
	io.WriteString(table, "  \tName\tVersion\tLicense(s)\n")
	io.WriteString(table, "  \t----\t-------\t----------\n")
	for _, modKey := range modNames {
		proprietary, err := licenseIsProprietary(modLicenses[modKey])
		if err != nil {
			return errors.Wrapf(err, "module %q", modKey)
		}
		if proprietary {
			continue
		}
		modVal := modInfos[modKey]
		var depName, depVersion, depLicenses string
		if modVal == nil {
			depName = "the Go language standard library (\"std\")"
			depVersion = goVersion
		} else {
			depName = modVal.Path
			depVersion = modVal.Version
			if modVal.Replace != nil {
				if modVal.Replace.Version == "" {
					depVersion = "(modified)"
				} else {
					if modVal.Replace.Path != modVal.Path {
						depName = fmt.Sprintf("%s (modified from %s)", modVal.Replace.Path, modVal.Path)
					}
					depVersion = modVal.Replace.Version
				}
			}
		}

		licenseList := make([]string, 0, len(modLicenses[modKey]))
		for license := range modLicenses[modKey] {
			licenseList = append(licenseList, license.Name)
		}
		sort.Strings(licenseList)
		depLicenses = strings.Join(licenseList, ", ")
		if depLicenses == "" {
			panic(errors.Errorf("this should not happen: empty license string for %q", depName))
		}
		fmt.Fprintf(table, "\t%s\t%s\t%s\n", depName, depVersion, depLicenses)
	}
	table.Flush()
	if args.OutputFormat == "tar" {
		readme.WriteString("\n")
		readme.WriteString(wordwrap(75, "The appropriate license notices and source code are in correspondingly named directories."))
	}

	switch args.OutputFormat {
	case "txt":
		if _, err := readme.WriteTo(os.Stdout); err != nil {
			return err
		}
	case "tar":
		// Build a listing of all files to go in to the tarball
		tarfiles := make(map[string][]byte)
		tarfiles["OPENSOURCE.md"] = readme.Bytes()
		for pkgName := range pkgFiles {
			proprietary, err := licenseIsProprietary(pkgLicenses[pkgName])
			if err != nil {
				return errors.Wrapf(err, "package %q", pkgName)
			}
			switch {
			case proprietary:
				// don't include anything
			case licenseIsWeakCopyleft(pkgLicenses[pkgName]):
				// include everything
				for filename, filebody := range pkgFiles[pkgName] {
					tarfiles[filename] = filebody
				}
			default:
				// just include metadata
				for filename, filebody := range pkgFiles[pkgName] {
					if matchMetadata(filename) {
						tarfiles[filename] = filebody
					}
				}
			}
		}

		// Write output
		var outputFile *os.File = os.Stdout
		outputFile = os.Stdout
		defer outputFile.Close()
		outputCompressed := gzip.NewWriter(outputFile)
		defer outputCompressed.Close()
		outputTar := tar.NewWriter(outputCompressed)
		defer outputTar.Close()

		filenames := make([]string, 0, len(tarfiles))
		for filename := range tarfiles {
			filenames = append(filenames, filename)
		}
		sort.Strings(filenames)
		for _, filename := range filenames {
			body := tarfiles[filename]
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
