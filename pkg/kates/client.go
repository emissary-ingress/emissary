package kates

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"

	// k8s libraries
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/disk"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kubectl/pkg/polymorphichelpers"

	// k8s types
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// k8s plugins
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	kates_internal "github.com/datawire/ambassador/v2/pkg/kates_internal"
	"github.com/datawire/dlib/dlog"
)

// The Client struct provides an interface to interact with the kubernetes api-server. You can think
// of it like a programatic version of the familiar kubectl command line tool. In fact a goal of
// these APIs is that where possible, your knowledge of kubectl should translate well into using
// these APIs. It provides a golang-friendly way to perform basic CRUD and Watch operations on
// kubernetes resources, as well as providing some additional capabilities.
//
// Differences from kubectl:
//
//  - You can also use a Client to update the status of a resource.
//  - The Client provides read/write coherence (more about this below).
//  - The Client provides load shedding via event coalescing for watches.
//  - The Client provides bootstrapping of multiple watches.
//
// The biggest difference from kubectl (and also from using client-go directly) is the Read/Write
// coherence it provides. Kubernetes Watches are inherently asynchronous. This means that if a
// kubernetes resource is modified at time T0, a client won't find out about it until some later
// time T1. It is normally difficult to notice this since the delay may be quite small, however if
// you are writing a controller that uses watches in combination with modifying the resources it is
// watching, the delay is big enough that a program will often be "notified" with versions of
// resources that do not included updates made by the program itself. This even happens when a
// program has a lock and is guaranteed to be the only process modifying a given resource. Needless
// to say, programming against an API like this can make for some brain twisting logic. The Client
// struct allows for much simpler code by tracking what changes have been made locally and updating
// all Watch results with the most recent version of an object, thereby providing the guarantee that
// your Watch results will *always* include any changes you have made via the Client performing the
// watch.
//
// Additionally, the Accumulator API provides two key pieces of watch related functionality:
//
//   1. By coalescing multiple updates behind the scenes, the Accumulator API provides a natural
//      form of load shedding if a user of the API cannot keep up with every single update.
//
//   2. The Accumulator API is guaranteed to bootstrap (i.e. perform an initial List operation) on
//      all watches prior to notifying the user that resources are available to process.
type Client struct {
	config    *ConfigFlags
	cli       dynamic.Interface
	mapper    meta.RESTMapper
	disco     discovery.CachedDiscoveryInterface
	mutex     sync.Mutex
	canonical map[string]*Unstructured

	// This is an internal interface for testing, it lets us deliberately introduce delays into the
	// implementation, e.g. effectively increasing the latency to the api server in a controllable
	// way and letting us reproduce and test for race conditions far more efficiently than
	// otherwise.
	watchAdded   func(*Unstructured, *Unstructured)
	watchUpdated func(*Unstructured, *Unstructured)
	watchDeleted func(*Unstructured, *Unstructured)
}

// The ClientConfig struct holds all the parameters and configuration
// that can be passed upon construct of a new Client.
type ClientConfig struct {
	Kubeconfig string
	Context    string
	Namespace  string
}

// The NewClient function constructs a new client with the supplied ClientConfig.
func NewClient(options ClientConfig) (*Client, error) {
	return NewClientFromConfigFlags(options.toConfigFlags())
}

// NewClientFactory adds flags to a flagset (i.e. before flagset.Parse()), and returns a function to
// be called after flagset.Parse() that uses the parsed flags to construct a *Client.
func NewClientFactory(flags *pflag.FlagSet) func() (*Client, error) {
	if flags.Parsed() {
		// panic is OK because this is a programming error.
		panic("kates.NewClientFactory(flagset) must be called before flagset.Parse()")
	}

	config := NewConfigFlags(false)

	// We can disable or enable flags by setting them to
	// nil/non-nil prior to calling .AddFlags().
	//
	// .Username and .Password are already disabled by default in
	// genericclioptions.NewConfigFlags().

	config.AddFlags(flags)

	return func() (*Client, error) {
		if !flags.Parsed() {
			return nil, fmt.Errorf("kates client factory must be called after flagset.Parse()")
		}
		return NewClientFromConfigFlags(config)
	}
}

func NewClientFromConfigFlags(config *ConfigFlags) (*Client, error) {
	restconfig, err := config.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	cli, err := dynamic.NewForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	mapper, disco, err := NewRESTMapper(config)
	if err != nil {
		return nil, err
	}

	return &Client{
		config:       config,
		cli:          cli,
		mapper:       mapper,
		disco:        disco,
		canonical:    make(map[string]*Unstructured),
		watchAdded:   func(oldObj, newObj *Unstructured) {},
		watchUpdated: func(oldObj, newObj *Unstructured) {},
		watchDeleted: func(oldObj, newObj *Unstructured) {},
	}, nil
}

