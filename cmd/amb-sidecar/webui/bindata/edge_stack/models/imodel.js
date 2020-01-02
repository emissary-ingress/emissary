/**
 * IModel
 * This is the Model interface class that defines the methods that any Model subclass should implement.
 * A Model maintains specific state for that Model, and a set of Listeners that require notification of
 * the model's state changes.  These listeners, typically Views in a Model-View-Controller architecture,
 * register themselves with one or more Models and will be called when any of the Models' state changes.
 *
 * Listeners are notified of changes through the Listeners onModelNotification method:
 * onModelNotification(notifyingModel, message, parameter).
 *
 * There is no Listener interface; any object that defines onModelNotification can be a listener.
 *
 * Standard messages are:
 *    notifyingModel 'created'  = has been created
 *    notifyingModel 'updated'  = one or more instance variables have new values
 *    notifyingModel 'deleted'  = just about to be deleted
 *
 * Subclasses of Model may want to have additional messages for specific changes that they note.
 * Listeners may subscribe to all messages or a selected list of messages.
 */

export class IModel {

  /* constructor()
   * Here the model initializes any internal state including any structures for storing Listeners
   * that have subscribed to the Model.
   */

  constructor() {
    /* do nothing for now, concrete subclasses will implement, but subclasses must call super()
    *  in their own constructor.
    */
  }

  /* uniqueID()
   * We have a unique ID for every model created during the session.  This is needed for labeling views
   * with their corresponding model ID's, for identifying the views in the DOM.
   */

  uniqueID() {
    throw new Error("Please implement Model:uniqueID()")
  }

  /* Add a new listener for changes.  The Listener's onModelNotification method will be called when the
  *  model is notifying it for any of the  messages listed in the message set.  if the message set is
  *  null, then add this listener for all messages.
  */
  addListener(listener, messageSet = null) {
    throw new Error("Please implement Model:addListener()")
  }

  /* Remove a listener from the given messages, or from all messages if null */
  removeListener(listener, messageSet = null) {
    throw new Error("Please implement Model:removeListener()")
  }

  /* Notify listeners of a update in the model with the given message.  Only listeners who have subscribed
   * to the message will be notified.  Listeners that have subscribed to all messages will also be notified.
   * The Listener's onModelNotification(model, message, parameter) method will be called.  Only Listeners
   * who have subscribed to the message will be notified. Listeners that have subscribed to all messages
   * will also receive a callback. Includes a notification message, the model itself, and an optional parameter.
   */
  notifyListeners(model = this, message, parameter = null) {
    throw new Error("Please implement Model:notifyListeners()")
  }
}

