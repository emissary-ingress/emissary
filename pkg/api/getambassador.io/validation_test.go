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