func NewRESTMapper(config *ConfigFlags) (meta.RESTMapper, discovery.CachedDiscoveryInterface, error) {
	// Throttling is scoped to rest.Config, so we use a dedicated
	// rest.Config for discovery so we can disable throttling for
	// discovery, but leave it in place for normal requests. This
	// is largely the same thing that ConfigFlags.ToRESTMapper()
	// does, hence the same thing that kubectl does. There are two
	// differences we are introducing here: (1) is that if there
	// is no cache dir supplied, we fallback to in-memory caching
	// rather than not caching discovery requests at all. The
	// second thing is that (2) unlike kubectl we do not cache
	// non-discovery requests.
	restconfig, err := config.ToRESTConfig()
	if err != nil {
		return nil, nil, err
	}
	restconfig.QPS = 1000000
	restconfig.Burst = 1000000

	var cachedDiscoveryClient discovery.CachedDiscoveryInterface
	if config.CacheDir != nil {
		cachedDiscoveryClient, err = disk.NewCachedDiscoveryClientForConfig(restconfig, *config.CacheDir, "",
			time.Duration(10*time.Minute))
		if err != nil {
			return nil, nil, err
		}
	} else {
		discoveryClient, err := discovery.NewDiscoveryClientForConfig(restconfig)
		if err != nil {
			return nil, nil, err
		}
		cachedDiscoveryClient = memory.NewMemCacheClient(discoveryClient)
	}

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedDiscoveryClient)
	expander := restmapper.NewShortcutExpander(mapper, cachedDiscoveryClient)

	return expander, cachedDiscoveryClient, nil
}

// The InCluster function returns true if the process is running inside a kubernetes cluster, and
// false if it is running outside the cluster. This is determined by heuristics, however it uses the
// exact same heuristics as client-go does. This is copied from
// (client-go/tools/clientcmd/client_config.go), as it is not publically invocable in its original
// place. This should be re-copied if the original code changes.
func InCluster() bool {
	fi, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token")
	return os.Getenv("KUBERNETES_SERVICE_HOST") != "" &&
		os.Getenv("KUBERNETES_SERVICE_PORT") != "" &&
		err == nil && !fi.IsDir()
}

// CurrentNamespace returns the namespace that is used if none is otherwise specified.
func (c *Client) CurrentNamespace() (string, error) {
	ns, _, err := c.config.ToRawKubeConfigLoader().Namespace()
	return ns, err
}

// DynamicInterface is an accessor method to the k8s dynamic client
func (c *Client) DynamicInterface() dynamic.Interface {
	return c.cli
}

func (c *Client) WaitFor(ctx context.Context, kindOrResource string) error {
	for {
		_, err := c.mappingFor(kindOrResource)
		if err != nil {
			_, ok := err.(*unknownResource)
			if ok {
				select {
				case <-time.After(1 * time.Second):
					if err := c.InvalidateCache(); err != nil {
						return err
					}
					continue
				case <-ctx.Done():
					return nil
				}
			}
		}
		return nil
	}
}

func (c *Client) InvalidateCache() error {
	// TODO: it's possible that invalidate could be smarter now
	// and use the methods on CachedDiscoveryInterface
	mapper, disco, err := NewRESTMapper(c.config)
	if err != nil {
		return err
	}
	c.mapper = mapper
	c.disco = disco
	return nil
}

// The ServerVersion() method returns a struct with information about
// the kubernetes api-server version.
func (c *Client) ServerVersion() (*VersionInfo, error) {
	return c.disco.ServerVersion()
}

