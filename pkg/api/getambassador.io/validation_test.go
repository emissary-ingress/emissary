package getambassadorio_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/datawire/dlib/dlog"
	getambassadorio "github.com/emissary-ingress/emissary/v3/pkg/api/getambassador.io"
	"github.com/emissary-ingress/emissary/v3/pkg/kates"
)

func TestValidation(t *testing.T) {
	jsonStr := `{
    "apiVersion":"getambassador.io/v2",
    "kind":"Mapping",
    "metadata":{
        "annotations":{
            "kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"getambassador.io/v3alpha1\",\"kind\":\"Mapping\",\"metadata\":{\"annotations\":{},\"name\":\"quote-rewrite\",\"namespace\":\"default\"},\"spec\":{\"hostname\":\"*\",\"prefix\":\"/ffs/\",\"rewrite\":\"\",\"service\":\"quote\"}}"
        },
        "creationTimestamp":"2022-01-19T00:11:43Z",
        "generation":1,
        "name":"quote-rewrite",
        "namespace":"default",
        "uid":"01b3ddea-24d7-45c6-a05a-64386f1b9588"
    },
    "spec":{
        "ambassador_id":[
            "--apiVersion-v3alpha1-only--default"
        ],
        "prefix":"/ffs/",
        "rewrite":"",
        "service":"quote"
    }
}`

	var obj kates.Unstructured
	require.NoError(t, json.Unmarshal([]byte(jsonStr), &obj.Object))

	validator := getambassadorio.NewValidator()
	ctx := dlog.NewTestContext(t, true)

	require.NoError(t, validator.Validate(ctx, &obj))
}

func TestValidationRetryBackOff(t *testing.T) {
	validator := getambassadorio.NewValidator()
	ctx := dlog.NewTestContext(t, true)

	mapping := func(t *testing.T, retryBackOff string) *kates.Unstructured {
		jsonStr := `{
    "apiVersion":"getambassador.io/v3alpha1",
    "kind":"Mapping",
    "metadata":{
        "name":"quote-backoff",
        "namespace":"default"
    },
    "spec":{
        "hostname":"*",
        "prefix":"/backend/",
        "service":"quote",
        "retry_policy":{
            "retry_on":"5xx",
            "retry_back_off":` + retryBackOff + `
        }
    }
}`

		var obj kates.Unstructured
		require.NoError(t, json.Unmarshal([]byte(jsonStr), &obj.Object))
		return &obj
	}

	t.Run("both intervals", func(t *testing.T) {
		require.NoError(t, validator.Validate(ctx,
			mapping(t, `{"base_interval":"0.025s","max_interval":"0.25s"}`)))
	})

	t.Run("just base_interval", func(t *testing.T) {
		require.NoError(t, validator.Validate(ctx,
			mapping(t, `{"base_interval":"1s"}`)))
	})

	t.Run("missing base_interval", func(t *testing.T) {
		require.Error(t, validator.Validate(ctx,
			mapping(t, `{"max_interval":"0.25s"}`)))
	})

	t.Run("interval that is not a number of seconds", func(t *testing.T) {
		require.Error(t, validator.Validate(ctx,
			mapping(t, `{"base_interval":"25ms"}`)))
	})
}
