/*
 * IResourceCollectionView
 */

import { ResourceCollectionView } from "../framework/resourcecollection_view.js"

export class IResourceCollectionView extends ResourceCollectionView {

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
     *       someProperty: {type: String}
     *     };
     *     return objectMerge(myProperties, IResourceCollectionView.properties);
     *   }
     */
    return ResourceCollectionView.properties;
  }

  /* styles
   * These are the styles of the IResourceCollectionView. LitElement allows each Element to provide
   * additional css style specifications that are valid only for that LitElement.
   */
  static get styles() {
    return ResourceCollectionView.styles;
  }

  /* constructor(model)
   * model is an IResourceCollection subclass.
   */
  constructor(model) {
    super(model);
  }

  /**
   * renderResource(resource)
   *
   * Renders a single resource. This is normally just:
   *
   *   return html`<dw-resourcename .model=${resource}></dw-resourcename>`
   *
   * But can be customized as whatever you like if desired.
   */

  renderResource(resource) {
    throw new Error("please implement ${this.constructor.name}.renderResource(resource)")
  }

  /* pageDescription()
   * Return the text describing the contents of the IResourceCollection being viewed
   * e.g. "Hosts are domains that are managed by Ambassador Edge Stack, e.g., example.org"
   */
  pageDescription() {
    throw new Error("please implement ${this.constructor.name}.pageDescription()")
  }

  /* pageLogo()
  * Return the alternate text and logo filename in an array [] of the IResourceCollection being viewed
  * e.g. ["Hosts Logo", "hosts.svg"]
  */
  pageLogo() {
    throw new Error("please implement ${this.constructor.name}.pageLogo()")
  }

  /* pageTitle()
  * Return the title of the IResourceCollection being viewed (e.g. "Hosts").
  */
  pageTitle() {
    throw new Error("please implement ${this.constructor.name}.pageTitle()")
  }

  /* readOnly()
   * Defaults to false. Override to true to hide the Add button.
   */
  readOnly() {
    return super.readOnly();
  }
}

