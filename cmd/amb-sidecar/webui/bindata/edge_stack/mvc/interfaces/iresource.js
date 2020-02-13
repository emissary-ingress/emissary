/**
 * IResource
 * This is the Resource interface class that defines the methods that any Resource subclass must implement
 * as well as listing all the superclass methods that a subclass can utilize.
 *
 * A Resource is a Model that maintains basic Kubernetes resource state and implements code to create
 * instances of that Resource from snapshot data.
 *
 * A Kubernetes Resource (also known as an Object) is a persistent entity in the Kubernetes system that is
 * used to represent the desired state of your cluster.  Kubernetes uses Resources to apply policies and
 * procedures to ensure that the cluster reaches that desired state, by allocating, deallocating, configuring
 * and running compute and networking entities.
 *
 * The Kubernetes documentation makes a distinction between a Kubernetes Object 9the persistent data entities
 * that are stored in the etcd database) and a Resource (an endpoint in the Kubernetes APO that stores a
 * collection of objects).  However, this distinction is not made consistently throughout the documentation.
 * Here we will use the term Resource to indicate a chunk of data--a kind, name, namespace, metadata, and
 * a spec--that we display and modify in the Web user interface.
 */

import { Resource } from "../framework/resource.js"

export class IResource extends Resource {

  /* ====================================================================================================
   *  These methods must be implemented by subclasses.
   * ====================================================================================================
   */

  /* constructor()
   * The constructor is internally implemented by calling this.updateSelfFrom(yaml) and
   * thus typically all the instance variables are initialized by that method.
   */
  constructor(yaml) {
    super(yaml);
  }

  /* copySelf()
   * Return a new instance with the same state.
   */
  copySelf() {
    throw new Error("Please implement ${this.constructor.name}:copySelf()");
  }

   /* getYAML()
    * Return YAML that has the Resource's values written back into the _fullYAML, and has been pruned so that only
    * the necessary attributes exist in the structure for use as the parameter to applyYAML().
    */
  getYAML() {
    return super.getYAML();
  }

  /* updateSelfFrom(yaml)
   * Update the Resource object state from the snapshot data block for this Resource.  Compare the values in the
   * data block with the stored state in the Resource.  If the data block has different data than is currently
   * stored, update that instance variable with the new data and set a flag to return true if any changes have
   * occurred.
   * This method is called from Resource.updateFrom() which will also notify listeners as needed.
   */
  updateSelfFrom(yaml) {
    throw new Error("Please implement ${this.constructor.name}:updateSelfFrom(yaml)");
  }

  /* validateSelf()
   * Validate this Resource's state by checking each object instance variable for correctness (e.g. email address
   * format, URL format, date/time, name restrictions).  Returns a dictionary of property: errorString if there
   * are any errors. If the dictionary is empty, there are no errors.
   */
  validateSelf() {
    throw new Error("Please implement ${this.constructor.name}:validateSelf()");
  }

  /* yamlIgnorePaths()
   * Return an array of paths arrays to be ignored when sending YAML to Kubernetes.  This is needed because Kubernetes
   * sends extra information in the Resource object that confuses it when sent back; only the data that is needed
   * (e.g. name, namespace, kind, and desired labels/annotations/spec) should be sent back.
   *
   * NOTE: one would think that a full path could be described by a string with the path delimiter "."
   * to separate the path elements.  BUT, Kubernetes allows keys in the YAML to use the same delimiter,
   * so we have to have arrays of path elements.  e.g. you can't parse at "." to get the full path for
   * "metadata.annotations.kubectl.kubernetes.io/last-applied-configuration"
   * because it is really
   * "metadata"."annotations"."kubectl.kubernetes.io/last-applied-configuration"
   */
  yamlIgnorePaths() {
    return super.yamlIgnorePaths();
  }


  /* ====================================================================================================
   * The following methods are implemented by Model, and may be useful for subclasses to use in their
   * implementation of the required interface methods.  These methods should not be overridden by
   * subclasses.
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


  /* ====================================================================================================
   * The following methods are implemented by Resource, and may be useful for subclasses to use in their
   * implementation of the required interface methods.  These methods should not be overridden by
   * subclasses.
   * ====================================================================================================
   */

  /* validateName(name)
   * returns null if name is valid, error string if not.
   */
  validateName(name) {
    return super.validateName(name);
  }

  /* validateEmail(email)
   * returns null if email is valid, error string if not.
   */
  validateEmail(email) {
    return super.validateEmail(email);
  }

  /* _validateURL(url)
  * returns null if url is valid, error string if not.
  */
  validateURL(url) {
    return super.validateURL(url);
  }

}

