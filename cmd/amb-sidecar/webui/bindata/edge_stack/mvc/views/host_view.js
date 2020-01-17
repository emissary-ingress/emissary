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

const _defaultAcmeProvider = "https://acme-v02.api.letsencrypt.org/directory";
const _defaultAcmeEmail    = "yourname@yourcompany.com";


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
    this.showTos = false;
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
    this.useAcmeCheckbox().value = this.useAcme;
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
    this.useAcme = this.useAcmeCheckbox().value;

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

    /*
     * We validate that the user has agreed to the Terms of Service,
     * which is either: (i) if we are not showing the Terms of Service,
     * then we assume that they have already agreed, or (ii) if we are
     * showing the TOS, then the checkbox needs to be checked.
     */
    if (this.useAcme && this.showTos && !this.tosAgreeCheckbox().checked) {
      errors.set("tos", "You must agree to terms of service");
    }

    /* Validate the user's email address.  The model is responsible
     * for correctly validating the input value.
     */

    if (this.model.validateEmail(this.acmeEmailInput().value)) {
      errors.set("acmeEmail", "That doesn't look like a valid email address");
    }

    return errors;
  }

  /* renderSelf()
  * This method renders the Host view within the HTML framework set up by ResourceView.render().
  */

  renderSelf() {
    let host = this.model;
    let acmeEmail = host.acmeEmail;
    let acmeProvider = host.acmeProvider;

    let status = host.status || {"state": "<none>"};
    let hostState = status.state;
    let reason = (hostState === "Error") ? `(${status.errorReason})` : '';
    let tos = this.isTOSShowing() ? "attribute-value" : "off";
    let editing = this.viewState === "add" || this.viewState === "edit";

    return html`
      <div class="row line">
        <div class="row-col margin-right justify-right">hostname:</div>
        <div class="row-col">
          <span class="${this.visibleWhen("list", "pending-delete")}">${this.hostname}</span>
          <input class="${this.visibleWhen("edit", "add")}"
            type="text"
            name="hostname"
            @change="${this.onHostnameChanged.bind(this)}"
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
            ?checked="${acmeProvider.authority !== "none"}"
          /> Use ACME to manage TLS</label>
        </div>
      </div>
      
      <div class="row line">
        <div class="row-col margin-right justify-right">acme provider:</div>
        <div class="row-col">
          <span class="${this.visibleWhen("list", "pending-delete")}">${this.acmeProvider}</span>
          <input class="${this.visibleWhen("edit", "add")}"
              type="url"
              size="60"
              name="provider"
              @change="${this.onProviderChanged.bind(this)}"
              value="${this.acmeProvider}"
              ?disabled="${!this.useAcme}"
            />
        </div>
      </div>
      
      <div class="${tos} row line">
        <div class="row-col margin-right justify-right"></div>
        <div class="row-col">
          <input
            type="checkbox"
            name="tos_agree"
            @change="${this.onTOSAgreeCheckbox.bind(this)}"
            ?disabled="${!this.useAcme}" />
            <span>I have agreed to to the Terms of Service at: ${this.tos}</span>
        </div>
      </div>
      
      <div class="row ${this.viewState !== "add" ? "line" : ""}">
        <div class="row-col margin-right justify-right">email:</div>
        <div class="row-col">
          <span class="${this.visibleWhen("list", "pending-delete")}">${this.acmeEmail}</span>
          <input class="${this.visibleWhen("edit", "add")}"
            type="email"
            name="email"
            @change="${this.onEmailChanged.bind(this)}"
            value="${this.acmeEmail}"
            ?disabled="${!this.useAcme}" />
        </div>
      </div>
      
      <div class="row line">
        <div class="row-col margin-right justify-right ${this.visibleWhen("list", "edit", "pending-delete")}">status:</div>
        <div class="row-col">
          <span class="${this.visibleWhen("list", "edit", "pending-delete")}">${hostState} ${reason}</span>
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

  /* ================================ Callback Functions ================================ */

  /* onACMECheckbox
   * TODO: When the checkbox changes, either hide or show the ACME
   * provider, TOS checkbox, and email fields.
   */

  onACMECheckbox() {
    this.useAcme = this.useAcmeCheckbox().checked;
    this.setAcmeFields(this.useAcme);
  }

  /* Email address has changed.  Because state is not a property,
  * we have to manually request the update.
  */
  onEmailChanged() {
    /* Write back to the model for validation. */
    this.writeToModel();

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
    /* TODO: let the model check, and return whether it is valid. */

    /* Write back to the model for validation.
     * TODO: at some point, writing back view data to the model
     * will be on timer and this will no longer be necessary.
     */
    this.writeToModel();

    /* update the YAML if showing. */
    if (this.showYAML) {
      this.yamlElement().requestUpdate();
    }
  }

  /* onProviderChanged()
   * The ACME provider has been changed by the user.  Fetch the terms of service for the new provider,
   * and write back to the model.
   */
  onProviderChanged() {
    this.showTos = true;
    this.fetchTermsOfService(this.acmeProviderInput().value);

    /* Write back to the model for validation.
     * NOTE: will be on timer and this
     * will no longer be necessary.
     */
    this.writeToModel();

    /* update the YAML if showing. */
    if (this.showYAML) {
      this.yamlElement().requestUpdate();
    }
  }

  /* onTOSAgreeCheckbox
    * TODO: When the checkbox changes, either hide or show the ACME
    * provider, TOS checkbox, and email fields.
    */

  onTOSAgreeCheckbox() {
    this.tosAgreed = this.tosAgreeCheckbox().checked;
  }

  /* ================================ Utility Functions ================================ */

  /* fetchTermsOfService()
   * Here we get the Terms of Service url from the ACME provider so that we can show it to the user. We do this
   * by calling an API on AES that then turns around and calls an API on the ACME provider. We cannot call the API
   * on the ACME provider directly due to CORS restrictions.
   */

  fetchTermsOfService(acmeProviderValue) {
    /* TODO: Let the model do this */
  }

  /* isTOSShowing()
   * Are the terms of service being shown during an Add operation?
   */

  isTOSShowing() {
    return (this.showTos || this.viewState === "add") && this.useAcme;
  }

  /* setAcmeFields(enabled)
   * Enable or disable the acmeProvider, acmeEmail, and tosAgree widgets,
   * depending on whether the Acme checkbox is set.
   */

  setAcmeFields(enabled) {
    if (enabled) {
      /* TODO: Use previous values of acmeProvider or acmeEmail if we have a saved copy during editing */
      this.acmeProviderInput().value =  _defaultAcmeProvider;
      this.acmeEmailInput().value =  _defaultAcmeEmail;
    } else {
      /* Disabled, set the values to empty. */
      this.acmeProviderInput().value = "";
      this.acmeEmailInput().value = "";
    }

    /* Make sure the model reflects the view. */
    this.writeToModel();
  }
}

/* Bind our custom elements to the HostView. */
customElements.define('dw-mvc-host', HostView);
