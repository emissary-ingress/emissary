/**
 * IResource
 * This is the Resource interface class that defines the methods that any Resource subclass must implement
 * as well as listing all the superclass methods that a subclass can utilize.
 *
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
 */

import { Resource } from "../framework/resource.js"

export class IResource extends Resource {

  /* ====================================================================================================
   *  These methods must be implemented by subclasses.
   * ====================================================================================================
   */

  /**
   * defaultYaml
   *
   * Supply the default yaml for a newly created resource.
   */
  static get defaultYaml() {
    return Resource.defaultYaml
  }

  /* validateSelf()
   * Validate this Resource's state by checking each object instance variable for correctness (e.g. email address
   * format, URL format, date/time, name restrictions).  Returns a dictionary of property: errorString if there
   * are any errors. If the dictionary is empty, there are no errors.
   */
  validateSelf() {
    throw new Error("Please implement ${this.constructor.name}:validateSelf()");
  }

}
