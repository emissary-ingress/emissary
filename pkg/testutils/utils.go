package testutils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v3alpha1"
	"github.com/datawire/ambassador/v2/pkg/kates"
)

func JSONify(obj interface{}) (string, error) {
	bytes, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func LoadYAML(path string) ([]kates.Object, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	objs, err := kates.ParseManifests(string(content))
	if err != nil {
		return nil, err
	}

	return objs, nil
}

func GlobMatch(what string, text string, pattern string, regex bool) (bool, string, string, error) {
	var err error
	rc := false
	authority := ""
	authorityMatch := ""

	if regex {
		// The hostname is a glob, and determining if a regex and a glob match each
		// other is (possibly NP-)hard, so meh, we'll say they always match.
		rc = true
		authority = pattern
		authorityMatch = "re~"
	} else if strings.HasPrefix(pattern, "*") || strings.HasSuffix(pattern, "*") {
		// It's a supportable glob.
		globre := strings.Replace(pattern, ".", "\\.", -1)
		globre = strings.Replace(globre, "*", "[^\\.]+", -1)
		globre = "^" + globre + "$"

		rc, err = regexp.MatchString(globre, text)
		if err != nil {
			return false, "", "", err
		}

		authority = pattern
		authorityMatch = "gl~"
	} else {
		// Nothing special, so exact match.
		rc = (pattern == text)
		authority = pattern
		authorityMatch = "=="
	}

	fmt.Printf("GlobMatch %s: '%s' %s '%s' == %v\n", what, text, authorityMatch, authority, rc)
	return rc, authority, authorityMatch, nil
}

func HostMatch(mapping v3alpha1.Mapping, host v3alpha1.Host) (bool, string, string, error) {
	hostName := host.Spec.Hostname

	if mapping.Spec.Hostname != "" {
		fmt.Printf("HostMatch: hostName %s mappingHost %s\n", hostName, mapping.Spec.Hostname)
		return GlobMatch("Hostname", hostName, mapping.Spec.Hostname, false)
	}

	if mapping.Spec.DeprecatedHost != "" {
		mappingHost := mapping.Spec.DeprecatedHost

		mappingHostRegex := false
		if mapping.Spec.DeprecatedHostRegex != nil {
			mappingHostRegex = *mapping.Spec.DeprecatedHostRegex
		}

		fmt.Printf("HostMatch: hostName %s mappingHost %s\n", hostName, mappingHost)
		return GlobMatch("Host", hostName, mappingHost, mappingHostRegex)
	}

	// No host in the Mapping -- how about authority?
	mappingAuthorityRegex := false
	mappingAuthorityBoolOrString, found := mapping.Spec.Headers[":authority"]
	mappingAuthority := ""

	if found && (mappingAuthorityBoolOrString.String != nil) {
		mappingAuthority = *mappingAuthorityBoolOrString.String
	} else {
		// Try a regex authority.
		mappingAuthorityBoolOrString, found = mapping.Spec.RegexHeaders[":authority"]

		if found && (mappingAuthorityBoolOrString.String != nil) {
			mappingAuthorityRegex = true
			mappingAuthority = *mappingAuthorityBoolOrString.String
		}
	}

	fmt.Printf("HostMatch: mappingAuthority %s\n", mappingAuthority)

	if mappingAuthority != "" {
		return GlobMatch("Authority", hostName, mappingAuthority, mappingAuthorityRegex)
	}

	fmt.Printf("HostMatch: fallthrough\n")
	// If we're here, there's no host to match, so return true.
	return true, "", "", nil
}
