package reproducer

import (
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/datawire/ambassador/pkg/kates"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create ( <snapshot> | <extraction> | <archive> )",
	Short: "create produces a working set of manifests based on snapshots and/or extractions produce by the extract subcommand",
	Long: `The create subcommand is designed for creating high fidelity reproductions of a source cluster using edge-stack. All of the ambassador inputs are recreated exactly as they are in the source cluster. All the services in the source cluster are also recreated, but they are transformed to point to a single set of pods with a "service: reproducer" label. This allows for a high fidelity working reproduction of the source cluster without requiring access to any of the propriety deployments in the source cluster.

The output of the create command can be passed directly to kubectl, e.g.:

    reproducer create sanitized.tgz | kubectl apply -f -

You can also save the output and hand edit it should you need to tweak some of the details:

    reproducer create sanitized.tgz > repro.yaml
    vi repro.yaml
    kubectl apply -f repro.yaml
`,
	Args: cobra.ExactArgs(1),
	RunE: create,
}

func create(cmd *cobra.Command, args []string) error {
	filename := args[0]

	extensions := map[string]bool{
		".yml":  true,
		".yaml": true,
		".json": true,
	}

	repro := NewRepro()

	err := search(filename, func(path, contentType, encoding string, content []byte) error {
		base := filepath.Base(path)
		ext := filepath.Ext(base)
		name := base[:len(base)-len(ext)]

		if !extensions[ext] {
			log.Printf("skipping %s", path)
			return nil
		}

		if !(name == "snapshot" || name == "manifests") {
			log.Printf("skipping %s", path)
			return nil
		}

		log.Printf("found resources from %s", path)

		switch name {
		case "snapshot":
			var snapshot struct {
				Kubernetes map[string][]*kates.Unstructured
			}
			err := json.Unmarshal(content, &snapshot)
			if err != nil {
				return errors.Wrapf(err, "decoding snapshot at %s", path)
			}

			for _, values := range snapshot.Kubernetes {
				for _, resource := range values {
					err := repro.Add(resource)
					if err != nil {
						return errors.Wrapf(err, "adding resource")
					}
				}
			}
		case "manifests":
			resources, err := kates.ParseManifests(string(content))
			if err != nil {
				return errors.Wrapf(err, "decoding manifests at %s", path)
			}

			for _, resource := range resources {
				err := repro.Add(resource)
				if err != nil {
					return errors.Wrapf(err, "adding resource")
				}
			}
		}

		return breakSearch
	})

	if err == nil {
		return errors.Errorf("unable to find suitable snapshot in %s", filename)
	}

	if err != breakSearch {
		return err
	}

	// Process all the resources we found.
	err = repro.Process()
	if err != nil {
		return err
	}

	// Marshal all the transformed resources.
	marshalled, err := repro.Marshal()
	if err != nil {
		return err
	}

	fmt.Print(string(marshalled))

	return nil
}

type Repro struct {
	Resources  map[string][]*kates.Unstructured
	Namespaces map[string]bool
	Ports      map[string]bool
	Processed  []*kates.Unstructured
}

func NewRepro() *Repro {
	return &Repro{
		Resources:  map[string][]*kates.Unstructured{},
		Namespaces: map[string]bool{},
		Ports:      map[string]bool{},
	}
}

// Add an input resource from the source cluster.
func (r *Repro) Add(resource kates.Object) error {
	un, err := kates.NewUnstructuredFromObject(resource)
	if err != nil {
		return err
	}
	gvk := resource.GetObjectKind().GroupVersionKind()
	r.Resources[gvk.Kind] = append(r.Resources[gvk.Kind], un)
	return nil

}

func (r *Repro) Process() error {
	// Process resources in order.
	for _, key := range r.OrderedKinds() {
		values, ok := r.Resources[key]
		if ok {
			delete(r.Resources, key)
			for _, resource := range values {
				p := r.callProcess(resource)
				if p != nil {
					r.Processed = append(r.Processed, p)
				}
			}
		}
	}

	// Remove any included namespaces
	for _, p := range r.Processed {
		if p.GetObjectKind().GroupVersionKind().Kind == "Namespace" {
			delete(r.Namespaces, p.GetName())
		}
	}

	// Auto create any missing namespaces and prepend so they are defined before being used.
	ns := []*kates.Unstructured{}
	for _, k := range sortedKeys(r.Namespaces) {
		un, err := kates.NewUnstructuredFromObject(&kates.Namespace{
			TypeMeta:   kates.TypeMeta{APIVersion: "v1", Kind: "Namespace"},
			ObjectMeta: kates.ObjectMeta{Name: k},
		})
		if err != nil {
			return errors.Wrapf(err, "error creating namespace %s", k)
		}
		ns = append(ns, un)
	}

	r.Processed = append(ns, r.Processed...)

	return nil
}

