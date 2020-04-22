/*
 * HostCollectionView
 * An IResourceCollectionView concrete subclass that implements a view on an IResourceCollection of HostViews.
 */

import { AllHosts } from "../models/host_collection.js"
import { HostView } from "./host_view.js"
import { IResourceCollectionView } from '../interfaces/iresourcecollection_view.js'
import { html } from '../../vendor/lit-element.min.js'

export class HostCollectionView extends IResourceCollectionView {

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
    super(AllHosts);
  }

  /* override */
  renderResource(resource) {
    return html`<dw-mvc-host .model=${resource}></dw-mvc-host>`
  }

  /* override */
  pageDescription() {
    return "Hosts are domains that are managed by Ambassador Edge Stack, e.g., example.org"
  }

  /* override */
  pageLogo() {
    return ["Hosts Logo", "hosts.svg"]
  }

  /* override */
  pageTitle() {
    return "Hosts"
  }

}

/* Bind our custom elements to the HostCollectionView. */
customElements.define('dw-mvc-hosts', HostCollectionView);
