/**
 * IResourceCollection
 * This is the Interface class for ResourceCollection.
 *
 * IResourceCollection subclasses, such as HostCollection, need only define three methods to specialize:
 *
 * - extractResourcesFrom(snapshot)
 *   this method is responsible for finding the right place in the snapshot to extract data objects that are used
 *   by the Resource's initFrom method to create a new instance of that resource.  This function returns a
 *   list that can be iterated over, returning data objects.
 *
 * - resourceClass()
 *   this method returns the IResource class to be instantiated when adding a new item, e.g. when implementing
 *   HostCollection, the resourceClass() should return HostResource.
 *
 * - uniqueKeyFor(yaml)
 *   this method computes a unique key for the collection's Resource, to determine whether the Resource described
 *   by the data block has already been initialized and stored in the ResourceCollection.
 *
 * Most Resources (CRD's) have the same data formats but there are other objects in the snapshot that
 * are not CRD's, have different structure, and are not in the same part of the snapshot (e.g. Resolvers).
 *
 * Listeners will be notified when Models are added, updated, or removed from the collection.  The collection's
 * listeners are generally Views, which implement the method onModelNotification(model, message, parameter).
 */

import { ResourceCollection } from "../framework/resource_collection.js";

export class IResourceCollection extends ResourceCollection {

  /* ====================================================================================================
   *  These methods must be implemented by subclasses.
   * ====================================================================================================
   */

  /* constructor()
   */
  constructor() {
    super();
  }

  /* extractResourcesFrom(snapshot)
   * Given a snapshot as received from the backend via snapshot.js, return a list of resource data blocks
   * given the resource's class name (e.g. Host, Mapping, Filter...).
   */
  extractResourcesFrom(snapshot) {
    throw new Error("please implement ${this.constructor.name}.extractResourcesFrom(snapshot)")
  }

  /* resourceClass()
   * Return the class of the resource that is being collected from the snapshot.
   */
  resourceClass() {
    throw new Error("Please implement ${this.constructor.name}.resourceClass()");
  }

  /* uniqueKeyFor(yaml)
   * Return a unique key given some structured resource data (a hierarchical key/value
   * structure) that is used to determine whether a collection already has an instance of the
   * Resource or whether a new one should be created.
   *
   * It's only necessary to implement this method in a subclass of IResource if the resource data for the
   * particular kind of resource being collected has a different structure than a standard resource,
   * which always has kind, name, and namespace attributes, which together uniquely identify a Resource
   * within Kubernetes.
   */
  uniqueKeyFor(yaml) {
    return super.uniqueKeyFor(yaml);
  }

  /* ====================================================================================================
   *  Subclasses do not implement the following methods.  They are implemented by Model and may be used by
   *  subclasses directly.
   * ====================================================================================================
   */

  /* Add a new listener for changes.  The listener's onModelNotification method will be called when the
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
   * The listener's onModelNotification(model, message, parameter) method will be called.  Only Listeners
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

