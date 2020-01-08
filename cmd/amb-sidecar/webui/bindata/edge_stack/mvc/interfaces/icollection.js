/**
 * ICollection
 * This is the Interface class for Collection.
 *
 * ICollection subclasses, such as HostCollection, need only define three methods to specialize:
 * - resourceClass(), which returns the class that should be instantiated if a new item (e.g. Host) is added
 *
 * - uniqueKeyFor(resourceData)
 *   this method computes a unique key for the collection's Resource, to determine whether the Resource described
 *   by the data block has already been initialized and stored in the Collection.
 *
 * - extractResourcesFrom(snapshot)
 *   this method is responsible for finding the right place in the snapshot to extract data objects that are used
 *   by the Resource's initFrom method to create a new instance of that resource.  This function returns a
 *   list that can be iterated over, returning data objects.
 *
 * Most Resources (CRD's) have the same data formats but there are other objects in the snapshot that
 * are not CRD's, have different structure, and are not in the same part of the snapshot (e.g. Resolvers).
 *
 * Listeners will be notified when Models are added, updated, or removed from the collection.  The collection's
 * listeners are generally Views.
 */

import { Collection } from "../framework/collection.js";

export class ICollection extends Collection {

  /* ====================================================================================================
   *  These methods must be implemented by subclasses.
   * ====================================================================================================
   */

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

  /* ====================================================================================================
   *  Subclasses do not implement these methods.  They are implemented by Model and may be used by
   *  subclasses directly.
   * ====================================================================================================
   */

  /* Add a new listener for changes.  The Listener's onModelNotification method will be called when the
   *  model is notifying it for any of the  messages listed in the message set.  if the message set is
   *  null, then add this listener for all messages.
   */

  addListener(listener, messageSet = null) {
    super.addListener(listener, messageSet);
  }


  /* Remove a listener from the given messages, or from all messages if null */
  removeListener(listener, messageSet = null) {
    super.removeListener(listener, messageSet);
   }

  /* Notify listeners of a update in the model with the given message.  Only listeners who have subscribed
   * to the message will be notified.  Listeners that have subscribed to all messages will also be notified.
   * The Listener's onModelNotification(model, message, parameter) method will be called.  Only Listeners
   * who have subscribed to the message will be notified. Listeners that have subscribed to all messages
   * will also receive a callback. Includes a notification message, the model itself, and an optional parameter.
   */

  notifyListeners(notifyingModel = this, message, parameter = null) {
    super.notifyListeners(notifyingModel, message, parameter);
  }

  /* Convenience methods for updated, created, deleted. */
  notifyListenersUpdated(notifyingModel) {
    super.notifyListeners(notifyingModel, 'updated');
  }

  notifyListenersCreated(notifyingModel) {
    super.notifyListeners(notifyingModel, 'created');
  }

  notifyListenersDeleted(notifyingModel) {
    super.notifyListeners(notifyingModel, 'deleted');
  }
}

