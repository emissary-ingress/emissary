package detectlicense

import (
	"fmt"
	"strings"
)

func isAmbassadorProprietarySoftware(packageName string) bool {
	const SmartAgentRepo = "github.com/datawire/telepresence2-proprietary/"
	const AmbassadorCloudRepo = "github.com/datawire/saas_app/"
	const TelepresencePro = "github.com/datawire/telepresence-pro/"

	return strings.HasPrefix(packageName, SmartAgentRepo) || strings.HasPrefix(packageName, AmbassadorCloudRepo) || strings.HasPrefix(packageName, TelepresencePro)
}

// knownDependencies will return a list of licenses for any dependency that has been
// hardcoded due to the difficulty to parse the license file(s).
func knownDependencies(dependencyName string, dependencyVersion string) (licenses []License, ok bool) {
	hardcodedGoDependencies := map[string][]License{
		"github.com/josharian/intern@v1.0.1-0.20211109044230-42b52b674af5":       {MIT},
		"github.com/dustin/go-humanize@v1.0.0":                                   {MIT},
		"github.com/garyburd/redigo/internal@v0.0.0-20150301180006-535138d7bcd7": {Apache2},
		"github.com/garyburd/redigo/redis@v0.0.0-20150301180006-535138d7bcd7":    {Apache2},
	}

	licenses, ok = hardcodedGoDependencies[fmt.Sprintf("%s@%s", dependencyName, dependencyVersion)]
	return
}
