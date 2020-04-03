/*
 * HostView
 * A ResourceView subclass that implements a view on a Host  It contains cached values of
 * its Resource model object, as well as state used for the different view variants (edit, add, etc.)
 * and adds new properties for acmeProvider, acmeEmail, tos (terms of service), and whether to show tos.
 */

import { html } from '../../vendor/lit-element.min.js'
import { objectMerge } from "../framework/utilities.js"
import { IResourceView } from '../interfaces/iresource_view.js'

export class HostView extends IResourceView {

  /* ====================================================================================================
   *  These functions and methods implement the IResourceView interface.
   * ====================================================================================================
   */

  /* extend */
  constructor() {
    super();
  }

  get agreed() {
    return this.model.agreed_terms_of_service
  }

  set agreed(value) {
    this.model.agreed_terms_of_service = value
  }

  get tos() {
    return this.model.getTermsOfService()
  }

  /* isTOSShowing()
   * Are the terms of service being shown during an Add operation?
   */
  isTOSShowing() {
    return this.model.useAcme && (this.viewState === "add" || this.viewState === "edit");
  }

  /* override */
  validateSelf() {
    let errors = new Map();

    /* Validate that the user has agreed to the Terms of Service, which is either:
     * (i) if  not showing the Terms of Service, then assume that they have already agreed, or
     * (ii) if the TOS is shown, then the checkbox needs to be checked.
     */
    if (this.model.useAcme && this.isTOSShowing() && !this.agreed) {
      errors.set("tos", "You must agree to terms of service");
    }

    return errors;
  }

  /* override */
  renderSelf() {
    let host    = this.model;
    let status  = host.status || {"state": "<none>"};
    let state   = status.state;
    let reason  = (state === "Error") ? `(${status.errorReason})` : '';
    let acme    = (this.model.useAcme ? "": "none");
    let tos     = this.isTOSShowing() ? "attribute-value" : "off";
    let editing = this.viewState === "add" || this.viewState === "edit";

    return html`
      <div class="row line">
        <div class="row-col margin-right justify-right">hostname:</div>
        <div class="row-col">
          <span class="${this.visibleWhen("list", "pending")}">${this.model.hostname}</span>
          <input class="${this.visibleWhen("edit", "add")}"
            type="text"
            name="hostname"
            @input="${(e)=>this.model.hostname = e.target.value}"
            .value="${this.model.hostname}"/>
        </div>
      </div>
      
      <div class="row line">
        <div class="row-col margin-right justify-right"></div>
        <div class="row-col">
          <input
            type="checkbox"
            name="use_acme"
            @change="${(e)=>{throw new Error("todo")}}"
            ?disabled="${!editing}"
            ?checked="${this.model.acmeAuthority !== "none"}"
          /> Use ACME to manage TLS</label>
        </div>
      </div>
      
      <div class="row line" id="acme_provider" style="display:${acme}">
        <div class="row-col margin-right justify-right">acme provider:</div>
        <div class="row-col">
          <span class="${this.visibleWhen("list", "pending")}">${this.model.acmeAuthority}</span>
          <input class="${this.visibleWhen("edit", "add")}"
              type="url"
              size="60"
              name="provider"
              @input="${(e)=>{this.model.acmeAuthority=e.target.value}}"
              .value="${this.model.acmeAuthority}"
              ?disabled="${!this.model.useAcme}"
            />
        </div>
      </div>
      
      <div class="row line" id="acme_email" style="display:${acme}">
        <div class="row-col margin-right justify-right">contact email:</div>
        <div class="row-col">
          <span class="${this.visibleWhen("list", "pending")}">${this.model.acmeEmail}</span>
          <input class="${this.visibleWhen("edit", "add")}"
            type="email"
            name="email"
            @input="${(e)=>{this.model.acmeEmail = e.target.value}}"
            .value="${this.model.acmeEmail}"
            ?disabled="${!this.model.useAcme}" />
        </div>
      </div>
      
       <div class="${tos} row line" id="acm_tos" style="display:${acme}">
        <div class="row-col margin-right justify-right"></div>
        <div class="row-col">
          <input
            type="checkbox"
            name="tos_agree"
            @change="${(e)=>this.agreed = e.target.checked}"
            ?disabled="${!this.isTOSShowing()}"
            ?checked="${this.agreed}" />
            <span>I have agreed to to the Terms of Service: ${this.model.getTermsOfService()}</span>
        </div>
      </div>
      
     <div class="row line">
        <div class="row-col margin-right justify-right ${this.visibleWhen("list", "edit", "pending")}">status:</div>
        <div class="row-col">
          <span class="${this.visibleWhen("list", "edit", "pending")}">${state} ${reason}</span>
        </div>
      </div>
      `

  }

}

/* Bind our custom elements to the HostView. */
customElements.define('dw-mvc-host', HostView);
