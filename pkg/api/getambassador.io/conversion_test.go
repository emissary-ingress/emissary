package v2

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"testing"

	"github.com/stretchr/testify/require"

	v2 "github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v2"
	v3alpha1 "github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v3alpha1"
)

func TestAuthService(t *testing.T) {
	bytes, err := ioutil.ReadFile(path.Join("v2/testdata", "authsvc.json"))
	require.NoError(t, err)

	var v2AuthServices []v2.AuthService

	err = json.Unmarshal(bytes, &v2AuthServices)
	require.NoError(t, err)

	for _, v2authsvc := range v2AuthServices {
		var v3authsvc v3alpha1.AuthService

		err := v3authsvc.ConvertFrom(&v2authsvc)
		require.NoError(t, err)

		v2bytes, _ := json.MarshalIndent(v2authsvc, "", "  ")
		v3bytes, err := json.MarshalIndent(v3authsvc, "", "  ")
		require.NoError(t, err)

		fmt.Printf("V2: %s\n", string(v2bytes))
		fmt.Printf("V3: %s\n", string(v3bytes))
	}
}
