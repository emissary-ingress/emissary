package runner

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/datawire/apro/cmd/amb-sidecar/types"
	lyftserver "github.com/lyft/ratelimit/src/server"
)

func DevSetup(cfg types.Config, httpHandler lyftserver.DebugHTTPHandler) {
	httpHandler.AddEndpoint("/dev/tests", "mocha test endpoint", mochaTester(cfg.DevWebUIDir))
	httpHandler.AddEndpoint("/dev/", "developer documentation",
		http.StripPrefix("/dev", http.FileServer(http.Dir(cfg.DevWebUIDir))).ServeHTTP)
}

// Serve a standalone test index that loads pinned versions of mocha
// chai from a CDN with SRI codes. The index will automatically scan
// for all javascript test files and include html to load
// them. Javascript test files count as any test file underneath a
// "/tests/" that ends in ".js"
func mochaTester(rootPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var testFiles []string
		err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if strings.Contains(path, "/tests/") && strings.HasSuffix(path, ".js") {
				rel, err := filepath.Rel(rootPath, path)
				if err != nil {
					return err
				}
				testFiles = append(testFiles, rel)
			}
			return nil
		})

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var scripts []string
		for _, file := range testFiles {
			scripts = append(scripts, fmt.Sprintf(`<script src="/%s" type="module"></script>`, file))
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(fmt.Sprintf(TEST_INDEX, strings.Join(scripts, "\n    "))))
	}
}

const TEST_INDEX = `
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <title>Mocha</title>
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/mocha/7.1.1/mocha.css" integrity="sha256-a4j/9d0TJl1bbNF1UC8zUh8pnur9RLQIyyaGAVtj8fM=" crossorigin="anonymous" />
  </head>
  <body>
    <div id="mocha"></div>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/chai/4.2.0/chai.js" integrity="sha256-Oe35Xz+Zi1EffYsTw5ENBhzOS06LOTV5PSV4OVvnyU8=" crossorigin="anonymous"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/mocha/7.1.1/mocha.js" integrity="sha256-gJZVi9hHQ8zv2cZLFQOVja7GVXm5gflG1S7azqrVqsg=" crossorigin="anonymous"></script>

    <script>mocha.setup('bdd');</script>
    <script>
      window.onload = ()=>{
        mocha.run();
      }
    </script>

    %s

  </body>
</html>
`
