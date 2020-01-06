/**
 * ICollection
 * This is the Interface class for Collection.
 *
 * ICollection subclasses, such as HostCollection, need only define one method to specialize:
 * - resourceClass(), which returns the class that should be instantiated if a new item (e.g. Host) is added
 *
 * - uniqueKeyFor(data)
 *   this method computes a unique key for the collection's Resource, to determine whether the Resource described
 *   by the data block has already been initialized and stored in the Collection.
 *
 * - dataExtractor(snapshot)
 *   this method is responsible for finding the right place in the snapshot to extract data objects that are used
 *   by the Resource's initFrom method to create a new instance of that resource.  This function returns a
 *   list that can be iterated over, returning data objects.
 *
 * Most Resources (CRD's) have the same data formats but there are other objects in the snapshot that
 * are not CRD's, have different structure, and are not in the same part of the snapshot (e.g. Resolvers).  Similarly,
 * different Model classes will generate different unique keys, and so each will implement a class function
 * resourceKeyFor(data).
 *
 * Listeners will be notified when Models are added, updated, or removed from the collection.  The collection's
 * listeners are generally Views.
 */

import { Collection } from "../framework/collection.js";

export class ICollection extends Collection {
  /* constructor()
   * Simply calls Collection to initialize the object state.
   */
  constructor() {
    super();
  }

  /* resourceClass()
   * Return the class of the resource that is being czollected from the snapshot.
   */

  resourceClass() {
    throw new Error("Please implement Collection:resourceClass()");
  }

  /* uniqueKeyFor(resourceData)
   * Return a computed modelKey given some structured resource data (a hierarchical key/value
   * structure).  This is a method that is given the data block from a snapshot and returns
   * the unique key for that data.  Each Collection subclass will know the structure and extract
   * the appropriate information to create the Resource's key.  This is needed for identity in a
   * collection of Resources.
   */

  uniqueKeyFor(resourceData) {
    throw new Error("please implement Collection:uniqueKeyFor(resourceData)");
  }

  /* extractResourcesFrom(snapshot)
   * Given a snapshot as received from the backend via snapshot.js, return a list of resource data blocks
   * given the resource's class name (e.g. Host, Mapping, Filter...).  Since this is a Collection superclass
   * from which other Collection classes will inherit, they must implement their own extractResourcesFrom methods.
   */

  extractResourcesFrom(snapshot) {
    throw new Error("please implement Collection:extractResourcesFrom(snapshot)")
  }
}

