/*
 * IResourceCollectionView
 * This is the Interface class to the ResourceCollectionView.
 */

import { ResourceCollectionView } from "./resourcecollection_view.js"

export class IResourceCollectionView extends ResourceCollectionView {

  /* properties()
   * These are the properties of the ResourceCollectionView. LitElement manages these declared properties and
   * provides various services depending on how they are used.  For further details on LitElement, see
   * https://lit-element.polymer-project.org/guide/properties
   */

  static get properties() {
    /* For reference, subclasses implement the static function properties() by returning a Map
     * that lists the properties in the subclass, and merging with the parent's properties():

*     first, import object merge:
*     import { objectMerge } from "../framework/utilities.js"

      in properties():
      let myProperties = {
        someProperty:     {type: String},
        ...
      };

      return objectMerge(myProperties, ResourceCollectionView.properties());
     */

    /* The interface simply returns the properties of the ResourceCollectionView. */
    return ResourceCollectionView.properties;
  }

  /* styles
   * These are the styles of the ResourceCollectionView. LitElement allows each Element to provide
   * additional css style specifications that are valid only for that LitElement.
   *
   * The interface simply returns the styles of the ResourceCollectionView.
   */

  static get styles() {
    return ResourceCollectionView.styles;
  }

  /* constructor()
   */

  constructor() {
    super();
  }

  /* readOnly()
   * Override to false to allow the Add button to show up.  Defaults to false.
   */
  readOnly() {
    return super.readOnly();
  }

  /* viewClass()
   * Return the viewClass that the subclass uses to create a new view in the ResourceCollectionView.
   * e.g. for a HostCollection, return HostView.
   */
  viewClass() {
    throw new Error("please implement ${this.constructor.name}.viewClass()")
  }
}

