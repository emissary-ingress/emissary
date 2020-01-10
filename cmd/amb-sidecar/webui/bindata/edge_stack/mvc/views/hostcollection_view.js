/*
 * HostCollectionView
 * An IResourceCollectionView concrete subclass that implements a view on a ResourceCollection of HostViews.
 */

/* The ResourceCollection we're listening to. */
import { AllHosts } from "../models/host_collection.js"

/* The HostView that will be rendered in the HostCollectionView. */
import { HostView } from "./host_view.js"

/* ResourceCollectionView interface class */
import { IResourceCollectionView } from '../interfaces/iresourcecollection_view.js'

export class HostCollectionView extends IResourceCollectionView {

  /* properties()
   * These are the properties of the HoatCollectionView. LitElement manages these declared properties and
   * provides various services depending on how they are used.  For further details on LitElement, see
   * https://lit-element.polymer-project.org/guide/properties
   */

  static get properties() {
    return IResourceCollectionView.properties;
  }

  /* styles
   * These are the styles of the HostCollectionView. LitElement allows each Element to provide
   * additional css style specifications that are valid only for that LitElement.
   */

  static get styles() {
    return IResourceCollectionView.styles;
   }

  /* constructor()
   */

  constructor() {
    super();

    /* Listen to AllHosts for updates. */
    AllHosts.addListener(this);
  }

  /* viewClass()
   * Return HostView for instantiating new views in this resource collection.
   */

  viewClass() {
    return HostView;
  }
}

/* Bind our custom elements to the HostCollectionView. */
customElements.define('dw-mvc-hosts', HostCollectionView);
