/**
 * ResourceCollection
 * This is a Model subclass that monitors the snapshot data and keeps a consistent set of Resource objects
 * that mirror the actual model data in the snapshot.
 *
 * The IResourceCollection interface class, which inherits from ResourceCollection, defines the three required methods
 * for creating specialized subclasses of ResourceCollection:
 *
 * resourceClass(), uniqueKeyFor(data), and extractDataFrom(snapshot).
 *
 * See IResourceCollection for further details.
 */

import { Model }    from "./model.js";
import { Snapshot } from "../../components/snapshot.js";

export class ResourceCollection extends Model {

  /* constructor()
   * Create a map to hold the collection of resources, and subscribe to Snapshot changes.
   */
  constructor() {
    super();

    /* Here's where we store all the real resources, where each key is the resourceKey for that resource. */
    this._resources = new Map();

    /* Here's our subscription to data changes from the backend so that we can update the set of all models. */
    Snapshot.subscribe(this.onSnapshotChange.bind(this));
  }

  /* onSnapshotChange(snapshot)
   * When new snapshot data is available, we need to create, delete, or update  models from our collection (set) of
   * models and notify the listeners of any changes.
   */

  onSnapshotChange(snapshot) {
    /* Save the keys of all our existing resources */
    let previousKeys = new Set(this._resources.keys());
    let resourceClass = this.resourceClass();

    /* For each of the snapshot data records for this model... */
    for (let yaml of this.extractResourcesFrom(snapshot)) {
      let key = this.uniqueKeyFor(yaml);
      /* ...if we already have a model object for this data, then ask
       * that object to check if it needs to update any data fields.
       */
      let existingResource = this._resources.get(key);
      if (existingResource) {
        /* Only need to update if the existing Resource's version has changed.  Note that resourceVersion can only
         * be compared with equality, and is not necessarily a monotonically increasing value.
         */
        if (existingResource.version !== yaml.metadata.resourceVersion) {
          existingResource.updateFrom(yaml);
        }

        /* Note that we've seen this resource, so delete this key from our set of initial object
        * keys.  If any keys are left at the end of the process, that means that the objects
        * with those keys were not observed in the snapshot and thus must be removed. */
        previousKeys.delete(key);
      }
      else {
        /* ...if we do not have a model object for this Resource (as defined by the unique key), then create a new
         * Resource object. After creating the object, notify all my listeners of the creation. See views/resources.js
         * for more information on how the ResourceListView uses that notification to add new child web components.
         */
        let newResource = new resourceClass(yaml);
        this._resources.set(key, newResource);
        this.notifyListenersCreated(newResource);
      }
    }

    /*
     * After looking at all the data records, if there are left over resource objects, those represent objects
     * that were in Kubernetes but have since been deleted.  So we have to delete our resource objects from
     * the collection.  Notify the listeners of each Resource object (e.g. Host and their listening HostView)
     * of the impending deletion of the Resource, so that the listeners can appropriately clean up their own
     * state or (more likely) delete themselves from the DOM if they are Views.  See views/resources.js for more
     * information on how the View uses that notification to delete the corresponding child component.
     */
    for (let key of previousKeys) {
      let oldResource = this._resources.get(key);
      this.notifyListenersDeleted(oldResource);
      this._resources.delete(key);
    }
  }

  /* uniqueKeyFor(yaml)
 * Return a unique key given some structured resource data (a hierarchical key/value
 * structure) that is used to determine whether a collection already has an instance of the
 * Resource or whether a new one should be created.
 *
 * This is a method that is given the data block from a snapshot and returns
 * the unique key for that data.  Each ResourceCollection subclass will know the structure and extract
 * the appropriate information to create the Resource's key.  This is needed for identity in a
 * collection of Resources.
 *
 * It's only necessary to implement this method in a subclass of IResource if the resource data for the
 * particular kind of resource being collected has a different structure than a standard resource,
 * which always has kind, name, and namespace attributes, which together uniquely identify a Resource
 * within Kubernetes.
 *
 * Here we simply concatenate kind, name, and namespace to return the uniqueKey.
 */

  uniqueKeyFor(yaml) {
    return yaml.kind + "::" + yaml.metadata.name + "::" + yaml.metadata.namespace;
  }
}
