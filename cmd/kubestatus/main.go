package kubestatus

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/datawire/dlib/dlog"
	"github.com/emissary-ingress/emissary/v3/pkg/kates"
)

func Main(ctx context.Context, version string, args ...string) error {
	var st = &cobra.Command{
		Use:           "kubestatus <kind> [<name>]",
		Short:         "get and set status of kubernetes resources",
		Args:          cobra.RangeArgs(1, 2),
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	fields := st.Flags().StringP("field-selector", "f", "", "field selector")
	labels := st.Flags().StringP("label-selector", "l", "", "label selector")
	statusFile := st.Flags().StringP("update", "u", "", "update with new status from file (must be json)")
	kubeconfig := kates.NewConfigFlags(false)
	kubeconfig.AddFlags(st.Flags())

	st.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		var status map[string]interface{}

		if *statusFile != "" {
			rawStatus, err := os.Open(*statusFile)
			if err != nil {
				return err
			}
			defer rawStatus.Close()

			dec := json.NewDecoder(rawStatus)
			err = dec.Decode(&status)
			if err != nil {
				return err
			}
		}

		client, err := kates.NewClientFromConfigFlags(kubeconfig)
		if err != nil {
			return err
		}

		kind := args[0]
		namespace, err := client.CurrentNamespace()
		if err != nil {
			return err
		}

		name := ""
		if len(args) == 2 {
			name = args[1]
		}

		// Special case and supply the name argument so we use Get instead of List if we can
		// tell the FieldSelector is equivalent to a Get. This appeared to show a
		// performance improvement at one point, but it was later discovered that the
		// improvement actually came from the kates client doing discovery caching.
		if *labels == "" {
			parts := strings.Split(*fields, ",")
			if len(parts) == 1 {
				parts = strings.Split(parts[0], "=")
				if len(parts) == 2 {
					lhs := strings.TrimSpace(parts[0])
					if lhs == "metadata.name" {
						name = strings.TrimSpace(parts[1])
					}
				}
			}
		}

		if name != "" {
			obj := kates.NewUnstructured(kind, "")
			obj.SetName(name)
			if namespace != "" {
				obj.SetNamespace(namespace)
			}
			err = client.Get(ctx, obj, obj)
			if err != nil {
				return err
			}

			if *statusFile == "" {
				fmt.Println("Status of", obj.GetKind(), obj.GetName(), "in namespace",
					obj.GetNamespace())
				fmt.Printf("  %v\n", obj.Object["status"])
				return nil
			} else {
				obj.Object["status"] = status
				return client.UpdateStatus(ctx, obj, obj)
			}
		}

		var items []*kates.Unstructured

		err = client.List(ctx,
			kates.Query{
				Kind:          kind,
				Namespace:     namespace,
				FieldSelector: *fields,
				LabelSelector: *labels,
			},
			&items)

		if err != nil {
			return err
		}

		for _, obj := range items {
			if *statusFile == "" {
				// The user is asking for the status, so print it.
				fmt.Println("Status of", obj.GetKind(), obj.GetName(), "in namespace",
					obj.GetNamespace())
				fmt.Printf("  %v\n", obj.Object["status"])
			} else {
				// The user is asking for a status update.
				// log.Debugf doesn't exist.
				if false {
					fmt.Println("Updating", obj.GetName(), "in namespace", obj.GetNamespace())
				}

				obj.Object["status"] = status
				err = client.UpdateStatus(ctx, obj, nil)
				if err != nil {
					dlog.Debugf(ctx, "error updating resource: %v", err)
				}
			}
		}

		return nil
	}

	st.SetArgs(args)
	return st.ExecuteContext(ctx)
}
