// The kates package is a library for writing kubernetes extensions. The library provides a number
// of capabilities:
//
//   - Consistent bootstrap of multiple resources
//   - Graceful Load Shedding via Coalesced Events
//   - Read/write coherence
//   - Grouping
//   - Works well with typed (e.g. corev1.Pod) and untyped
//     (e.g. map[string]interface{}) representations of k8s resources.
//
// It does not provide codegen or admissions controllers, for those we should use kubebuilder.
//
// Comparison to other libraries:
//   - higher level, simpler, and more idiomatic than client-go
//   - lower level (and more flexible) than operator-sdk or kubebuilder
//
// # Constructing a Client
//
// The primary entrypoint for the kates library is the Client type. A Client is constructed by
// passing in the ClientConfig struct with a path to a kubeconfig file:
//
//	client, err := NewClient(ClientConfig{Kubeconfig: "path/to/kubeconfig"}) // or NewClient(ClientConfig{}) for defaults
//
// # Creating, Modifying, and Deleting Resources
//
// A client can be used to Create, Update, and/or Delete any kubernetes resource. Each of the "CRUD"
// methods will, upon success, store an updated copy of the resource into the object referenced by
// the last argument. This will typically be different than the value you supplied if e.g. the
// server defaults fields, updates the resource version, assigns UIDs, etc.
//
//	var result kates.Pod
//	err = client.Create(ctx, &kates.Pod{...}, &result)
//	err = client.Update(ctx, result, &result)
//	err = client.UpdateStatus(ctx, result, &result)
//	err = client.Delete(ctx, result, &result)
//
// You can pass both typed and untyped values into the APIs. The only requirement is that the values
// you pass will json.Marshal to and json.Unmarshal from something that looks like a kubernetes
// resource:
//
//	pod := kates.Pod{...}
//	err := client.Create(ctx, &pod, &pod)
//	// -or-
//	pod := map[string]interface{}{"kind": "Pod", ...}
//	err := client.Create(ctx, &pod, &pod)
//
// # Watching Resources
//
// The client can be used to watch sets of multiple related resources. This is accomplished via the
// Accumulator type. An accumulator tracks events coming from the API server for the indicated
// resources, and merges those events with any locally initiated changes made via the client in
// order to maintain a snapshot that is coherent.
//
// You can construct an Accumulator by invoking the Client's Watch method:
//
//	accumulator = client.Watch(ctx,
//	                           Query{Name: "Services", Kind: "svc"},
//	                           Query{Name: "Deployments", Kind: "deploy"})
//
// The Accumulator will bootstrap a complete list of each supplied Query, and then provide
// continuing notifications if any of the resources change. Notifications that the initial bootstrap
// is complete as well as notifications of any subsequent changes are indicated by sending an empty
// struct down the Accumulator.Changed() channel:
//
//	<-accumulator.Changed() // Wait until all Queries have been initialized.
//
// The Accumulator provides access to the values it tracks via the Accumulator.Update(&snapshot)
// method. The Update() method expects a pointer to a snapshot that is defined by the caller. The
// caller must supply a pointer to a struct with fields that match the names of the Query structs
// used to create the Accumulator. The types of the snapshot fields are free to be anything that
// will json.Unmarshal from a slice of kubernetes resources:
//
//	// initialize an empty snapshot
//	snapshot := struct {
//	    Services    []*kates.Service
//	    Deployments []*kates.Deployment
//	}{}
//
//	accumulator.Update(&snapshot)
//
// The first call to update will populate the snapshot with the bootstrapped values. At this point
// any startup logic can be performed with confidence that the snapshot represents a complete and
// recent view of cluster state:
//
//	// perform any startup logic
//	...
//
// The same APIs can then be  used to watch for and reconcile subsequent changes:
//
//	// reconcile ongoing changes
//	for {
//	    select {
//	        case <-accumulator.Changed():
//	            wasChanged = accumulator.Update(&snapshot)
//	            if wasChanged {
//	                reconcile(snapshot)
//	            }
//	        case <-ctx.Done():
//	            break
//	    }
//	}
//
// The Accumulator will provide read/write coherence for any changes made using the client from
// which the Accumulator was created. This means that any snapshot produced by the Accumulator is
// guaranteed to include all the Create, Update, UpdateStatus, and/or Delete operations that were
// performed using the client. (The underlying client-go CRUD and Watch operations do not provide
// this guarantee, so a straighforward reconcile operation will often end up creating duplicate
// objects and/or performing updates on stale versions.)
//
// # Event Coalescing for Graceful Load Shedding
//
// The Accumulator automatically coalesces events behind the scenes in order to facilitate graceful
// load shedding for situations where the reconcile operation takes a long time relative to the
// number of incoming events. This allows the Accumulator.Update(&snapshot) method to provide a
// guarantee that when it returns the snapshot will contain the most recent view of cluster state
// regardless of how slowly and infrequently we read from the Accumulator.Changed() channel:
//
//	snapshot := Snapshot{}
//	for {
//	    <-accumulator.Changed()
//	    wasChanged := accumulator.Update(&snapshot) // Guaranteed to return the most recent view of cluster state.
//	    if wasChanged {
//	        slowReconcile(&snapshot)
//	    }
//	}
package kates

// TODO:
// - Comment explaining what the different client-go pieces are, what the pieces here are, and how they fit together: What is an "informer", what is a "RESTMapper", what is an "accumulator"? How do they fit together?
// - FieldSelector is omitted.
// - LabelSelector is stringly typed.
// - Add tests to prove that Update followed by Get/List is actually synchronous and doesn't require patchWatch type functionality.

/** XXX: thoughts...
 *
 * Problems with the way we currently write controllers:
 *
 *  - delayed write propagation
 *  - typed vs untyped
 *  - detecting no resources
 *  - detecting when multiple watches are synced
 *  - fetching references
 *  - handling conflicts, fundamentally need to retry, but at what granularity?
 *  - resilience to poison inputs
 *  - garbage collection/ownership
 *
 * With a partition function, this could get a lot more efficient and resilient.
 * What would a partition function look like?
 *  - index pattern? f(item) -> list of accumulators
 *  - single kind is easy, right now f(item) -> constant, f(mapping)->prefix
 *  - how does multiple work?
 *
 * Accumulator can probabily be merged with client since we don't really need inner and outer in the same snapshot.
 *
 *   cli := ...
 *
 *   acc := cli.Watch(ctx, ...) // somehow include partition factory and index function?
 *
 *   cli.CRUD(ctx, ...)
 *
 *   partition := <-acc.Changed()    // returns active partition?
 *
 *   acc.Update(partition)
 *
 * --
 *
 *  project, revision, jobs, jobs-podses, mapping, service, deployments, deployments-podses
 *
 *  simple: f(obj) -> partition-key(s)
 *
 *  escape hatches:
 *   - f(obj) -> * (every partition gets them), f(obj) -> "" no partition gets them but you can query them
 *   - one partition
 * --
 *
 */
