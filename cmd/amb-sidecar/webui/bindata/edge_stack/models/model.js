/**
 * Model
 * a concrete implementation, the Model class implements the IModel interface.
 *
 * This implementation uses a Map and a Set to maintain listener object for notification.
 */

/* Utility functions for sets. */
import {union} from "./set.js"

/* Interface class for Model */
import { IModel } from "./imodel.js"

export class Model extends IModel {
  /* constructor()
   * Here the model initializes any internal state including any structures for storing Listeners
   * that have subscribed to the Model.
   */

  constructor() {
    /* The listeners are stored in Sets, and the sets are stored in a Map where the map's keys are the
      * message names and the values are sets of listeners to be called when that message is sent.
      * _listenersToAll is the set of listeners that want to be notified on all messages.
      */

    super();

    this._listenersByMessage  = new Map();
    this._listenersToAll      = new Set();
  }

  /* uniqueID()
   * We have a unique ID for every model created during the session.  This is needed for labeling views
   * with their corresponding model ID's, for identifying the views in the DOM.
   */

  uniqueID() {
    return `Model#${this._modelUID}`;
  }

  /* Add a new listener for changes.  The Listener's onModelNotification method will be called when the
  *  model is notifying it for any of the  messages listed in the message set.  if the message set is
  *  null, then add this listener for all messages.
  */
  /* Add a new listener for changes.  The function will be called
  *  when the model is notifying listeners for any of the given
  *  messages listed in the message set.  if the message set is
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


  /* Remove a listener from the given messages, or from all messages if null */
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

  /* Notify listeners of a update in the model with the given message.  Only listeners who have subscribed
   * to the message will be notified.  Listeners that have subscribed to all messages will also be notified.
   * The Listener's onModelNotification(model, message, parameter) method will be called.  Only Listeners
   * who have subscribed to the message will be notified. Listeners that have subscribed to all messages
   * will also receive a callback. Includes a notification message, the model itself, and an optional parameter.
   */

  notifyListeners(model = this, message, parameter = null) {
    for (let listener of this.listenersForMessage(message)) {
      listener(message, model, parameter);
    }
  }

   /* Return the listeners for a given message, or an empty set if none. */
  _listenersForMessage(message) {
    let allListeners = this._listenersToAll;
    let msgListeners = this._listenersByMessage.has(message) ? this._listenersByMessage[message] : new Set();

    return union(allListeners, msgListeners);
  }

  /* Convenience methods for updated, created, deleted. */
  notifyListenersUpdated(model) {
    this.notifyListeners(model, 'updated');
  }

  notifyListenersCreated(model) {
    this.notifyListeners(model, 'created');
  }

  notifyListenersDeleted(model) {
    this.notifyListeners(model, 'deleted');
  }
}
