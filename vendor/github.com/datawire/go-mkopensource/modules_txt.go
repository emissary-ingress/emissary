package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/datawire/go-mkopensource/pkg/golist"
)

// VendorList returns a listing of all packages in
// `vendor/modules.txt`, which is superior to `go list -deps` in that
// it includes dependencies for all platforms and build
// configurations, but inferior in that it cannot be asked to only
// consider dependencies of a specific package rather than the whole
// module.
func VendorList() ([]golist.Package, error) {
	// References: In the Go stdlib source code, see
	// - `cmd/go/internal/modcmd/vendor.go` for the code that writes modules.txt, and
	// - `cmd/go/internal/modload/vendor.go` for the code that parses it.
	cmd := exec.Command("go", "mod", "vendor")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%q: %w", []string{"go", "mod", "vendor"}, err)
	}

	file, err := os.Open("vendor/modules.txt")
	if err != nil {
		if os.IsNotExist(err) {
			// If there are no dependencies outside of stdlib.
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	var pkgs []golist.Package
	var curModuleName string
	var curModule *golist.Module // lazily populated from curModuleName
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "## ") {
			// These lines are introduced in Go 1.17 and indicate (1) the Go version in
			// go.mod, and (2) whether we implicitly or explicitly depend on it; neither
			// of which are things we care about.
		} else if strings.HasPrefix(line, "# ") {
			parts := strings.Split(line, " ")
			// Just do some quick validation of the line format.  We're not tring to be
			// super strict with the validation, just a quick check that we're not
			// looking at something totally insane.
			switch len(parts) {
			case 3:
				// 0 1      2
				// # module version
			case 4, 5, 6:
				// 0 1      2       3      4       5
				// # module version =>     module version
				// # module =>      module version
				// # module version =>     path
				// # module =>      path
				if parts[2] != "=>" && parts[3] != "=>" {
					return nil, fmt.Errorf("malformed line in vendor/modules.txt: %q", line)
				}
			default:
				return nil, fmt.Errorf("malformed line in vendor/modules.txt: %q", line)
			}
			// Defer looking up curModule from curModuleName until we actually need it;
			// a non-used replaced module might not be present in `vendor/`.  We could
			// instead download it by using `-mod=readonly` instead of `-mod=vendor`,
			// but what would the point in that be?
			curModuleName = parts[1]
			curModule = nil
		} else {
			if curModule == nil && curModuleName != "" {
				modules, err := golist.GoListModules([]string{"-mod=vendor"}, []string{curModuleName})
				if err != nil {
					return nil, err
				}
				if len(modules) != 1 {
					return nil, errors.New("unexpected output from go list")
				}
				curModule = &modules[0]
			}
			pkgname := line
			pkgs = append(pkgs, golist.Package{
				Dir:        "vendor/" + pkgname,
				ImportPath: pkgname,
				Name:       path.Base(pkgname),
				Module:     curModule,
				DepOnly:    true,
			})
		}
	}

	return pkgs, nil
}
