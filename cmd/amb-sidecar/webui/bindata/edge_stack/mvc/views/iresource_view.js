/*
 * IResourceView
 * A ResourceView subclass that implements a generic view on a Resource model object.  It contains cached values of
 * its Resource model object, as well as state used for the different view variants (edit, add, etc.)
 */

/* ResourceView superclass. */
import { ResourceView } from './resource_view.js'

export class IResourceView extends ResourceView {

  /* ====================================================================================================
   *  These methods must be implemented by subclasses.
   * ====================================================================================================
   */

  /* properties
   * These are the properties of the ResourceView, which reflect the properties of the underlying Resource,
   * and also include transient state (e.g. viewState). LitElement manages these declared properties and
   * provides various services depending on how they are used.  For further details on LitElement, see
   * https://lit-element.polymer-project.org/guide/properties
   */

  static get properties() {
    /* For reference, subclasses implement the static function properties() by returning a Map
     * that lists the properties in the subclass, and merging with the parent's properties():

*     first, import set merge:
*     import { mapMerge } from "./map.js"

      static get properties() {
      let myProperties = {
        hostname:     {type: String},   // Host
        acmeProvider: {type: String},   // Host
        acmeEmail:    {type: String},   // Host
        tos:          {type: String},   // HostView
        showTos:      {type: Boolean}   // HostView
      };

      return mapMerge(myProperties, ResourceView.properties);
     */


    /* The interface simply returns the properties of the ResourceView. */
    return ResourceView.properties;
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
    throw new Error("please implement ${this.constructor.name}.readSelfFromModel()")
  }

  /* writeSelfToModel()
   * This method is called on the View when the View has new, validated state that should be written back
   * to the Model.  This happens during a Save operation after the user has modified the View.
   */

  writeSelfToModel() {
    throw new Error("please implement ${this.constructor.name}.writeSelfToModel()")
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
    throw new Error("please implement ${this.constructor.name}.validateSelf()")
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
    throw new Error("please implement ${this.constructor.name}.renderSelf()")
  }
}