// processAPIResourceLists takes a `[]*metav1.APIResourceList` as returned by any of several calls
// to a DiscoveryInterface, and transforms it in to a straight-forward `[]metav1.APIResource`.
//
// If you weren't paying close-enough attention, you might have thought I said it takes a
// `*metav1.APIResourceList` object, and now you're wondering why this needs to be anything more
// than `return input.APIResources`.  Well:
//
//   1. The various DiscoveryInterface calls don't return a List, they actually return an array of
//      Lists, where the Lists are grouped by the group/version of the resources.  So we need to
//      flatten those out.
//   2. I guess the reason they group them that way is to avoid repeating the group and version in
//      each resource, because the List objects themselvs have .Group and .Version set, but the
//      APIresource objects don't.  This lets them save 10s of bytes on an infrequently use API
//      call!  Anyway, we'll need to fill those in on the returned objects because we're discarding
//      the grouping.
func processAPIResourceLists(listsByGV []*metav1.APIResourceList) []APIResource {
	// Do some book-keeping to allow us to pre-allocate the entire list.
	count := 0
	for _, list := range listsByGV {
		if list != nil {
			count += len(list.APIResources)
		}
	}
	if count == 0 {
		return nil
	}

	// Build the processed list to return.
	ret := make([]APIResource, 0, count)
	for _, list := range listsByGV {
		if list != nil {
			gv, err := schema.ParseGroupVersion(list.GroupVersion)
			if err != nil {
				continue
			}
			for _, typeinfo := range list.APIResources {
				// I'm not 100% sure that none of the DiscoveryInterface calls fill
				// in .Group and .Version, so just in case one of the calls does
				// fill them in, we'll only fill them in if they're not already set.
				if typeinfo.Group == "" {
					typeinfo.Group = gv.Group
				}
				if typeinfo.Version == "" {
					typeinfo.Version = gv.Version
				}
				ret = append(ret, typeinfo)
			}
		}
	}

	return ret
}

// ServerPreferredResources returns the list of resource types that the server supports.
//
// If a resource type supports multiple versions, then *only* the preferred version is returned.
// Use ServerResources to return a list that includes all versions.
func (c *Client) ServerPreferredResources() ([]APIResource, error) {
	// It's possible that an error prevented listing some apigroups, but not all; so process the
	// output even if there is an error.
	listsByGV, err := c.disco.ServerPreferredResources()
	return processAPIResourceLists(listsByGV), err
}

// ServerResources returns the list of resource types that the server supports.
//
// If a resource type supports multiple versions, then a list entry for *each* version is returned.
// Use ServerPreferredResources to return a list that includes just the preferred version.
func (c *Client) ServerResources() ([]APIResource, error) {
	// It's possible that an error prevented listing some apigroups, but not all; so process the
	// output even if there is an error.
	_, listsByGV, err := c.disco.ServerGroupsAndResources()
	return processAPIResourceLists(listsByGV), err
}

// ==

// TODO: Query is interpreted a bit differently for List and
// Watch. Should either reconcile this or perhaps split Query into two
// separate types.

// A Query holds all the information needed to List or Watch a set of
// kubernetes resources.
type Query struct {
	// The Name field holds the name of the Query. This is used by
	// Watch to determine how multiple queries are unmarshaled by
	// Accumulator.Update(). This is ignored for List.
	Name string
	// The Kind field indicates what sort of resource is being queried.
	Kind string
	// The Namespace field holds the namespace to Query.
	Namespace string
	// The FieldSelector field holds a string in selector syntax
	// that is used to filter results based on field values. The
	// only field values supported are metadata.name and
	// metadata.namespace. This is only supported for List.
	FieldSelector string
	// The LabelSelector field holds a string in selector syntax
	// that is used to filter results based on label values.
	LabelSelector string
}

func (c *Client) Watch(ctx context.Context, queries ...Query) (*Accumulator, error) {
	return newAccumulator(ctx, c, queries...)
}

// ==

