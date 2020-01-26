/*
 * HostView
 * A ResourceView subclass that implements a view on a Host  It contains cached values of
 * its Resource model object, as well as state used for the different view variants (edit, add, etc.)
 * and adds new properties for acmeProvider, acmeEmail, tos (terms of service), and whether to show tos.
 */

import { html } from '../../vendor/lit-element.min.js'

/* Object merge operation */
import { objectMerge } from "../framework/utilities.js"

/* ResourceView interface class */
import { IResourceView } from '../interfaces/iresource_view.js'

export class HostView extends IResourceView {

  /* ====================================================================================================
   *  These functions and methods implement the IResourceView interface.
   * ====================================================================================================
   */

  /* properties
   * These are the properties of the HostView, which reflect the properties of the underlying Resource,
   * and also include transient state (e.g. viewState). LitElement manages these declared properties and
   * provides various services depending on how they are used.  For further details on LitElement, see
   * https://lit-element.polymer-project.org/guide/properties
   */

  static get properties() {
    let myProperties = {
      hostname: {type: String},     // Host
      acmeProvider: {type: String}, // Host
      acmeEmail: {type: String},    // Host
      useAcme: {type: Boolean},     // HostView
      tos: {type: String}           // HostView
    };

    /* Merge my properties with those defined by my superclasses. */
    return objectMerge(myProperties, IResourceView.properties);
  }

  /* constructor(model)
   * The IResourceView constructor, which takes a Resource (model) as its parameter.
   */

  constructor(model) {
    super(model);

    /* Cache state from the model. Don't call
    *  readFromModel yet, since that updates the UI
    *  which hasn't been instantiated.
    */
    this.hostname     = model.hostname;
    this.useAcme      = model.useAcme;
    this.acmeProvider = model.acmeProvider;
    this.acmeEmail    = model.acmeEmail;

    /* The Host object has a Terms of Service checkbox. Once the user has agreed to the TOS, we no longer
     * show the checkbox or link in the Host detail display.
     */
    this.tos = html`...`;
  }

  /* readSelfFromModel()
   * This method is called on the View when the View needs to match the current state of its Model.
   * Generally this happens during initialization and during editing when the Cancel button is pressed and the
   * View reverts to displaying the original Model's state.  The ResourceView assumes that the HostView
   * has nameInput() and namespaceInput()
   */

  readSelfFromModel() {
    /* Get the values from the model. */
    this.hostname = this.model.hostname;
    this.acmeProvider = this.model.acmeProvider;
    this.acmeEmail = this.model.acmeEmail;
    this.useAcme = this.model.useAcme;

    /* Set the fields of the form.  The DOM must be generated before calling readFromModel. */
    this.hostnameInput().value = this.hostname;
    this.acmeEmailInput().value = this.acmeEmail;
    this.acmeProviderInput().value = this.acmeProvider;
    this.tosAgreeCheckbox().value  = this.useAcme;
    this.useAcmeCheckbox().checked = this.useAcme;
  }

  /* writeSelfToModel()
   * This method is called on the View when the View has new, validated state that should be written back
   * to the Model.  This happens during a Save operation after the user has modified the View.
   */

  writeSelfToModel() {
    /* Get the values from the form.  The DOM must be generated before calling writeToModel. */
    this.hostname = this.hostnameInput().value;
    this.acmeEmail = this.acmeEmailInput().value;
    this.acmeProvider = this.acmeProviderInput().value;
    this.useAcme = this.useAcmeCheckbox().checked;

    /* Write back to the model */
    this.model.hostname = this.hostname;
    this.model.acmeProvider = this.acmeProvider;
    this.model.acmeEmail = this.acmeEmail;
    this.model.useAcme = this.useAcme;
  }

  /* validateSelf()
   * This method is invoked on save in order to validate input prior to proceeding with the save action.
   * The model validates its current state, so anything that the View wants to validate must already be in the model.
   *
   * validateSelf() returns a Map of fieldnames and error strings. If the dictionary is empty, there are no errors.
   *
   * For now we will have a side-effect of validate in that any errors will be added to the message list.
   */

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

  /* renderSelf()
  * This method renders the Host view within the HTML framework set up by ResourceView.render().
  */

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
            ?disabled="${!this.isTOSShowing()}" />
            <span>I have agreed to to the Terms of Service at: ${this.tos}</span>
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

  /* onACMECheckbox
   */

  onACMECheckbox() {
    /* Write back to the model for validation. */
    this.writeToModel();

    /* Is the ACME information being shown now?  If so, and we have a valid URL for an ACME provider,
     * fetch the terms of service and uncheck the "I have agreed to the Terms of Service" checkbox.
     */

    if (this.useAcme) {
      this.tos = this.model.fetchTOS();
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
    /* Write back to the model for validation. */
    this.writeToModel();

    /* Note that we have to check */

    /* The email changed, update the YAML if showing. */
    if (this.showYAML) {
      this.yamlElement().requestUpdate();
    }
  }

  /* onHostnameChanged()
   * This is called when the hostname field changes in an Edit or Add dialog to check if the new hostname can
   * be used with ACME. If it can be, we check the checkbox, otherwise we uncheck it.
   */

  onHostnameChanged() {
    this.writeToModel();

    /* update the YAML if showing. */
    if (this.showYAML) {
      this.yamlElement().requestUpdate();
    }
  }

  /* onProviderChanged()
   * The ACME provider has been changed by the user.  Write back to the model, uncheck the TOS agreement checkbox,
   * and then fetch the terms of service for the new provider.
   */
  onProviderChanged() {
    this.writeToModel();
    this.tosAgreeCheckbox().checked = false;
    this.tos = this.model.fetchTOS();

    /* update the YAML if showing. */
    if (this.showYAML) {
      this.yamlElement().requestUpdate();
    }
  }

  /* onTOSAgreeCheckbox
   * Toggle the agree
   */

  onTOSAgreeCheckbox() {
    /* nothing needed, just check the value when desired. */
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
