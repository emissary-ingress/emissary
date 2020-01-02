/**
 * IResource
 * This is the Resource interface class that defines the methods that any Resource subclass should implement.
 * A Resource is a Model that maintains basic Kubernetes resource state and implements code to create
 * instances of that Resource from snapshot data.
 *
 * A Kubernetes Resource (also known as an Object) is a persistent entity in the Kubernetes system that is
 * used to represent the desired state of your cluster.  Kubernetes uses Resources to apply policies and
 * procedures to ensure that the cluster reaches that desired state, by allocating, deallocating, configuring
 * and running compute and networking entities.
 *
 * The Kubernetes documentation makes a distinction between a Kubernetes Object 9the persistent data entities
 * that are stored in the etcd database) and a Resource (an endpoint in the Kubernetes APO that stores a
 * collection of objects).  However, this distinction is not made consistently throughout the documentation.
 * Here we will use the term Resource to indicate a chunk of data--a kind, name, namespace, metadata, and
 * a spec--that we display and modify in the Web user interface.
 *
 * This class is the generic interface for all Kubernetes resource that are created, viewed, modified
 * and deleted in the Web UI.
 */

export class IResource {

  /* constructor()
   * Here the model initializes any internal state that is common to all Resources.
   * Typically a concrete Resource class would initialize the Resource kind, name, namespace,
   * and other useful state to be maintained in the Resource instance.
   */

  constructor(kind, name, namespace) {
    /* do nothing for now, concrete subclasses will implement, but subclasses must call
     * super(kind, name, namespace) in their own constructor.
    */
  }

  /* resourceKey()
   * Model instances are typically created from a chunk of Kubernetes resource data, which is
   * a hierarchical key/value structure (JSON or dictionary).  To determine whether a particular
   * Model corresponds to an existing Kubernetes resource, there must be a key that is invariant
   * for that particular Resource and its Kubernetes data.  This is needed to maintain a collection
   * of Resources that map 1-1 to objects in the Kubernetes resource space.
   */

  resourceKey() {
    throw new Error("Please implement Resource:resourceKey()")
  }

  /* static resourceKeyFor(data)
   * Return a computed modelKey given some structured data (as above, a hierarchical key/value
   * structure).  This is a static function that is given the data from a snapshot and returns
   * the model key for that data.  Each Model subclass will know the structure and extract
   * the appropriate information to create the Model's key.  This is needed for identity in a
   * collection of Models.  It is a static function because a given Model may not yet exist in
   * the collection and its key must be created from the raw data.
   */

  static resourceKeyFor(data) {
    throw new Error("please implement Resource:resourceKeyFor(data) if this Resource is part of a Collection");
  }

  /* resourceExtractor(snapshot)
  * Return a list of resources from the snapshot, given the resource's class name (e.g. Host, Mapping,
  * Filter, ...)
  */

  static resourceExtractor(snapshot) {
    throw new Error("please implement resourceExtractor()")
  }
}