func (c *Client) watchRaw(ctx context.Context, query Query, target chan rawUpdate, cli dynamic.ResourceInterface) {
	var informer cache.SharedInformer

	// we override Watch to let us signal when our initial List is
	// complete so we can send an update() even when there are no
	// resource instances of the kind being watched
	lw := newListWatcher(ctx, cli, query, func(lw *lw) {
		if lw.hasSynced() {
			target <- rawUpdate{query.Name, true, nil, nil}
		}
	})
	informer = cache.NewSharedInformer(lw, &Unstructured{}, 5*time.Minute)
	// TODO: uncomment this when we get to kubernetes 1.19. Right now errors will get logged by
	// klog. With this error handler in place we will log them to our own logger and provide a
	// more useful error message:
	/*
		informer.SetWatchErrorHandler(func(r *cache.Reflector, err error) {
			// This is from client-go/tools/cache/reflector.go:563
			isExpiredError := func(err error) bool {
				// In Kubernetes 1.17 and earlier, the api server returns both apierrors.StatusReasonExpired and
				// apierrors.StatusReasonGone for HTTP 410 (Gone) status code responses. In 1.18 the kube server is more consistent
				// and always returns apierrors.StatusReasonExpired. For backward compatibility we can only remove the apierrors.IsGone
				// check when we fully drop support for Kubernetes 1.17 servers from reflectors.
				return apierrors.IsResourceExpired(err) || apierrors.IsGone(err)
			}

			switch {
			case isExpiredError(err):
				log.Printf("Watch of %s closed with: %v", query.Kind, err)
			case err == io.EOF:
				// watch closed normally
			case err == io.ErrUnexpectedEOF:
				log.Printf("Watch for %s closed with unexpected EOF: %v", query.Kind, err)
			default:
				log.Printf("Failed to watch %s: %v", query.Kind, err)
			}
		})
	*/
	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				// This is for testing. It allows us to deliberately increase the probability of
				// race conditions by e.g. introducing sleeps. At some point I'm sure we will want a
				// nicer prettier set of hooks, but for now all we need is this hack for
				// better/faster tests.
				c.watchAdded(nil, obj.(*Unstructured))
				lw.countAddEvent()
				target <- rawUpdate{query.Name, lw.hasSynced(), nil, obj.(*Unstructured)}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				old := oldObj.(*Unstructured)
				new := newObj.(*Unstructured)

				// This is for testing. It allows us to deliberately increase the probability of
				// race conditions by e.g. introducing sleeps. At some point I'm sure we will want a
				// nicer prettier set of hooks, but for now all we need is this hack for
				// better/faster tests.
				c.watchUpdated(old, new)
				target <- rawUpdate{query.Name, lw.hasSynced(), old, new}
			},
			DeleteFunc: func(obj interface{}) {
				var old *Unstructured
				switch o := obj.(type) {
				case cache.DeletedFinalStateUnknown:
					old = o.Obj.(*Unstructured)
				case *Unstructured:
					old = o
				}

				// This is for testing. It allows us to deliberately increase the probability of
				// race conditions by e.g. introducing sleeps. At some point I'm sure we will want a
				// nicer prettier set of hooks, but for now all we need is this hack for
				// better/faster tests.
				c.watchDeleted(old, nil)

				key := unKey(old)
				// For the Add and Update cases, we clean out c.canonical in
				// patchWatch.
				c.mutex.Lock()
				delete(c.canonical, key)
				c.mutex.Unlock()
				target <- rawUpdate{query.Name, lw.hasSynced(), old, nil}
			},
		},
	)

	go informer.Run(ctx.Done())
}

type rawUpdate struct {
	name   string
	synced bool
	old    *unstructured.Unstructured
	new    *unstructured.Unstructured
}

type lw struct {
	// All these fields are read-only and initialized on construction.
	ctx    context.Context
	client dynamic.ResourceInterface
	query  Query
	synced func(*lw)
	once   sync.Once

	// The mutex protects all the read-write fields.
	mutex            sync.Mutex
	initialListDone  bool
	initialListCount int
	addEventCount    int
	listForbidden    bool
}

func newListWatcher(ctx context.Context, client dynamic.ResourceInterface, query Query, synced func(*lw)) *lw {
	return &lw{ctx: ctx, client: client, query: query, synced: synced}
}

func (lw *lw) withMutex(f func()) {
	lw.mutex.Lock()
	defer lw.mutex.Unlock()
	f()
}

func (lw *lw) countAddEvent() {
	lw.withMutex(func() {
		lw.addEventCount++
	})
}

// This computes whether we have synced a given watch. We used to use SharedInformer.HasSynced() for
// this, but that seems to be a blatant lie that always return true. My best guess as to why it lies
// is that it is actually reporting the synced state of an internal queue, but because the
// SharedInformer mechanism adds another layer of dispatch on top of that internal queue, the
// syncedness of that internal queue is irrelevant to whether enough layered events have been
// dispatched to consider things synced at the dispatch layer.
//
// So to track syncedness properly for our users, when we do our first List() we remember how many
// resourcees there are and we do not consider ourselves synced until we have dispatched at least as
// many Add events as there are resources.
func (lw *lw) hasSynced() (result bool) {
	lw.withMutex(func() {
		result = lw.initialListDone && lw.addEventCount >= lw.initialListCount
	})
	return
}

// List is used by a SharedInformer to get a baseline list of resources
// that can then be maintained by a watch.
func (lw *lw) List(opts ListOptions) (runtime.Object, error) {
	// Our SharedInformer will call us every so often. Every time through,
	// we'll decide whether we can be synchronized, and whether the list was
	// forbidden.
	synced := false
	forbidden := false

	opts.FieldSelector = lw.query.FieldSelector
	opts.LabelSelector = lw.query.LabelSelector
	result, err := lw.client.List(lw.ctx, opts)

	if err == nil {
		// No error, the list worked out fine. We can be synced now...
		synced = true
		// ...and the list was not forbidden.
		forbidden = false
	} else if apierrors.IsForbidden(err) {
		// Forbidden. We'll still consider ourselves synchronized, but
		// remember the forbidden error!
		// dlog.Debugf(lw.ctx, "couldn't list %s (forbidden)", lw.query.Kind)
		synced = true
		forbidden = true

		// Impedance matching for the SharedInformer interface: pretend
		// that we got an empty list and no error.
		result = &unstructured.UnstructuredList{}
		err = nil
	} else {
		// Any other error we'll consider transient, and try again later.
		// We're neither synced nor forbidden
		dlog.Infof(lw.ctx, "couldn't list %s (will retry): %s", lw.query.Kind, err)
	}

	lw.withMutex(func() {
		if synced {
			if !lw.initialListDone {
				lw.initialListDone = true
				lw.initialListCount = len(result.Items)
			}
		}

		lw.listForbidden = forbidden
	})

	return result, err
}

