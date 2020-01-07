/**
 * Collection
 * This is a Model subclass that monitors the snapshot data and keeps a consistent set of Resource objects
 * that mirror the actual model data in the snapshot.
 *
 * The ICollection interface class, which inherits from Collection, defines the three required methods for
 * creating specialized subclasses of Collection: resourceClass(), uniqueKeyFor(data), and extractDataFrom(snapshot).
 *
 * See ICollection for further details.
 */

import { Model }    from "./model.js";
import { Snapshot } from "../../components/snapshot.js";

export class Collection extends Model {

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
    let ResourceClass = this.resourceClass;

    /* For each of the snapshot data records for this model... */
    for (let resourceData of this.extractResourcesFrom(snapshot)) {
      let key = this.uniqueKeyFor(resourceData);
      /*
       * ...if we already have a model object for this data, then ask
       *    that object to check if it needs to update any data fields.
       */
      let existingResource = this._resources.get(key);

      if (existingResource) {
        if (existingResource.version !== resourceData.metadata.resourceVersion) {
          previousKeys.delete(key);
          existingResource.updateFrom(resourceData);
        }
      }
      else {
        /*
         * ...if we do not have a model object for this Resource (as defined by the unique key), then create a new
         * Resource object. After creating the object, notify all my listeners of the creation. See views/resources.js
         * for more information on how the ResourceListView uses that notification to add new child web components.
         */
        let newResource = new ResourceClass(resourceData);
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
}
