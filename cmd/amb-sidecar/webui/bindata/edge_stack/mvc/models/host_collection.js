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

  /* uniqueKeyFor(data)
   * Return a computed modelKey given some structured data (a hierarchical key/value
   * structure).  This is a method that is given the data block from a snapshot and returns
   * the unique key for that data.  Each Collection subclass will know the structure and extract
   * the appropriate information to create the Resource's key.  This is needed for identity in a
   * collection of Resources.
   *
   * Here we use the Resource kind, name, and namespace concatenated together for a unique key.
   */

  uniqueKeyFor(data) {
    return data.kind + "::" + data.metadata.name + "::" + data.metadata.namespace;
  }

  /* resourceClass()
  * Return the class of the resource that is being collected from the snapshot.
  */

  resourceClass() {
    return HostResource;
  }

  /* extractDataFrom(snapshot)
  * Given a snapshot as received from the backend via snapshot.js, return a list of resource data blocks
  * given the resource's class name (e.g. HostResource...).  Since this is a Collection superclass
  * from which other Collection classes will inherit, they must implement their own extracDataFrom methods.
  */

  extractDataFrom(snapshot) {
    return snapshot.getResources("Host");
  }
}


/* Export a HostCollection instance.  This object manages every Host instance and synchronizes the list of Hosts
 * that are instantiated with the real world of Kubernetes.
 */

export var AllHosts = new HostCollection();
