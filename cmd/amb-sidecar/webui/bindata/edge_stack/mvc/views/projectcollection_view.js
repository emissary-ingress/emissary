/*
 * ProjectCollectionView
 * An IResourceCollectionView concrete subclass that implements a view on an IResourceCollection of ProjectViews.
 */

import { AllProjects } from "../models/project_collection.js"
import { IResourceCollectionView } from '../interfaces/iresourcecollection_view.js'
import { html } from '../framework/view.js'
import './project_view.js'

export class ProjectCollectionView extends IResourceCollectionView {

  /* extend. See the explanation in IResourceCollectionView. */
  static get properties() {
    return IResourceCollectionView.properties;
  }

  /* extend */
  static get styles() {
    return IResourceCollectionView.styles;
   }

  /* extend */
  constructor() {
    super(AllProjects);
  }

  /* override */
  renderResource(resource) {
    return html`<dw-mvc-project .model=${resource}></dw-mvc-project>`
  }

  /* override */
  pageDescription() {
    return "Projects are custom HTTP services managed by Ambassador Edge Stack"
  }

  /* override */
  pageLogo() {
    return ["Projects Logo", "projects.svg"]
  }

  /* override */
  pageTitle() {
    return "Projects"
  }

}

/* Bind our custom elements to the HostCollectionView. */
customElements.define('dw-mvc-projects', ProjectCollectionView);