// OrderedKinds returns all the k8s kinds in the proper order to avoid dangling references.
func (r *Repro) OrderedKinds() []string {
	return append([]string{
		"CustomResourceDefinition",
		"Namespace",
		"ServiceAccount",
		"ClusterRole",
		"ClusterRoleBinding",
		"Role",
		"RoleBinding",
		"Secret",
	},
		sortedKeys(r.Resources)...)
}

func (r *Repro) callProcess(resource *kates.Unstructured) *kates.Unstructured {
	if len(resource.GetOwnerReferences()) > 0 {
		return nil
	}
	if resource.GetNamespace() == "kube-system" {
		return nil
	}

	gvk := resource.GetObjectKind().GroupVersionKind()
	switch gvk.Kind {
	case "APIService":
		return nil
	case "ComponentStatus":
		return nil
	case "EndpointSlice":
		return nil
	case "Endpoints":
		return nil
	case "Event":
		return nil
	case "Lease":
		return nil
	case "Node":
		return nil
	case "NodeMetrics":
		return nil
	case "PodMetrics":
		return nil
	case "StorageClass":
		return nil
	case "PriorityClass":
		return nil
	}

	obj, err := kates.NewObjectFromUnstructured(resource)
	if err != nil {
		log.Printf("error processing object: %+v", err)
		return nil
	}

	obj = r.process(obj)

	// convert back to unstructured so we serialize prettier, e.g. no creationTimestamp: null
	result, err := kates.NewUnstructuredFromObject(obj)
	if err != nil {
		log.Printf("error making unstructured from object: %+v", err)
		return nil
	}

	return clean(result)
}

func (r *Repro) process(object kates.Object) kates.Object {
	r.Namespaces[object.GetNamespace()] = true

	rbac := false
	switch obj := object.(type) {
	case *kates.Service:
		obj.Spec.ClusterIP = ""
		if !isAmbassadorResource(object) {
			obj.Spec.Selector = map[string]string{
				"service": "reproducer",
			}
			for _, port := range obj.Spec.Ports {
				r.Ports[port.TargetPort.String()] = true
			}
		}
	case *kates.ClusterRole:
		rbac = true
	case *kates.ClusterRoleBinding:
		rbac = true
	case *kates.Role:
		rbac = true
	case *kates.RoleBinding:
		rbac = true
	case *kates.ServiceAccount:
		rbac = true
		if obj.GetName() == "default" {
			return nil
		}
	}

	if rbac && strings.Contains(object.GetName(), "system:") {
		return nil
	}

	return object
}

const lastApplied = "kubectl.kubernetes.io/last-applied-configuration"
const bootstrappingLabel = "kubernetes.io/bootstrapping"

// Clean does generic cleanup of resources from the source cluster. Kubectl and/or the API server
// will add a bunch of annotations about last-applied-configurations and managed fields and what
// not, and these annotations will make kubectl and/or the API server barf if present on a resource
// supplied to `kubectl apply`.
func clean(resource *kates.Unstructured) *kates.Unstructured {
	if resource == nil {
		return nil
	}

	ann := resource.GetAnnotations()
	if ann == nil {
		ann = map[string]string{}
	}
	delete(ann, lastApplied)

	labels := resource.GetLabels()
	_, ok := labels[bootstrappingLabel]
	if ok {
		return nil
	}

	if len(ann) > 0 {
		resource.SetAnnotations(ann)
	} else {
		resource.SetAnnotations(nil)
	}
	resource.SetManagedFields(nil)
	resource.SetCreationTimestamp(kates.Time{Time: time.Time{}})
	resource.SetUID("")
	resource.SetResourceVersion("")
	resource.SetSelfLink("")
	resource.SetGeneration(0)
	delete(resource.Object, "status")
	return resource
}

func (r *Repro) Marshal() ([]byte, error) {
	return marshalManifests(r.Processed)
}

func sortedKeys(m interface{}) (result []string) {
	mval := reflect.ValueOf(m)
	for _, v := range mval.MapKeys() {
		result = append(result, v.Interface().(string))
	}
	sort.Strings(result)
	return
}
