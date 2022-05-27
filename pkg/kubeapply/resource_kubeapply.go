package kubeapply

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	_path "path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/datawire/ambassador/v2/pkg/k8s"
	"github.com/datawire/dlib/dexec"
)

var readyChecks = map[string]func(k8s.Resource) bool{
	"": func(_ k8s.Resource) bool { return false },
	"Deployment": func(r k8s.Resource) bool {
		// NOTE - plombardi - (2019-05-20)
		// a zero-sized deployment never gets status.readyReplicas and friends set by kubernetes deployment controller.
		// this effectively short-circuits the wait.
		//
		// in the future it might be worth porting this change to StatefulSets, ReplicaSets and ReplicationControllers
		if r.Spec().GetInt64("replicas") == 0 {
			return true
		}

		return r.Status().GetInt64("readyReplicas") > 0
	},
	"Service": func(r k8s.Resource) bool {
		return true
	},
	"Pod": func(r k8s.Resource) bool {
		css := r.Status().GetMaps("containerStatuses")
		for _, cs := range css {
			if !k8s.Map(cs).GetBool("ready") {
				return false
			}
		}
		return true
	},
	"Namespace": func(r k8s.Resource) bool {
		return r.Status().GetString("phase") == "Active"
	},
	"ServiceAccount": func(r k8s.Resource) bool {
		_, ok := r["secrets"]
		return ok
	},
	"ClusterRole": func(r k8s.Resource) bool {
		return true
	},
	"ClusterRoleBinding": func(r k8s.Resource) bool {
		return true
	},
	"CustomResourceDefinition": func(r k8s.Resource) bool {
		conditions := r.Status().GetMaps("conditions")
		if len(conditions) == 0 {
			return false
		}
		last := conditions[len(conditions)-1]
		return last["status"] == "True"
	},
}

// ReadyImplemented returns whether or not this package knows how to
// wait for this resource to be ready.
func ReadyImplemented(r k8s.Resource) bool {
	if r.Empty() {
		return false
	}
	kind := r.Kind()
	_, ok := readyChecks[kind]
	return ok
}

// Ready returns whether or not this resource is ready; if this
// package does not know how to check whether the resource is ready,
// then it returns true.
func Ready(r k8s.Resource) bool {
	if r.Empty() {
		return false
	}
	kind := r.Kind()
	fn, fnOK := readyChecks[kind]
	if !fnOK {
		return true
	}
	return fn(r)
}

func isTemplate(input []byte) bool {
	return strings.Contains(string(input), "@TEMPLATE@")
}

func image(ctx context.Context, dir, dockerfile string) (string, error) {
	iidfile, err := ioutil.TempFile("", "iid")
	if err != nil {
		return "", err
	}
	defer os.Remove(iidfile.Name())
	err = iidfile.Close()
	if err != nil {
		return "", err
	}

	dockerCtx := filepath.Dir(filepath.Join(dir, dockerfile))
	cmd := dexec.CommandContext(ctx, "docker", "build", "-f", filepath.Base(dockerfile), ".", "--iidfile", iidfile.Name())
	cmd.Dir = dockerCtx
	err = cmd.Run()
	if err != nil {
		return "", err
	}
	content, err := ioutil.ReadFile(iidfile.Name())
	if err != nil {
		return "", err
	}
	iid := strings.Split(strings.TrimSpace(string(content)), ":")[1]
	short := iid[:12]

	registry := strings.TrimSpace(os.Getenv("DOCKER_REGISTRY"))
	if registry == "" {
		return "", errors.Errorf("please set the DOCKER_REGISTRY environment variable")
	}
	tag := fmt.Sprintf("%s/kubeapply:%s", registry, short)

	if err := dexec.CommandContext(ctx, "docker", "tag", iid, tag).Run(); err != nil {
		return "", err
	}

	cmd = dexec.CommandContext(ctx, "docker", "push", tag)
	if err := cmd.Run(); err != nil {
		return "", err
	}

	return tag, nil
}

// ExpandResource takes a path to a YAML file, and returns its
// contents, with any kubeapply templating expanded.
func ExpandResource(ctx context.Context, path string) (result []byte, err error) {
	input, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", path, err)
	}
	if isTemplate(input) {
		funcs := sprig.TxtFuncMap()
		usedImage := false
		funcs["image"] = func(dockerfile string) (string, error) {
			usedImage = true
			return image(ctx, filepath.Dir(path), dockerfile)
		}
		tmpl := template.New(filepath.Base(path)).Funcs(funcs)
		_, err := tmpl.Parse(string(input))
		if err != nil {
			return nil, fmt.Errorf("%s: %v", path, err)
		}

		buf := bytes.NewBuffer(nil)
		err = tmpl.ExecuteTemplate(buf, filepath.Base(path), nil)
		if err != nil {
			return nil, fmt.Errorf("%s: %v", path, err)
		}

		result = buf.Bytes()

		if usedImage && os.Getenv("DEV_USE_IMAGEPULLSECRET") != "" {
			dockercfg, err := json.Marshal(map[string]interface{}{
				"auths": map[string]interface{}{
					_path.Dir(os.Getenv("DEV_REGISTRY")): map[string]string{
						"auth": base64.StdEncoding.EncodeToString([]byte(os.Getenv("DOCKER_BUILD_USERNAME") + ":" + os.Getenv("DOCKER_BUILD_PASSWORD"))),
					},
				},
			})
			if err != nil {
				return nil, errors.Wrap(err, "DEV_USE_IMAGEPULLSECRET")
			}

			secretYaml := fmt.Sprintf(`

---
apiVersion: v1
kind: Secret
metadata:
  name: dev-image-pull-secret
type: kubernetes.io/dockerconfigjson
data:
  ".dockerconfigjson": %q
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: default
imagePullSecrets:
- name: dev-image-pull-secret
`, base64.StdEncoding.EncodeToString(dockercfg))

			result = append(result, secretYaml...)
		}
	} else {
		result = input
	}

	return
}

// LoadResources is like ExpandResource, but follows it up by actually
// parsing the YAML.
func LoadResources(ctx context.Context, path string) (result []k8s.Resource, err error) {
	var input []byte
	input, err = ExpandResource(ctx, path)
	if err != nil {
		return
	}
	result, err = k8s.ParseResources(path, string(input))
	return
}

// SaveResources serializes a list of k8s.Resources to a YAML file.
func SaveResources(path string, resources []k8s.Resource) error {
	output, err := MarshalResources(resources)
	if err != nil {
		return fmt.Errorf("%s: %v", path, err)
	}
	err = ioutil.WriteFile(path, output, 0644)
	if err != nil {
		return fmt.Errorf("%s: %v", path, err)
	}
	return nil
}

// MarshalResources serializes a list of k8s.Resources in to YAML.
func MarshalResources(resources []k8s.Resource) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	e := yaml.NewEncoder(buf)
	for _, r := range resources {
		err := e.Encode(r)
		if err != nil {
			return nil, err
		}
	}
	e.Close()
	return buf.Bytes(), nil
}