func (lw *lw) Watch(opts ListOptions) (watch.Interface, error) {
	lw.once.Do(func() { lw.synced(lw) })
	opts.FieldSelector = lw.query.FieldSelector
	opts.LabelSelector = lw.query.LabelSelector

	iface, err := lw.client.Watch(lw.ctx, opts)

	if err != nil {
		// If the list was forbidden, this error will likely just be "unknown", since we
		// returned an unstructured.UnstructuredList to fake out the lister, so in that
		// case just synthesize a slightly nicer error.
		if lw.listForbidden {
			err = errors.New(fmt.Sprintf("can't watch %s: forbidden", lw.query.Kind))
		} else {
			// Not forbidden. Go ahead and make sure the Kind we're querying for is in
			// there, though.
			err = errors.Wrap(err, fmt.Sprintf("can't watch %s", lw.query.Kind))
		}
	}

	return iface, err
}

// ==

// IsNamespaced returns whether a (fully-qualified) GVK is namespaced.
func (c *Client) IsNamespaced(gvk GroupVersionKind) (bool, error) {
	mapping, err := c.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return false, err
	}
	return mapping.Scope.Name() == meta.RESTScopeNameNamespace, nil
}

func (c *Client) cliFor(mapping *meta.RESTMapping, namespace string) dynamic.ResourceInterface {
	cli := c.cli.Resource(mapping.Resource)
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace && namespace != NamespaceAll {
		return cli.Namespace(namespace)
	} else {
		return cli
	}
}

func (c *Client) cliForResource(resource *Unstructured) (dynamic.ResourceInterface, error) {
	mapping, err := c.mappingFor(resource.GroupVersionKind().GroupKind().String())
	if err != nil {
		return nil, err
	}

	// this will canonicalize the kind and version so any
	// shortcuts are expanded
	resource.SetGroupVersionKind(mapping.GroupVersionKind)

	ns := resource.GetNamespace()
	if ns == "" {
		ns = "default"
	}
	return c.cliFor(mapping, ns), nil
}

func (c *Client) newField(q Query) (*field, error) {
	mapping, err := c.mappingFor(q.Kind)
	if err != nil {
		return nil, err
	}
	sel, err := ParseSelector(q.LabelSelector)
	if err != nil {
		return nil, err
	}

	return &field{
		query:    q,
		mapping:  mapping,
		selector: sel,
		values:   make(map[string]*Unstructured),
		deltas:   make(map[string]*Delta),
	}, nil
}

// mappingFor returns the RESTMapping for the Kind given, or the Kind referenced by the resource.
// Prefers a fully specified GroupVersionResource match. If one is not found, we match on a fully
// specified GroupVersionKind, or fallback to a match on GroupKind.
//
// This is copy/pasted from k8s.io/cli-runtime/pkg/resource.Builder.mappingFor() (which is
// unfortunately private), with modified lines marked with "// MODIFIED".
func (c *Client) mappingFor(resourceOrKind string) (*meta.RESTMapping, error) { // MODIFIED: args
	fullySpecifiedGVR, groupResource := schema.ParseResourceArg(resourceOrKind)
	gvk := schema.GroupVersionKind{}
	// MODIFIED: Don't call b.restMapperFn(), use c.mapper instead.

	if fullySpecifiedGVR != nil {
		gvk, _ = c.mapper.KindFor(*fullySpecifiedGVR)
	}
	if gvk.Empty() {
		gvk, _ = c.mapper.KindFor(groupResource.WithVersion(""))
	}
	if !gvk.Empty() {
		return c.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	}

	fullySpecifiedGVK, groupKind := schema.ParseKindArg(resourceOrKind)
	if fullySpecifiedGVK == nil {
		gvk := groupKind.WithVersion("")
		fullySpecifiedGVK = &gvk
	}

	if !fullySpecifiedGVK.Empty() {
		if mapping, err := c.mapper.RESTMapping(fullySpecifiedGVK.GroupKind(), fullySpecifiedGVK.Version); err == nil {
			return mapping, nil
		}
	}

	mapping, err := c.mapper.RESTMapping(groupKind, gvk.Version)
	if err != nil {
		// if we error out here, it is because we could not match a resource or a kind
		// for the given argument. To maintain consistency with previous behavior,
		// announce that a resource type could not be found.
		// if the error is _not_ a *meta.NoKindMatchError, then we had trouble doing discovery,
		// so we should return the original error since it may help a user diagnose what is actually wrong
		if meta.IsNoMatchError(err) {
			return nil, &unknownResource{resourceOrKind}
		}
		return nil, err
	}

	return mapping, nil
}

