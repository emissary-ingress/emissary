/**
 * Collection
 * This is a Model subclass that monitors the snapshot data and keeps a consistent set of Resource objects
 * that mirror the actual model data in the snapshot.
 *
 * Collection subclasses, such as HostCollection, need only define one method to specialize:
 *  - resourceClass(), which returns the class that should be instantiated if a new item (e.g. Host) is added
 *
 * The Resource subclass must implement two static functions on the class that are needed for Collections:
 *
 * - dataExtractor(snapshot)
 *   this method is responsible for finding the right place in the snapshot to extract data objects that are used
 *   by the Resource's initFrom method to create a new instance of that resource.  This function returns a
 *   list that can be iterated over, returning data objects.
 *
 * - resourceKeyFor(data)
 *   this function takes a data object that dataExtractor returns and computes a unique key for the model so that
 *   the collection can determine whether a new data object represents a new Model or if it already exists in the
 *   collection.
 *
 * Most Resources (CRD's) have the same data formats but there are other objects in the snapshot that
 * are not CRD's, have different structure, and are not in the same part of the snapshot (e.g. Resolvers).  Similarly,
 * different Model classes will generate different unique keys, and so each will implement a class function
 * resourceKeyFor(data).
 *
 * Listeners will be notified when Models are added, updated, or removed from the collection.  The collection's
 * listeners are generally Views.
 */

import {Model}    from "./model.js";
import {Snapshot} from "../components/snapshot.js";

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
    let resourceClass = this.resourceClass();

    /* For each of the snapshot data records for this model... */
    for (let data of this._modelExtractor(snapshot)) {
      let key = resourceClass.modelKeyFor(data);
      /*
       * ...if we already have a model object for this data, then ask
       *    that object to check if it needs to update any data fields.
       */
      let existingModel = this._resources.get(key);
      if (existingModel) {
        previousKeys.delete(key);
        existingModel.updateFrom(data);
      } else {
        /*
         * ...if we do not have a model object for this Resource (as defined by the unique key), then create a new
         * Resource object. After creating the object, notify all my listeners of the creation. See views/resources.js
         * for more information on how the ResourceListView uses that notification to add new child web components.
         */
        let newResource = new resourceClass(data);
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
      let oldModel = this._resources.get(key);
      this.notifyListenersDeleted(oldModel);
      this._resources.delete(key);
    }
  }

  /* resourceClass()
   * Return the class of the resource that is being collected from the snapshot.
   */

  resourceClass() {
    throw new Error("Please implement Collection:resourceClass()");
  }
}
