/**
 * Model is the concrete base class for all model classes.
 *
 * A model class is intended to capture business logic within a given
 * domain in a way that:
 *
 * - Allows a separation of concerns between business and presentation
 *   logic.
 *
 * - Facilitates automated testing of business logic.
 *
 * - Facilitates building multiple views of a given domain object and
 *   keeping them in sync.
 *
 * - Achieves a high performance UI by minimizing what needs to be
 *   re-rendered when data changes.
 *
 * What exactly this means in practice can be fuzzy, so here are some
 * guidelines:
 *
 * - A model class should never depend on any views or UI code. It
 *   must be possible to run all model code outside the browser in
 *   nodejs.
 *
 * - An instance of a model class must function as a source of truth
 *   for the state of the domain object it represents. In practice
 *   this means a model object will have a well defined concept of
 *   externalizable identity, and this identity will be used to ensure
 *   that there is only ever one instance of a model class associated
 *   with a given identity at a time.
 *
 * The code in this base class facilitates keeping multiple views in
 * sync by providing a simple listener API that notifies objects when
 * a Model has changed.
 *
 * Whenever Model.notify() is invoked, the onModelChanged(model, tag)
 * method is invoked on all registered objects.
 *
 * The listener API consists of addListener(listener[, tag]),
 * removeListener(listener[, tag]), and notify(). The
 * addListener/removeListener methods can be used manually, but there
 * is no need to do so if your web-component extends the View
 * class. The View class is a specialized subclass of LitElement that
 * has been made aware of the Model class and will automatically
 * register/unregister for notifications from any declared properties
 * that extend the Model class.
 *
 * This makes writing a View for one or more Models as simple as:
 *
 * 1. Write a web component, but extend View instead of LitElement.
 * 2. Make sure you store any relevant models in declared properties.
 *
 * That's it. Whenever Model.notify() is invoked, all Views that store
 * their Models in a declared property will automatically get a
 * requestUpdate(property), and will re-render.
 *
 */

export class Model {

  constructor() {
    // This map is keyed by listener and the value is a Set of tag names.
    this.listeners = new Map()
  }

  /**
   * Ask the model to invoke listener.onModelChanged(model, tag) when
   * changes occur.
   */
  addListener(listener, tag = "") {
    if (typeof listener.onModelChanged === "undefined") {
      throw new Error("listener does not have onModelChanged method")
    }
    var tags
    if (this.listeners.has(listener)) {
      tags = this.listeners.get(listener)
    } else {
      tags = new Set()
      this.listeners.set(listener, tags)
    }
    tags.add(tag)
  }

  /**
   * Ask the model to not invoke listener.onModelChanged(model, tag)
   * when changes occur.
   */
  removeListener(listener, tag = "") {
    if (this.listeners.has(listener)) {
      let tags = this.listeners.get(listener)
      tags.delete(tag)
      if (tags.size === 0) {
        this.listeners.delete(listener)
      }
    }
  }

  /**
   * Notify all listeners that the model has changed.
   */
  notify() {
    for (let [listener, tags] of this.listeners.entries()) {
      for (let tag of tags) {
        listener.onModelChanged(this, tag)
      }
    }
  }

}
