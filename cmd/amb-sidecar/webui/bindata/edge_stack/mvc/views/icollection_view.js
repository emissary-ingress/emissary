/*
 * ICollectionView
 * This is the Interface class to the CollectionView.
 */

/* Map merge operation */
import { mapMerge } from "./map.js"

export class ICollectionView extends CollectionView {

  /* properties()
   * These are the properties of the CollectionView. LitElement manages these declared properties and
   * provides various services depending on how they are used.  For further details on LitElement, see
   * https://lit-element.polymer-project.org/guide/properties
   */

  static get properties() {
    /* For reference, subclasses implement the static function properties() by returning a Map
     * that lists the properties in the subclass, and merging with the parent's properties():

*     first, import set merge:
*     import { mapMerge } from "./map.js"

      in properties():
      let myProperties = {
        someProperty:     {type: String},
        ...
      };

      return mapMerge(myProperties, CollectionView.properties());
     */

    /* The interface simply returns the properties of the CollectionView. */
    return CollectionView.properties();
  }

  /* styles
   * These are the styles of the CollectionView. LitElement allows each Element to provide
   * additional css style specifications that are valid only for that LitElement.
   *
   * The interface simply returns the styles of the CollectionView.
   */

  static get styles() {
    return CollectionView.styles();
  }

  /* constructor(model, sortFields)
   * model is the Collection that is being rendered by this CollectionView.
   * sortFields is an array of {value: label} objects, where the value is the Resource property
   * on which to sort, and label is the display name for the HTML component.
   */

  constructor(model, sortFields) {
    super(model, sortFields);
  }

  onAdd() {
    throw Error("Please implement CollectionView.onAdd()")
  }

  /**
   * Override to false to allow the Add button to show up.
   */
  readOnly() {
    throw new Error("please implement ${this.constructor.name}.readOnly()")
  }

  /* onModelNotification.
  * Listener for model-created notifications.  This is called when a new Resource has been created, and a
  * new view must be created to display that Resource.
  */

  onModelNotification(model, message, parameter) {
    throw new Error("please implement ${this.constructor.name}.onModelNotification(model, message, parameter)")
  }
}

