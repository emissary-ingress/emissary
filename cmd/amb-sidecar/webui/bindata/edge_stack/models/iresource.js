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
 * and deleted in the Web UI.  For example implementations, see the Host class, which is a concrete implementation
 * of the IResource interface.
 */

/* Interface class for Model */
import { Resource } from "./resource.js"

export class IResource extends Resource {

  /* constructor()
   * Here the model initializes any internal state that is common to all Resources.
   * Typically a concrete Resource class would initialize the Resource kind, name, namespace,
   * and other useful state to be maintained in the Resource instance.
   */

  constructor(data) {
    /* do nothing for now, concrete subclasses will implement, but subclasses must call
     * super(data) in their own constructor.
    */

    /* call IModel's constructor, which is a no-op as well. */
    super();
  }

  /* static uniqueKeyFor(data)
   * Return a computed modelKey given some structured data (a hierarchical key/value
   * structure).  This is a static function that is given the data block from a snapshot and returns
   * the model key for that data.  Each Resource subclass will know the structure and extract
   * the appropriate information to create the Resource's key.  This is needed for identity in a
   * collection of Resources.  It is a static function because a given Resource may not yet exist in
   * the collection and its key must be created from the raw data.
   */

  static uniqueKeyFor(data) {
    throw new Error("please implement Resource:uniqueKeyFor(data) if this Resource is part of a Collection");
  }

   /* updateFrom(data)
   * Update the Resource object state from the snapshot data block for this Resource.  Compare the values in the
   * data block with the stored state in the Resource.  If the data block has different data than is currently
   * stored, update that instance variable with the new data and set a flag to notify listeners of the changed
   * state once the Resource has been fully updated.
   */

  updateFrom(data) {
    throw new Error("Please implement Resource:updateFrom(data)");
  }

  /* getEmptyStatus()
   * Utility method for initializing the status of the resource.  Returns a dictionary that has the basic
   * structure of the status attribute in the Kubernetes resource structure.
   */

  getEmptyStatus() {
    throw new Error("Please implement Resource:getEmptyStatus()");
  }

  /* getSpec()
   * Return the spec attribute of the Resource.  This method is needed for the implementation of the Save
   * function which uses kubectl apply.  This method must return an object that will be serialized with JSON.stringify
   * and supplied as the "spec:" portion of the Kubernetes YAML that is passed to kubectl.  See the Host class for
   * an example implementation.
   */

  getSpec() {
    throw new Error("Please implement Resource:getSpec()");
  }

  /* getYAML()
  * Return the YAML object to JSON.stringify for the implementation of the Save function which uses kubectl apply.
  * Like getSpec, this method must return an object to be serialized and supplied to kubectl apply.  Note that this
  * likewise is only a partial YAML structure (getSpec being the spec: portion).  The full YAML for the resource
  * should be saved separately.  See the Host class for an example implementation.
  */
  getYAML() {
    throw new Error("Please implement Resource:getYAML()");
  }

  /* sourceURI()
   * Return the source URI for this resource, if one exists.  IN the case we have a source URI, the view may provide
   * a button which, when clicked, opens a window on that source URI.  This is useful for tracking resource as they
   * are applied using GitOps, though an annotation specifying the sourceURI must be applied in the GitOps pipeline
   * in order for this to have a value.  If there is no sourceURI, return undefined.
   */
  sourceURI() {
    throw new Error("Please implement Resource:sourceURI()");
  }

  /* validate()
   * Validate this Resource's state by checking each object instance variable for correctness (e.g. email address
   * format, URL format, date/time, name restrictions).  Returns a dictionary of property: errorString if there
   * are any errors. If the dictionary is empty, there are no errors.
   */

  validateSelf() {
    throw new Error("Please implement Resource:validateSelf()");
  }
}

