/*
 * HostView
 * A ResourceView subclass that implements a view on a Host  It contains cached values of
 * its Resource model object, as well as state used for the different view variants (edit, add, etc.)
 * and adds new properties for acmeProvider, acmeEmail, tos (terms of service), and whether to show tos.
 */

import { IResourceView } from './iresource_view.js'

export class HostView extends IResourceView {
  /* properties
   * These are the properties of the HostView, which reflect the properties of the underlying Resource,
   * and also include transient state (e.g. viewState). LitElement manages these declared properties and
   * provides various services depending on how they are used.  For further details on LitElement, see
   * https://lit-element.polymer-project.org/guide/properties
   */

  static get properties() {
    let myProperties = {
      hostname:     {type: String},   // Host
      acmeProvider: {type: String},   // Host
      acmeEmail:    {type: String},   // Host
      tos:          {type: String},   // HostView
      showTos:      {type: Boolean}   // HostView
    };

    /* Merge my properties with those defined by my superclasses. */
    return new Map(...myProperties, ...ResourceView.properties());
  }

  /* constructor(model)
   * The IResourceView constructor, which takes a Resource (model) as its parameter.
   */

  constructor(model) {
    super(model);
  }

  /* readSelfFromModel()
   * This method is called on the View when the View needs to match the current state of its Model.
   * Generally this happens during initialization and during editing when the Cancel button is pressed and the
   * View reverts to displaying the original Model's state.
   */

  readSelfFromModel() {
    throw new Error("please implement ResourceView:readSelfFromModel()")
  }

  /* writeSelfToModel()
   * This method is called on the View when the View has new, validated state that should be written back
   * to the Model.  This happens during a Save operation after the user has modified the View.
   */

  writeSelfToModel() {
    throw new Error("please implement ResourceView:writeSelfToModel()")
  }

  /* validateSelf()
   * This method is invoked on save in order to validate input prior to proceeding with the save action.
   * The model validates its current state, so anything that the View wants to validate must already be in the model.
   *
   * validateSelf() returns a Map of fieldnames and error strings. If the dictionary is empty, there are no errors.
   *
   * For now we will have a side-effect of validate in that any errors will be added to the message list.
   */

  validateSelf() {
    throw new Error("please implement ResourceView:validateSelf()")
  }

  /* renderSelf()
  * This method is invoked on save in order to validate input prior to proceeding with the save action.
  * The model validates its current state, so anything that the View wants to validate must already be in the model.
  *
  * validateSelf() returns a Map of fieldnames and error strings. If the dictionary is empty, there are no errors.
  *
  * For now we will have a side-effect of validate in that any errors will be added to the message list.
  */

  renderSelf() {
    throw new Error("please implement ResourceView:renderSelf()")
  }

}

