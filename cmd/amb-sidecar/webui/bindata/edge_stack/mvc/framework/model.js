/**
 * Model
 * This is the concrete Model class.
 * A Model maintains specific state for that Model, and a set of Listeners that require notification of
 * the model's state changes.  These listeners, typically Views in a Model-View-Controller architecture,
 * register themselves with one or more Models and will be called when any of the Models' state changes.
 *
 * Listeners are notified of changes through the Listeners onModelNotification method:
 * onModelNotification(notifyingModel, message, parameter).
 *
 * Any object that defines onModelNotification can be a listener.
 *
 * Standard messages are:
 *    notifyingModel 'created'  = has been created
 *    notifyingModel 'updated'  = one or more instance variables have new values
 *    notifyingModel 'deleted'  = just about to be deleted
 *
 * Subclasses of Model may want to have additional messages for specific changes that they note.
 * Listeners may subscribe to all messages or a selected list of messages.c
 */

/* Utility functions . */
import { setUnion } from "../framework/utilities.js"

export class Model {

  /* constructor()
   * Here the model initializes any internal state including any structures for storing Listeners
   * that have subscribed to the Model.
   */

  constructor() {
    /* The listeners are stored in Sets, and the sets are stored in a Map where the map's keys are the
      * message names and the values are sets of listeners to be called when that message is sent.
      * _listenersToAll is the set of listeners that want to be notified on all messages.
      */

    this._listenersByMessage  = new Map();
    this._listenersToAll      = new Set();
  }

  /* addListener(listener, messageSet = null)
   * Add a new listener for changes.  The listener's onModelNotification method will be called when the
   *  model is notifying it for any of the  messages listed in the message set.  if the message set is
   *  null, then add this listener for all messages.
   */

  addListener(listener, messageSet = null) {
    if (messageSet === null) {
      this._listenersToAll.add(listener);
    }
    else {
      for (let message of messageSet) {
        let set = this._listenersByMessage.has(message) ? this._listenersByMessage[message] : new Set();
        set.add(listener);
        this._listenersByMessage[message] = set;
      }
    }
  }


  /* removeListener(listener, messageSet = null)
   * Remove a listener from the given messages, or from all messages if null
   */

  removeListener(listener, messageSet = null) {
    /* Complete removal */
    if (messageSet === null) {
      /* Go through every message set and remove. */
      for (let [_, listeners] of this._listenersByMessage) {
        listeners.delete(listener);
      }
      /* Delete from allListeners too. */
      this._listenersToAll.delete(listener);
    }

    /* Remove from each requested message set */
    else {
      for (let message of messageSet) {
        if (this._listenersByMessage.has(message)) {
          this._listenersByMessage[message].delete(listener);
        }
      }
    }
  }

  /* notifyListeners(notifyingModel, message, parameter)
   * Notify listeners of a update in the model with the given message.  Only listeners who have subscribed
   * to the message will be notified.  Listeners that have subscribed to all messages will also be notified.
   * The listener's onModelNotification(model, message, parameter) method will be called.  Only listeners
   * who have subscribed to the message will be notified. Listeners that have subscribed to all messages
   * will also receive a callback. Includes a notification message, the model itself, and an optional parameter.
   */

  notifyListeners(notifyingModel = this, message, parameter = null) {
    for (let listener of this._listenersForMessage(message)) {
      listener.onModelNotification(notifyingModel, message, parameter);
    }
  }

  /* notifyListenerUpdated(notifyingModel)
   * Convenience methods for notifying listeners of an updated model.
   */

  notifyListenersUpdated(notifyingModel = this) {
    this.notifyListeners(notifyingModel, 'updated');
  }

  /* notifyListenersCreated(notifyingModel)
   * Convenience methods for notifying listeners of a newly-created model.
   */

  notifyListenersCreated(notifyingModel = this) {
    this.notifyListeners(notifyingModel, 'created');
  }

  /* notifyListenersDeleted(notifyingModel)
   * Convenience method for notifying listeners of a deleted model.
   */

  notifyListenersDeleted(notifyingModel = this) {
    this.notifyListeners(notifyingModel, 'deleted');
  }

  /* _listenersForMessage(message)
   * Return the listeners for a given message, or an empty set if none.
   */

  _listenersForMessage(message) {
    let allListeners = this._listenersToAll;
    let msgListeners = this._listenersByMessage.has(message) ? this._listenersByMessage[message] : new Set();

    return setUnion(allListeners, msgListeners);
  }
}
