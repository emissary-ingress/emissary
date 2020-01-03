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
}

