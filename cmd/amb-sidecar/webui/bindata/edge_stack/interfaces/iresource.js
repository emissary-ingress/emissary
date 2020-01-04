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

  /* constructor() */

  constructor(data) {
    /* call Resource's constructor */
    super(data);
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
    throw new Error("please implement Resource:uniqueKeyFor(data)");
  }

   /* updateSelfFrom(data)
   * Update the Resource object state from the snapshot data block for this Resource.  Compare the values in the
   * data block with the stored state in the Resource.  If the data block has different data than is currently
   * stored, update that instance variable with the new data and set a flag to return true if any changes have
   * occurred.  The Resource class's method, updateFrom, will call this method and then notify listeners as needed.
   */

  updateSelfFrom(data) {
    throw new Error("Please implement Resource:updateSelfFrom(data)");
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

  /* validateSelf()
   * Validate this Resource's state by checking each object instance variable for correctness (e.g. email address
   * format, URL format, date/time, name restrictions).  Returns a dictionary of property: errorString if there
   * are any errors. If the dictionary is empty, there are no errors.
   */

  validateSelf() {
    throw new Error("Please implement Resource:validateSelf()");
  }
}

