/**
 * HostCollection
 * This is a Collection subclass for Host objects.  It simply defines the following methods:
 * - resourceClass() => return the Host class
 * - extractDataFrom(snapshot) => return a list of data objects representing Hosts for instantiation.
 *
 * * Everything else is implemented in Collection.
 */

import { HostResource }  from "./host_resource.js"
import { ICollection }   from "../interfaces/icollection.js";

export class HostCollection extends ICollection {
  /* constructor()
  * call Collection's constructor.
  */
  constructor() {
    super();
  }

  /* extractResourcesFrom(snapshot)
   * Given a snapshot as received from the backend via snapshot.js, return a list of resource data blocks
   * given the resource's type name, Host in this case.  Since the snapshot itself has a method to get resource
   * data for given types, we can simply call the snapshot to return the list of Host data blocks.
   */

  extractResourcesFrom(snapshot) {
    return snapshot.getResources("Host");
  }

  /* resourceClass()
  * Return the class of the resource that is being collected from the snapshot.
  */

  resourceClass() {
    return HostResource;
  }

  /* uniqueKeyFor(resourceData)
   * Return a computed modelKey given some structured resource data (a hierarchical key/value
   * structure). Here we use the Resource kind, name, and namespace concatenated together for a unique key.
   */

  uniqueKeyFor(resourceData) {
    return resourceData.kind + "::" + resourceData.metadata.name + "::" + resourceData.metadata.namespace;
  }
 }


/* Export a HostCollection instance.  This object manages every Host instance and synchronizes the list of Hosts
 * that are instantiated with the real world of Kubernetes.
 */

export var AllHosts = new HostCollection();
