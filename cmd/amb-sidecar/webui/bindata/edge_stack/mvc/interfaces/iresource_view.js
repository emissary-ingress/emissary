/*
 * IResourceView
 * A ResourceView subclass that implements a generic view on a Resource model object.  It contains cached values of
 * its Resource model object, as well as state used for the different view variants (edit, add, etc.)
 */

import { ResourceView } from '../framework/resource_view.js'

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
    /* Note that you MUST implement your subclasses static function properties() by returning a Map
     * that lists the properties in the subclass, and merging with the parent's properties(). This
     * whole MVC framework will NOT work if you fail to merge with the parent's properties.
     *
     * first, import object merge:
     *   import { objectMerge } from "../framework/utilities.js"
     *
     * then, implement your static get properties that merges with the superclass's get properties:
     *   static get properties() {
     *     let myProperties = {
     *       hostname:     {type: String},   // Host
     *       acmeProvider: {type: String},   // Host
     *       acmeEmail:    {type: String},   // Host
     *       tos:          {type: String},   // HostView
     *       showTos:      {type: Boolean}   // HostView
     *     };
     *     return objectMerge(myProperties, IResourceView.properties);
     *   }
     */
    return ResourceView.properties;
  }

  /* validateSelf()
   * This method is invoked on a Save in order to validate input prior to proceeding with the save action.
   * Returns a Map of field names and error strings. If the dictionary is empty, there are no errors.
   */
  validateSelf() {
    throw new Error("please implement ${this.constructor.name}.validateSelf()")
  }

  /* renderSelf()
   * This method renders the view within the HTML framework set up by ResourceView.render().
   */
  renderSelf() {
    throw new Error("please implement ${this.constructor.name}.renderSelf()")
  }


}

