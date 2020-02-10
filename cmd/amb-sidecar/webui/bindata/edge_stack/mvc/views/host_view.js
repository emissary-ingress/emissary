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

  /* extend. See the explanation in IResourceView. */
  static get properties() {
    let myProperties = {
      hostname: {type: String},     // Host
      acmeProvider: {type: String}, // Host
      acmeEmail: {type: String},    // Host
      useAcme: {type: Boolean},     // HostView
      tos: {type: String},          // HostView
      agreed: {type: Boolean}       // HostView
    };
    return objectMerge(myProperties, IResourceView.properties);
  }

  /* extend */
  constructor(model) {
    super(model);

    /* see comment in IResourceView */
    this.hostname     = model.hostname;
    this.acmeProvider = model.acmeProvider;
    this.acmeEmail    = model.acmeEmail;
    this.useAcme      = model.useAcme;
    this.tos          = model.getTermsOfService();
    this.agreed       = model.agreed_terms_of_service;
  }

  /* override */
  readSelfFromModel() {
    /* Get the values from the model. */
    this.hostname = this.model.hostname;
    this.acmeProvider = this.model.acmeProvider;
    this.acmeEmail = this.model.acmeEmail;
    this.useAcme = this.model.useAcme;
    this.tos = this.model.getTermsOfService();
    this.agreed = this.model.agreed_terms_of_service;

    /* Set the fields of the form.  The DOM must be generated before calling readFromModel. */
    this.hostnameInput().value = this.hostname;
    this.acmeEmailInput().value = this.acmeEmail;
    this.acmeProviderInput().value = this.acmeProvider;
    this.tosAgreeCheckbox().checked  = this.agreed;
    this.useAcmeCheckbox().checked = this.useAcme;
  }

  /* override */
  writeSelfToModel() {
    /* Get the values from the form.  The DOM must be generated before calling writeToModel. */
    this.hostname = this.hostnameInput().value;
    this.acmeEmail = this.acmeEmailInput().value;
    this.acmeProvider = this.acmeProviderInput().value;
    this.useAcme = this.useAcmeCheckbox().checked;
    this.agreed = this.tosAgreeCheckbox().checked;

    /* Write back to the model */
    this.model.hostname = this.hostname;
    this.model.setAcmeProvider(this.acmeProvider);
    this.model.acmeEmail = this.acmeEmail;
    this.model.useAcme = this.useAcme;
    this.model.agreed_terms_of_service = this.agred;

  }

  /* override */
  validateSelf() {
    let errors = new Map();

    /* Validate that the user has agreed to the Terms of Service, which is either:
     * (i) if  not showing the Terms of Service, then assume that they have already agreed, or
     * (ii) if the TOS is shown, then the checkbox needs to be checked.
     */
    if (this.useAcme && this.isTOSShowing() && !this.tosAgreeCheckbox().checked) {
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
    let acme    = (this.useAcme ? "": "none");
    let tos     = this.isTOSShowing() ? "attribute-value" : "off";
    let editing = this.viewState === "add" || this.viewState === "edit";

    return html`
      <div class="row line">
        <div class="row-col margin-right justify-right">hostname:</div>
        <div class="row-col">
          <span class="${this.visibleWhen("list", "pending")}">${this.hostname}</span>
          <input class="${this.visibleWhen("edit", "add")}"
            type="text"
            name="hostname"
            @input="${this.onHostnameChanged.bind(this)}"
            value="${this.hostname}"/>
        </div>
      </div>
      
      <div class="row line">
        <div class="row-col margin-right justify-right"></div>
        <div class="row-col">
          <input
            type="checkbox"
            name="use_acme"
            @change="${this.onACMECheckbox.bind(this)}"
            ?disabled="${!editing}"
            ?checked="${this.acmeProvider !== "none"}"
          /> Use ACME to manage TLS</label>
        </div>
      </div>
      
      <div class="row line" id="acme_provider" style="display:${acme}">
        <div class="row-col margin-right justify-right">acme provider:</div>
        <div class="row-col">
          <span class="${this.visibleWhen("list", "pending")}">${this.acmeProvider}</span>
          <input class="${this.visibleWhen("edit", "add")}"
              type="url"
              size="60"
              name="provider"
              @input="${this.onProviderChanged.bind(this)}"
              value="${this.acmeProvider}"
              ?disabled="${!this.useAcme}"
            />
        </div>
      </div>
      
      <div class="row line" id="acme_email" style="display:${acme}">
        <div class="row-col margin-right justify-right">contact email:</div>
        <div class="row-col">
          <span class="${this.visibleWhen("list", "pending")}">${this.acmeEmail}</span>
          <input class="${this.visibleWhen("edit", "add")}"
            type="email"
            name="email"
            @input="${this.onEmailChanged.bind(this)}"
            value="${this.acmeEmail}"
            ?disabled="${!this.useAcme}" />
        </div>
      </div>
      
       <div class="${tos} row line" id="acm_tos" style="display:${acme}">
        <div class="row-col margin-right justify-right"></div>
        <div class="row-col">
          <input
            type="checkbox"
            name="tos_agree"
            @change="${this.onTOSAgreeCheckbox.bind(this)}"
            ?disabled="${!this.isTOSShowing()}"
            ?checked="${this.agreed}" />
            <span>I have agreed to to the Terms of Service: ${this.tos}</span>
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

  /* ================================ QuerySelector Accessors ================================ */

  acmeEmailInput() {
    return this.shadowRoot.querySelector('input[name="email"]')
  }

  acmeProviderInput() {
    return this.shadowRoot.querySelector('input[name="provider"]')
  }

  hostnameInput() {
    return this.shadowRoot.querySelector('input[name="hostname"]')
  }

  tosAgreeCheckbox() {
    return this.shadowRoot.querySelector('input[name="tos_agree"]')
  }

  useAcmeCheckbox() {
    return this.shadowRoot.querySelector('input[name="use_acme"]')
  }

  acmeProviderDiv() {
    return this.shadowRoot.getElementById("acme_provider");
  }

  acmeTOSDiv() {
    return this.shadowRoot.getElementById("acme_tos");
  }

  acmeEmailDiv() {
    return this.shadowRoot.getElementById("acme_email");

  }

  /* ================================ Callback Functions ================================ */

  onACMECheckbox() {
    this.writeToModel();

    /* Is the ACME information being shown now?  If so, and we have a valid URL for an ACME provider,
     * fetch the terms of service and uncheck the "I have agreed to the Terms of Service" checkbox.
     */
    if (this.useAcme) {
      this.tos = this.model.getTermsOfService();
      this.tosAgreeCheckbox().checked = false;
    }

    /* Request an update, since the Acme checkbox controls the visibility of the ACME provider, contact email,
     * and TOS agreement checkbox; need the update on either hide or show.
     */
    this.requestUpdate();
  }

  /* Email address has changed.  Because state is not a property,
  * we have to manually request the update.
  */
  onEmailChanged() {
    this.writeToModel();
  }

  onHostnameChanged() {
    this.writeToModel();
  }

  onProviderChanged() {
    this.writeToModel();

    /* When the ACME provider changes, we assume that the user has not agreed to the
     * new provider's terms of service. Also we have to get those new terms of service
     * to display to the user. */
    this.tosAgreeCheckbox().checked = false;
    this.tos = this.model.getTermsOfService();
  }

  onTOSAgreeCheckbox() {
    /* nothing needed here as the verify function checks the DOM element directly */
  }

  /* ================================ Utility Functions ================================ */

  /* isTOSShowing()
   * Are the terms of service being shown during an Add operation?
   */
  isTOSShowing() {
    return this.useAcme && (this.viewState === "add" || this.viewState === "edit");
  }
}

/* Bind our custom elements to the HostView. */
customElements.define('dw-mvc-host', HostView);