type unknownResource struct {
	arg string
}

func (e *unknownResource) Error() string {
	return fmt.Sprintf("the server doesn't have a resource type %q", e.arg)
}

// ==

func (c *Client) List(ctx context.Context, query Query, target interface{}) error {
	mapping, err := c.mappingFor(query.Kind)
	if err != nil {
		return err
	}

	items := make([]*Unstructured, 0)
	if err := func() error {
		c.mutex.Lock()
		defer c.mutex.Unlock()
		cli := c.cliFor(mapping, query.Namespace)
		res, err := cli.List(ctx, ListOptions{
			FieldSelector: query.FieldSelector,
			LabelSelector: query.LabelSelector,
		})
		if err != nil {
			return err
		}

		for _, un := range res.Items {
			copy := un.DeepCopy()
			key := unKey(copy)
			// TODO: Deal with garbage collection in the case
			// where there is no watch.
			c.canonical[key] = copy
			items = append(items, copy)
		}
		return nil
	}(); err != nil {
		return err
	}

	return convert(items, target)
}

// ==

func (c *Client) Get(ctx context.Context, resource interface{}, target interface{}) error {
	var un Unstructured
	err := convert(resource, &un)
	if err != nil {
		return err
	}

	var res *Unstructured
	if err := func() error {
		c.mutex.Lock()
		defer c.mutex.Unlock()
		cli, err := c.cliForResource(&un)
		if err != nil {
			return err
		}
		res, err = cli.Get(ctx, un.GetName(), GetOptions{})
		if err != nil {
			return err
		}
		key := unKey(res)
		// TODO: Deal with garbage collection in the case
		// where there is no watch.
		c.canonical[key] = res
		return nil
	}(); err != nil {
		return err
	}

	return convert(res, target)
}

// ==

func (c *Client) Create(ctx context.Context, resource interface{}, target interface{}) error {
	var un Unstructured
	err := convert(resource, &un)
	if err != nil {
		return err
	}

	var res *Unstructured
	if err := func() error {
		c.mutex.Lock()
		defer c.mutex.Unlock()
		cli, err := c.cliForResource(&un)
		if err != nil {
			return err
		}
		res, err = cli.Create(ctx, &un, CreateOptions{})
		if err != nil {
			return err
		}
		key := unKey(res)
		c.canonical[key] = res
		return nil
	}(); err != nil {
		return err
	}

	return convert(res, target)
}

// ==

func (c *Client) Update(ctx context.Context, resource interface{}, target interface{}) error {
	var un Unstructured
	err := convert(resource, &un)
	if err != nil {
		return err
	}

	prev := un.GetResourceVersion()

	var res *Unstructured
	if err := func() error {
		c.mutex.Lock()
		defer c.mutex.Unlock()
		cli, err := c.cliForResource(&un)
		if err != nil {
			return err
		}
		res, err = cli.Update(ctx, &un, UpdateOptions{})
		if err != nil {
			return err
		}
		if res.GetResourceVersion() != prev {
			key := unKey(res)
			c.canonical[key] = res
		}
		return nil
	}(); err != nil {
		return err
	}

	return convert(res, target)
}

// ==

func (c *Client) Patch(ctx context.Context, resource interface{}, pt PatchType, data []byte, target interface{}) error {
	var un Unstructured
	err := convert(resource, &un)
	if err != nil {
		return err
	}

	prev := un.GetResourceVersion()

	var res *Unstructured
	if err := func() error {
		c.mutex.Lock()
		defer c.mutex.Unlock()
		cli, err := c.cliForResource(&un)
		if err != nil {
			return err
		}
		res, err = cli.Patch(ctx, un.GetName(), pt, data, PatchOptions{})
		if err != nil {
			return err
		}
		if res.GetResourceVersion() != prev {
			key := unKey(res)
			c.canonical[key] = res
		}
		return nil
	}(); err != nil {
		return err
	}

	return convert(res, target)
}

// ==

