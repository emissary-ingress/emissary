package dependency

import "github.com/datawire/go-mkopensource/pkg/detectlicense"

var hardcodedDependencies = map[string][]string{
	"cyclist@0.2.2": []string{detectlicense.MIT.Name},
	"indexof@0.0.1": []string{detectlicense.MIT.Name},
	"pako@1.0.10":   []string{detectlicense.MIT.Name},
}
