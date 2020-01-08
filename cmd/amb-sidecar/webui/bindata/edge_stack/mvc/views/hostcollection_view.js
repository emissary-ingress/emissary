/*
 * HostCollectionView
 * An ICollectionView concrete subclass that implements a view on a Collection of HostViews.
 */

/* The HostView that will be rendered in the CollectionView. */
import { HostView } from "./host_view.js"

/* CollectionView interface class */
import { ICollectionView } from './icollection_view.js'

export class HostCollectionView extends ICollectionView {

  /* properties()
   * These are the properties of the HoatCollectionView. LitElement manages these declared properties and
   * provides various services depending on how they are used.  For further details on LitElement, see
   * https://lit-element.polymer-project.org/guide/properties
   */

  static get properties() {
    return ICollectionView.properties;
  }

  /* styles
   * These are the styles of the HostCollectionView. LitElement allows each Element to provide
   * additional css style specifications that are valid only for that LitElement.
   */

  static get styles() {
    return ICollectionView.styles;
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
    throw Error("Not Yet Implemented");
  }

  /**
   * Override to false to allow the Add button to show up.
   */
  readOnly() {
    return true;
  }

  /* onModelNotification.
  * Listener for model-created notifications.  This is called when a new Host has been created, and a
  * new view must be created to display that Host.
  */

  onModelNotification(model, message, parameter) {
    if (message === 'created') {
      /* Create a new dw-host web component and add it as a child. Because this view is a web component, adding
       * that child component queues the appropriate re-render at the correct time,and are rendered in our <slot>.
      */

      let child_view = new HostView(model);
      this.appendChild(child_view);
    }
  }
}

/* Bind our custom elements to the HostCollectionView. */
customElements.define('dw-hosts', HostCollectionView);
