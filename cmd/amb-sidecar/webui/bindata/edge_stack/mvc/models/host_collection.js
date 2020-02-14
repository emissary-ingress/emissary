/**
 * HostCollection
 * This is an IResourceCollection subclass for Host objects.
 */

import { HostResource }          from "./host_resource.js"
import { IResourceCollection }   from "../interfaces/iresource_collection.js";

export class HostCollection extends IResourceCollection {
  /* extend */
  constructor() {
    super();
  }

  /* override */
  extractResourcesFrom(snapshot) {
    return snapshot.getResources("Host");
  }

  /* override */
  resourceClass() {
    return HostResource;
  }
}


/* The AllHosts object manages every Host instance and synchronizes the list of Hosts
 * that are instantiated with the real world of Kubernetes.
 */
export var AllHosts = new HostCollection();
