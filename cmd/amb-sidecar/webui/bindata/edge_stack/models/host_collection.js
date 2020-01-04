/**
 * HostCollection
 * This is a Collection subclass for Host objects.  It simply defines the following method:
 * - resourceClass() => return the Host class
 *
 * * Everything else is implemented in Collection.
 */

import {Host}       from "./host.js"
import {Collection} from "./collection.js";

export class HostCollection extends Collection {
  /* constructor()
  * call Collection's constructor.
  */
  constructor() {
    super();
  }

  /* resourceClass()
  * Return the class of the resource that is being collected from the snapshot.
  */

  resourceClass() {
    return Host;
  }

  /* extractDataFrom(snapshot)
  * Given a snapshot as received from the backend via snapshot.js, return a list of resource data blocks
  * given the resource's class name (e.g. Host, Mapping, Filter...).  Since this is a Collection superclass
  * from which other Collection classes will inherit, they must implement their own extracDataFrom methods.
  */

  extractDataFrom(snapshot) {
    return snapshot.getResources("Host");
  }
}


/* Export a Collection for Hosts.  This object manages every Host instance and synchronizes the list of Hosts
 * that are instantiated with the real world of Kubernetes.
 */

export var AllHosts = new HostCollection();