func (c *Client) Upsert(ctx context.Context, resource interface{}, source interface{}, target interface{}) error {
	if resource == nil || reflect.ValueOf(resource).IsNil() {
		resource = source
	}

	var un Unstructured
	err := convert(resource, &un)
	if err != nil {
		return err
	}

	var unsrc Unstructured
	err = convert(source, &unsrc)
	if err != nil {
		return err
	}
	MergeUpdate(&un, &unsrc)

	prev := un.GetResourceVersion()

	var res *Unstructured
	if err := func() error {
		c.mutex.Lock()
		defer c.mutex.Unlock()
		cli, err := c.cliForResource(&un)
		if err != nil {
			return err
		}
		create := false
		rsrc := &un
		if prev == "" {
			stored, err := cli.Get(ctx, un.GetName(), GetOptions{})
			if err != nil {
				if IsNotFound(err) {
					create = true
					rsrc = &un
				} else {
					return err
				}
			} else {
				rsrc = stored
				MergeUpdate(rsrc, &unsrc)
			}
		}
		if create {
			res, err = cli.Create(ctx, rsrc, CreateOptions{})
		} else {
			// XXX: need to clean up the conflict case and add a test for it
		update:
			res, err = cli.Update(ctx, rsrc, UpdateOptions{})
			if err != nil && IsConflict(err) {
				stored, err := cli.Get(ctx, un.GetName(), GetOptions{})
				if err != nil {
					return err
				}
				rsrc = stored
				MergeUpdate(rsrc, &unsrc)
				goto update
			}
		}
		if err != nil {
			return err
		}
		if res.GetResourceVersion() != prev {
			key := unKey(res)
			c.canonical[key] = res
		}
		return nil
	}(); err != nil {
		return err
	}

	return convert(res, target)
}

// ==

func (c *Client) UpdateStatus(ctx context.Context, resource interface{}, target interface{}) error {
	var un Unstructured
	err := convert(resource, &un)
	if err != nil {
		return err
	}

	prev := un.GetResourceVersion()

	var res *Unstructured
	if err := func() error {
		c.mutex.Lock()
		defer c.mutex.Unlock()
		cli, err := c.cliForResource(&un)
		if err != nil {
			return err
		}
		res, err = cli.UpdateStatus(ctx, &un, UpdateOptions{})
		if err != nil {
			return err
		}
		if res.GetResourceVersion() != prev {
			key := unKey(res)
			c.canonical[key] = res
		}
		return nil
	}(); err != nil {
		return err
	}

	return convert(res, target)
}

// ==

func (c *Client) Delete(ctx context.Context, resource interface{}, target interface{}) error {
	var un Unstructured
	err := convert(resource, &un)
	if err != nil {
		return err
	}

	if err := func() error {
		c.mutex.Lock()
		defer c.mutex.Unlock()
		cli, err := c.cliForResource(&un)
		if err != nil {
			return err
		}
		err = cli.Delete(ctx, un.GetName(), DeleteOptions{})
		if err != nil {
			return err
		}
		key := unKey(&un)
		c.canonical[key] = nil
		return nil
	}(); err != nil {
		return err
	}

	return convert(nil, target)
}

// ==

// Update the result of a watch with newer items from our local cache. This guarantees we never give
// back stale objects that are known to be modified by us.
func (c *Client) patchWatch(ctx context.Context, field *field) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// The canonical map holds all local changes made by this client which have not been reflected
	// back to it through a watch. This should normally be quite small since objects only occupy
	// this map for the duration of a round trip to the API server. (XXX: We don't yet handle the
	// case of modifying objects that are not watched. Those objects will get stuck in this map, but
	// that is ok for our current set of use cases.)

	// Loop over the canonical map and patch the watch result.
	for key, can := range c.canonical {
		item, ok := field.values[key]
		if ok {
			// An object is both in our canonical map and in the watch.
			if can == nil {
				// The object is deleted, but has not yet been reported so by the apiserver, so we
				// remove it.
				dlog.Println(ctx, "Patching delete", field.mapping.GroupVersionKind.Kind, key)
				delete(field.values, key)
				field.deltas[key] = newDelta(ObjectDelete, item)
			} else if newer, err := gteq(item.GetResourceVersion(), can.GetResourceVersion()); err != nil {
				return err
			} else if newer {
				// The object in the watch result is the same or newer than our canonical value, so
				// no need to track it anymore.
				dlog.Println(ctx, "Patching synced", field.mapping.GroupVersionKind.Kind, key)
				delete(c.canonical, key)
			} else {
				// The object in the watch result is stale, so we update it with the canonical
				// version and track it as a delta.
				dlog.Println(ctx, "Patching update", field.mapping.GroupVersionKind.Kind, key)
				field.values[key] = can
				field.deltas[key] = newDelta(ObjectUpdate, can)
			}
		} else if can != nil && can.GroupVersionKind() == field.mapping.GroupVersionKind &&
			field.selector.Matches(LabelSet(can.GetLabels())) {
			// An object that was created locally is not yet present in the watch result, so we add it.
			dlog.Println(ctx, "Patching add", field.mapping.GroupVersionKind.Kind, key)
			field.values[key] = can
			field.deltas[key] = newDelta(ObjectAdd, can)
		}
	}
	return nil
}

