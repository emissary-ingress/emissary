/*
 * HostView
 * A ResourceView subclass that implements a view on a Host  It contains cached values of
 * its Resource model object, as well as state used for the different view variants (edit, add, etc.)
 * and adds new properties for acmeProvider, acmeEmail, tos (terms of service), and whether to show tos.
 */

/* Set merge operation */
import { mapMerge } from "./map.js"

/* ResourceView interface class */
import { IResourceView } from './iresource_view.js'

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
      hostname:     {type: String},   // Host
      acmeProvider: {type: String},   // Host
      acmeEmail:    {type: String},   // Host
      useAcme:      {type: Boolean},  // HostView
      tos:          {type: String},   // HostView
      showTos:      {type: Boolean}   // HostView

    };

    /* Merge my properties with those defined by my superclasses. */
    return mapMerge(myProperties, ResourceView.properties);
  }

  /* constructor(model)
   * The IResourceView constructor, which takes a Resource (model) as its parameter.
   */

  constructor(model) {
    super(model);

    /* The Host object has a Terms of Service checkbox. Once the user has agreed to the TOS, we no longer
     * show the checkbox or link in the Host detail display.
     */
    this.tos     = html`...`;
    this.showTos = false;
  }

  /* readSelfFromModel()
   * This method is called on the View when the View needs to match the current state of its Model.
   * Generally this happens during initialization and during editing when the Cancel button is pressed and the
   * View reverts to displaying the original Model's state.  The ResourceView assumes that the HostView
   * has nameInput() and namespaceInput()
   */

  readSelfFromModel() {
    this.acmeProvider = model.acmeProvider;
    this.acmeEmail    = model.acmeEmail;
    this.useAcme      = model.useAcme;
  }

  /* writeSelfToModel()
   * This method is called on the View when the View has new, validated state that should be written back
   * to the Model.  This happens during a Save operation after the user has modified the View.
   */

  writeSelfToModel() {
    throw new Error("please implement ${this.constructor.name}.writeSelfToModel()")
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
    throw new Error("please implement ${this.constructor.name}.validateSelf()")
  }

  /* renderSelf()
  * This method renders the Host view.
  */

  /* need:
  this.visible
  this.onProviderChanged

   */
  renderSelf() {
    let host       = this.model;
    let spec       = host.getSpec();
    let status     = host.status || {"state": "<none>"};
    let hostState  = status.state;
    let reason     = (hostState === "Error") ? `(${status.reason})` : '';
    let tos        = this.isTOSshowing() ? "attribute-value" : "off";
    let editing =    this.viewState === "add" || this.viewState === "edit";

    return html`
      <div class="row line">
        <div class="row-col margin-right justify-right">hostname:</div>
        <div class="row-col">
          <span class="${this.visible("list")}">${this.hostname}</span>
          <input class="${this.visible("edit", "add")}" type="text" name="hostname"  value="${this.hostname}"/>
        </div>
      </div>
      
      <div class="row line">
        <div class="row-col margin-right justify-right"></div>
        <div class="row-col">
          <input type="checkbox"
            name="use_acme"
            ?disabled="${!editing}"
            ?checked="${spec.acmeProvider.authority !== "none"}"
          /> Use ACME to manage TLS</label>
        </div>
      </div>
      
      <div class="row line">
        <div class="row-col margin-right justify-right">acme provider:</div>
        <div class="row-col">
          <span class="${this.visible("list")}">${this.acmeProvider}</span>
          <input
              class="${this.visible("edit", "add")}"
              type="url"
              size="60"
              name="provider"
              value="${this.acmeProvider}"
              @change="${this.onProviderChanged.bind(this)}"
              ?disabled="${!this.useAcme()}"
            />
        </div>
      </div>
      
      <div class="${tos} row line">
        <div class="row-col margin-right justify-right"></div>
        <div class="row-col">
          <input type="checkbox" name="tos_agree" ?disabled="${!this.useAcme}" />
            <span>I have agreed to to the Terms of Service at: ${this.tos}</span>
        </div>
      </div>
      
      <div class="row ${this.viewState !== "add" ? "line" : ""}">
        <div class="row-col margin-right justify-right">email:</div>
        <div class="row-col">
          <span class="${this.visible("list")}">${this.acmeEmail}</span>
          <input class="${this.visible("edit", "add")}" type="email" name="email" value="${this.acmeEmail}" ?disabled="${!this.useAcme}" />
        </div>
      </div>
      
      <div class="row line">
        <div class="row-col margin-right justify-right ${this.visible("list", "edit")}">status:</div>
        <div class="row-col">
          <span class="${this.visible("list", "edit")}">${hostState} ${reason}</span>
        </div>
      </div>
      `
  }

  /* ====================================================================================================
   *  These methods are specific to the HostView and extend IResourceView.
   * ====================================================================================================
   */

  /* Accessors for querySelectors */

  hostnameInput() {
    return this.shadowRoot.querySelector('input[name="hostname"]')
  }

  useAcmeCheckbox() {
    return this.shadowRoot.querySelector('input[name="use_acme"]')
  }

  acmeProviderInput() {
    return this.shadowRoot.querySelector('input[name="provider"]')
  }

  tosAgreeCheckbox() {
    return this.shadowRoot.querySelector('input[name="tos_agree"]')
  }

  acmeEmailInput() {
    return this.shadowRoot.querySelector('input[name="email"]')
  }

}

/* Bind our custom elements to the HostView. */
customElements.define('dw-host', HostView);