// ==

// The LogEvent struct is used to communicate log output from a pod. It includes PodID and Timestamp information so that
// LogEvents from different pods can be interleaved without losing information about total ordering and/or pod identity.
type LogEvent struct {
	// The PodID field indicates what pod the log output is associated with.
	PodID string `json:"podID"`
	// The Timestamp field indicates when the log output was created.
	Timestamp string `json:"timestamp"`

	// The Output field contains log output from the pod.
	Output string `json:"output,omitempty"`

	// The Closed field is true if the supply of log events from the given pod was terminated. This does not
	// necessarily mean there is no more log data.
	Closed bool
	// The Error field contains error information if the log events were terminated due to an error.
	Error error `json:"error,omitempty"`
}

func parseLogLine(line string) (timestamp string, output string) {
	if parts := strings.SplitN(line, " ", 2); len(parts) == 2 {
		if _, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
			timestamp = parts[0]
			output = parts[1]
			return
		}
	}
	output = line
	return
}

// The PodLogs method accesses the log output of a given pod by sending LogEvent structs down the supplied channel. The
// LogEvent struct is designed to hold enough information that it is feasible to invoke PodLogs multiple times with a
// single channel in order to multiplex log output from many pods, e.g.:
//
//   events := make(chan LogEvent)
//   client.PodLogs(ctx, pod1, options, events)
//   client.PodLogs(ctx, pod2, options, events)
//   client.PodLogs(ctx, pod3, options, events)
//
//   for event := range events {
//       fmt.Printf("%s: %s: %s", event.PodId, event.Timestamp, event.Output)
//   }
//
// The above code will print log output from all 3 pods.
func (c *Client) PodLogs(ctx context.Context, pod *Pod, options *PodLogOptions, events chan<- LogEvent) error {
	// always use timestamps
	options.Timestamps = true
	timeout := 10 * time.Second
	allContainers := true

	requests, err := polymorphichelpers.LogsForObjectFn(c.config, pod, options, timeout,
		allContainers)
	if err != nil {
		return err
	}

	podID := string(pod.GetUID())
	for _, request := range requests {
		go func(request rest.ResponseWrapper) {
			readCloser, err := request.Stream(ctx)
			if err != nil {
				events <- LogEvent{PodID: podID, Error: err, Closed: true}
				return
			}
			defer readCloser.Close()

			r := bufio.NewReader(readCloser)
			for {
				bytes, err := r.ReadBytes('\n')
				if len(bytes) > 0 {
					timestamp, output := parseLogLine(string(bytes))
					events <- LogEvent{
						PodID:     podID,
						Timestamp: timestamp,
						Output:    output,
					}
				}
				if err != nil {
					if err != io.EOF {
						events <- LogEvent{
							PodID:  podID,
							Error:  err,
							Closed: true,
						}
					} else {
						events <- LogEvent{PodID: podID, Closed: true}
					}
					return
				}
			}
		}(request)
	}

	return nil
}

// Technically this is sketchy since resource versions are opaque, however this exact same approach
// is also taken deep in the bowels of client-go and from what I understand of the k3s folk's
// efforts replacing etcd (the source of these resource versions) with a different store, the
// kubernetes team was very adamant about the approach to pluggable stores being to create an etcd
// shim rather than to go more abstract. I believe this makes it relatively safe to depend on in
// practice.
func gteq(v1, v2 string) (bool, error) {
	i1, err := strconv.ParseInt(v1, 10, 64)
	if err != nil {
		return false, err
	}
	i2, err := strconv.ParseInt(v2, 10, 64)
	if err != nil {
		return false, err
	}
	return i1 >= i2, nil
}

func convert(in interface{}, out interface{}) error {
	return kates_internal.Convert(in, out)
}

func unKey(u *Unstructured) string {
	return string(u.GetUID())
}

func (options ClientConfig) toConfigFlags() *ConfigFlags {
	result := NewConfigFlags(false)

	if options.Kubeconfig != "" {
		result.KubeConfig = &options.Kubeconfig
	}
	if options.Context != "" {
		result.Context = &options.Context
	}
	if options.Namespace != "" {
		result.Namespace = &options.Namespace
	}

	return result
}
