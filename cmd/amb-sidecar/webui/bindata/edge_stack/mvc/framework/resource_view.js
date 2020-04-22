/*
 * ResourceView
 * A View subclass that implements a generic view on a Resource model object.  It contains cached values of
 * its Resource model object, as well as state used for the different view variants (edit, add, etc.)
 */

import { html, css } from '../../vendor/lit-element.min.js'
import "../../vendor/js-yaml.min.js"

/* Object merge operation */
import { mapMerge, objectMerge } from "../framework/utilities.js"

/* View superclass */
import { View } from './view.js'
import { CoW } from './cow.js'

export class ResourceView extends View {

  /**
   * properties
   *
   * These are the properties of the ResourceView, which include the the underlying Resource, and also transient state
   * (e.g. validation messages).
   */
  static get properties() {
    return {
      model: {type: Object},
      messages:  {type: Array},
      showYAML:  {type: Boolean}
    }
  }

  static get styles() {
    return css`
      .error {
        color: red;
      }
      
      div.pending {
        background: repeating-linear-gradient(
          -45deg,
          #f8f8f8,
          #f8f8f8 10px,
          #eeeeee 10px,
          #eeeeee 20px
        );
      }

      * {
        margin: 0;
        padding: 0;
        border: 0;
        position: relative;
        box-sizing: border-box
      }
      
      *, textarea {
        vertical-align: top
      }
      
      .card {
        background: #fff;
        border-radius: 10px;
        padding: 10px 30px 10px 30px;
        box-shadow: 0 10px 5px -11px rgba(0, 0, 0, .6);
        width: 100%;
        -webkit-flex-direction: row;
        -ms-flex-direction: row;
        flex-direction: row;
        -webkit-flex: 1 1 1;
        -ms-flex: 1 1 1;
        flex: 1 1 1;
        margin: 30px 0 0;
        font-size: .9rem;
      }
      
      .card, .card .col .con {
        display: -webkit-flex;
        display: -ms-flexbox;
        display: flex
      }
      
      .card .col {
        -webkit-flex: 1 0 50%;
        -ms-flex: 1 0 50%;
        flex: 1 0 50%;
        padding: 0 30px 0 0
      }
      
      .card .col .con {
        margin: 10px 0;
        -webkit-flex: 1;
        -ms-flex: 1;
        flex: 1;
        -webkit-justify-content: flex-end;
        -ms-flex-pack: end;
        justify-content: flex-end;
        height: 30px
      }
      
      
      .card .col, .card .col .con label, .card .col2, .col2 a.cta .label {
        -webkit-align-self: center;
        -ms-flex-item-align: center;
        -ms-grid-row-align: center;
        align-self: center
      }
      
      .col2 a.cta  {
        text-decoration: none;
        border: 2px #efefef solid;
        border-radius: 10px;
        width: 90px;
        padding: 6px 8px;
        max-height: 35px;
        -webkit-flex: auto;
        -ms-flex: auto;
        flex: auto;
        margin: 10px auto;
        color: #000;
        transition: all .2s ease;
        cursor: pointer;
      }
      
      .col2 a.cta .label {
        text-transform: uppercase;
        font-size: .8rem;
        font-weight: 600;
        line-height: 1rem;
        padding: 0 0 0 10px;
        -webkit-flex: 1 0 auto;
        -ms-flex: 1 0 auto;
        flex: 1 0 auto
      }
      
      .col2 a.cta svg {
        width: 15px;
        height: auto
      }
      
      .col2 a.cta svg path, .col2 a.cta svg polygon {
        transition: fill .7s ease;
        fill: #000
      }
      
      .col2 a.cta:hover {
        color: #5f3eff;
        transition: all .2s ease;
        border: 2px #5f3eff solid
      }
      
      .col2 a.cta:hover svg path, .col2 a.cta:hover svg polygon {
        transition: fill .2s ease;
        fill: #5f3eff
      }
      
      .col2 a.cta {
        display: -webkit-flex;
        display: -ms-flexbox;
        display: flex;
        -webkit-flex-direction: row;
        -ms-flex-direction: row;
        flex-direction: row
      }
      
      .col2 a.off {
        display: none;
      }
      
      div.off, span.off, input.off, b.off {
        display: none;
      }
      
      .row {
        display: -webkit-flex;
        display: -ms-flexbox;
        display: flex;
        -webkit-flex-direction: row;
        -ms-flex-direction: row;
        flex-direction: row
      }
      
      .row .row-col {
        flex: 1;
        padding: 10px 5px;
      }
      .row .row-col:nth-child(1) {
        font-weight: 600;
      }
      .row .row-col:nth-child(2) {
        flex: 2;
      }
      
      .card .row {
        border-bottom: 1px solid rgba(0, 0, 0, .1);
      }
      
      .card div.line:nth-last-child(2) {
        border-bottom: none;
      }
      
      .errors ul {
        color: red;
      }
      
      .margin-right {
        margin-right: 20px;
      }
      .justify-right {
        text-align: right;
      }
      
      button, input, select, textarea {
        font-family: inherit;
        font-size: 100%;
        margin: 0
      }
      
      button, input {
        line-height: normal
      }
      
      button, html input[type=button], input[type=reset], input[type=submit] {
        -webkit-appearance: button;
        cursor: pointer
      }
      input[type=search] {
        -webkit-appearance: textfield;
        box-sizing: content-box
      }
      
      input {
        background: #efefef;
        padding: 5px;
        margin: -5px 0;
        width: 100%;
      }
      input[type=checkbox], input[type=radio] {
        box-sizing: border-box;
        padding: 0;
        width: inherit;
        margin: 0.2em 0;
      }
      textarea {
        background-color: #efefef;
        padding: 5px;
        width: 100%;
      }
      div.yaml-wrapper {
        overflow-x: scroll;
      }
      div.yaml-wrapper pre {
        width: 500px;
        font-size: 90%;
      }
      
      div.yaml ul {
        color: var(--dw-purple);
        margin-left: 2em;
        margin-bottom: 0.5em;
      }
      div.yaml li {
      }
      div.yaml li .yaml-path {
      }
      div.yaml li .yaml-change {
      }
      div.namespace {
        display: inline;
      }
      div.namespaceoff {
        display: none;
      }
      div.namespace-input {
        margin-top: 0.1em;
      }
      .pararen {
        -webkit-justify-content: center;
        -ms-flex-pack: center;
        justify-content: center;
        -webkit-align-content: center;
        -ms-flex-line-pack: center;
        align-content: center;
        -webkit-align-items: center;
        -ms-flex-align: center;
        align-items: center;
      }
      div.namespace-input .pararen {
        font-size: 1.2rem;
        padding: 0 5px;
        display: inline;
        top: -0.2em;
      }
      div.namespace-input input {
        width: 90%;
      }      
    }`
  }

  constructor() {
    super();

    // any validation/error messages
    this.messages = [];
    this.showYAML = false;
    this.viewState = "list";
  }

  /**
   * We implement our own getter/setter for messages so we can wrap
   * the value with a CoW proxy that automatically calls requestUpdate
   * on any change.
   */
  get messages() {
    return this._messages
  }

  set messages(value) {
    // By having a CoW wrapper, we can ensure requestUpdate is always
    // called when messages are added/removed.
    this._messages = new CoW(value, ()=>{
      this.requestUpdate("messages")
    })
  }

  /**
   * addMessage(message)
   *
   * Add a message to the messages list.  This list can be rendered along with the Resource information to display
   * errors, warnings, or other information.  Typically, messages will be added during validation, when the Save
   * operation is performed.  If any messages have been added they are displayed to the user rather than allowing the
   * save action to proceed.
   */
  addMessage(message) {
    this.messages.push(message)
  }

  /**
   * clearMessages()
   *
   * This method is called to clear the message list.
   */
  clearMessages() {
    this.messages  = [];
  }

  /**
   * onCancelButton()
   *
   * This method is called on the View when the View is in Edit mode, and the user clicks on the Cancel button to
   * discard the changes and return to the original state.
   */
  onCancelButton() {
    this.model.cancel()
    this.clearMessages()
  }

  /**
   * onDeleteButton()
   *
   * This method is called on the View when the user has clicked the Delete button to delete the Resource.  Like saving,
   * this switches to a pending mode until the Resource has been observed to be deleted.
   */

  onDeleteButton() {
    let proceed = confirm(`You are about to delete the ${this.model.kind} named '${this.model.name}' in the '${this.model.namespace}' namespace.\n\nAre you sure?`);

    if (proceed) {
      /* Ask the Resource to delete itself. */
      this.model.delete()
      this.model.save()
        .catch((error)=>{
          alert(`${this.model.kind} ${this.model.name} was unable to be deleted.  Backend not available?`);
          console.log("Resource.delete() returned error ${error");
        })
    }
  }

  /**
   * onEditButton()
   * This method is called on the View when the View needs to change to its Edit mode.
   */
  onEditButton() {
    this.model.edit()
  }

  /**
   * onSaveButton()
   *
   * This method is called on the View when the user clicks on the Save button to save the changes.  There are two
   * circumstances in which the Save button will be clicked. When adding a resource (viewState of "add"), or when
   * editing an existing resource (viewState of "save").
   *
   */
  onSaveButton() {
    this.clearMessages()

    /* Validate the data in the model. */
    let validationErrors = this.validate();

    if (validationErrors.size === 0) {
      this.model.save()
        .catch((e)=>{
          alert(`${this.model.kind} ${this.model.name} was unable to be saved.  Backend did not respond.`);
        })
    }
    /* Have validation errors.  Update the message list. */
    else {
      for (let [field, message] of validationErrors) {
        this.addMessage(`${field}: ${message}`);
      }
    }
  }

  /**
   * onSourceButton()
   *
   * This method opens a window on the Resource's source URI.
   */
  onSourceButton(mouseEvent) {
    window.open(this.model.sourceURI());

    /* Defocus the button */
    mouseEvent.currentTarget.blur();
  }

  /**
   * onYamlButton()
   *
   * This method hides and shows the current YAML of the Resource.
   */
  onYamlButton(mouseEvent) {
    /* Toggle showYAML */
    this.showYAML = !this.showYAML;

    /* Defocus the button */
    mouseEvent.currentTarget.blur();
  }

  /**
   * render()
   * This renders the entire ResourceView.  It requires five callback methods for buttons:
   *
   * - onSourceButton
   * - onEditButton
   * - onSaveButton
   * - onCancelButton
   * - onDeleteButton
   * - onYamlButton
   *
   * Subclasses need only define renderSelf() for the specific rendering of the individual resource class.
   */
  render() {
    let oldState = this.viewState
    if (this.model.isPending()) {
      this.viewState = "pending"
    } else if (this.model.isNew()) {
      this.viewState = "add"
    } else if (this.model.isModified()) {
      this.viewState = "edit"
    } else {
      this.viewState = "list"
    }

    if (oldState !== this.viewState) {
      if (["add", "edit"].includes(this.viewState)) {
        this._needsFocus = true
      }
    }

    /* If pending, show a crosshatch over the view content. */
    let pendingWrite  = !this.model.isDeleted() && this.model.isPending();
    let pendingDelete = this.model.isDeleted() && this.model.isPending();
    let pendingAny    = this.model.isPending();

    /* Return the HTML, including calls to renderSelf() to allow subclasses to specialize. */
    return html`
      <form>
        <div class="card ${this.viewState === "off" ? "off" : ""}">
          <div class="col">
            <div class="${pendingAny ? "pending" : ""}">
              <!-- Render common Resource fields: kind, name, namespace, as well as input fields when editing.   -->
              <div class="row line">
                <div class="row-col margin-right">${this.model.kind}:</div>
              </div>
              
              <div class="row line">
                <label class="row-col margin-right justify-right">name:</label>
                <div class="row-col">
                  <b class="${this.visibleWhen("list", "edit", "pending")}">${this.model.name}</b>
                  
                  <input class="${this.visibleWhen("add")}" name="name" type="text"
                         @input="${(e)=>{this.model.name = e.target.value}}"
                         .value="${this.model.name}"/>
                </div>
              </div>
              
              <div class="row line">
                <label class="row-col margin-right justify-right">namespace:</label>
                <div class="row-col">
                  <div class="namespace${this.visibleWhen("list", "edit", "pending")}">(${this.model.namespace})</div>
                  
                  <div class="namespace-input ${this.visibleWhen("add")}">
                    <div class="pararen">(</div>
                    <input class="${this.visibleWhen("add")}" name="namespace" type="text"
                           @input=${(e)=>{this.model.namespace = e.target.value}}
                           .value="${this.model.namespace}"/>
                    <div class="pararen">)</div>
                  </div>
                </div>
              </div>
  
              <!-- Render the customized HTML for subclasses of IResourceView, and any messages or YAML that should be displayed.  -->
          
              ${this.renderSelf()}
              ${this.renderMessages()}
              ${this.renderYAML()}
            </div>
          </div>
          
          <!-- Render buttons for showing sourceURI, editing, saving, deleting, and cancelling edits. -->
           
          <div class="col2">
          
            <a class="cta source ${typeof this.model.sourceURI() == 'string' ? "" : "off"}" @click=${(x)=>this.onSourceButton(x)}>
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 18.83 10.83"><defs><style>.cls-2{fill:none;stroke:#000;stroke-linecap:square;stroke-miterlimit:10;stroke-width:2px;}</style></defs><title>source_2</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><polyline class="cls-2" points="5.41 1.41 1.41 5.41 5.41 9.41"/><polyline class="cls-2" points="13.41 1.41 17.41 5.41 13.41 9.41"/></g></g></svg>
              <div class="label">source</div>
            </a>
            
            <a class="cta pending ${pendingDelete ? `` : `off`}">
              <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24"><path d="M14.078 7.061l2.861 2.862-10.799 10.798-3.584.723.724-3.585 10.798-10.798zm0-2.829l-12.64 12.64-1.438 7.128 7.127-1.438 12.642-12.64-5.691-5.69zm7.105 4.277l2.817-2.82-5.691-5.689-2.816 2.817 5.69 5.692z"/></svg>
              <div class="label">pending</div>
            </a>
            
            <a class="cta pending ${pendingWrite ? `` : `off`}">
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 16"><defs><style>.cls-1{fill-rule:evenodd;}</style></defs><title>delete</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><path class="cls-1" d="M24,16H7L0,8,7,0H24V16ZM7.91,2,2.66,8,7.9,14H22V2ZM14,6.59,16.59,4,18,5.41,15.41,8,18,10.59,16.59,12,14,9.41,11.41,12,10,10.59,12.59,8,10,5.41,11.41,4,14,6.59Z"/></g></g></svg>
              <div class="label">pending</div>
            </a>
            
            <a class="cta edit ${this.visibleWhen("list")}" @click=${()=>this.onEditButton()}>
              <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24"><path d="M14.078 7.061l2.861 2.862-10.799 10.798-3.584.723.724-3.585 10.798-10.798zm0-2.829l-12.64 12.64-1.438 7.128 7.127-1.438 12.642-12.64-5.691-5.69zm7.105 4.277l2.817-2.82-5.691-5.689-2.816 2.817 5.69 5.692z"/></svg>
              <div class="label">edit</div>
            </a>
            
            <a class="cta save ${this.visibleWhen("edit", "add")}" @click=${()=>this.onSaveButton()}>
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><title>Asset 1</title><g id="Layer_2" data-name="Layer 2"><g id="iconmonstr"><path id="save-2" d="M13,3h3V8H13ZM24,4V24H0V0H20ZM7,9H17V2H7ZM22,4.83,19.17,2H19v9H5V2H2V22H22Z"/></g></g></svg>
              <div class="label">save</div>
            </a>
            
            <a class="cta cancel ${this.visibleWhen("edit", "add")}" @click=${()=>this.onCancelButton()}>
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><title>cancel</title><g id="Layer_2" data-name="Layer 2"><g id="iconmonstr"><polygon id="x-mark-2" points="24 21.08 14.81 11.98 23.91 2.81 21.08 0 11.99 9.18 2.81 0.09 0 2.9 9.19 12.01 0.09 21.19 2.9 24 12.01 14.81 21.19 23.91 24 21.08"/></g></g></svg>
              <div class="label">cancel</div>
            </a>
            
            <a class="cta delete ${this.visibleWhen("list")}" @click=${()=>this.onDeleteButton()}>
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 16"><defs><style>.cls-1{fill-rule:evenodd;}</style></defs><title>delete</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><path class="cls-1" d="M24,16H7L0,8,7,0H24V16ZM7.91,2,2.66,8,7.9,14H22V2ZM14,6.59,16.59,4,18,5.41,15.41,8,18,10.59,16.59,12,14,9.41,11.41,12,10,10.59,12.59,8,10,5.41,11.41,4,14,6.59Z"/></g></g></svg>
              <div class="label">delete</div>
            </a>
            
            <a class="cta edit ${this.visibleWhen("list", "edit", "add")}" @click=${(e)=>this.onYamlButton(e)}>
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64" width="64" height="64"><title>zoom</title><g class="nc-icon-wrapper" stroke-linecap="square" stroke-linejoin="miter" stroke-width="2" fill="#000000" stroke="#000000"><line data-color="color-2" x1="59" y1="59" x2="42.556" y2="42.556" fill="none" stroke-miterlimit="10"/><circle cx="27" cy="27" r="22" fill="none" stroke="#000000" stroke-miterlimit="10"/></g></svg>
              <div class="label">yaml</div>
            </a>
            
          </div>
        </div>
      </form>
      `
  }

  /**
   * renderMessages()
   *
   * This renders the message list, if there are any messages to report for this particular resource.
   */

  renderMessages() {
    if (this.messages.length > 0) {
      return html`
        <div class="row line">
          <div class="row-col"></div>
          <div class="row-col errors">
            <ul>
              ${this.messages.map(m=>html`<li><span class="error">${m}</span></li>`)}
            </ul>
          </div>
        </div>`
    } else {
      return html``
    }
  }

  /**
   * renderYAML()
   *
   * This renders the YAML that would be applied to Kubernetes when the resource is added or saved.
   */
  renderYAML() {
    if (this.showYAML) {
      try {
        let yaml = jsyaml.safeDump(this.model.yaml)
        let entries = [];

        /* this is to show differences between the original and merged YAML.  Disabled for now.
        merged.diffs.forEach((v, k) => {
          if (v !== "ignored") {
            entries.push(html`<li><span class="yaml-path">${k}</span> <span class="yaml-change">${v}</span></li>`);
          }
        });
        */

        return html`
        <div class="yaml" id="merged-yaml">
          <div class="yaml-changes"><ul>
        ${entries}
          </ul></div>
          <div class="yaml-wrapper">
            <pre>${yaml}</pre>
          </div>
        </div>
        `;
      } catch (e) {
        return html`<pre>${e.stack}</pre>`;
      }
    }
    else {
      return html``;
    }
  }

  /**
   * updated()
   *
   * This method is a LitElement callback which is invoked when the element's DOM has been updated
   * and rendered.  Here it is used to focus and select the first enabled and visible field when
   * being edited.
   */
  updated(changedProperties) {
    if (this._needsFocus) {
      this.focus()
      this._needsFocus = false;
    }

    // css doesn't seem to have a way to select the last *visible*
    // child, so we do it here
    let lastVisible = null
    let last = null
    for (let el of this.shadowRoot.querySelectorAll(".card .line")) {
      if (el.offsetParent != null) {
        lastVisible = el
      }
      last = el
    }
    if (lastVisible) {
      lastVisible.style.borderBottom = "none"
    } else {
      // This probably means we are being rendered even though we
      // aren't visible. The best we can do is take a guess on the
      // last one.
      last.style.borderBottom = "none"
    }
  }

  focus() {
    for (let el of this.shadowRoot.querySelectorAll("input")) {
      if (!el.disabled && el.offsetParent !== null) {
        el.focus()
        el.select()
        return
      }
    }
  }

  /** validate()
   * 
   * This method is invoked on save in order to validate input prior to proceeding with the save action.  The model
   * validates its current state, so anything that the View wants to validate must already be in the model.
   *
   * validate() returns a Map of fieldnames and error strings. If the dictionary is empty, there are no errors.
   *
   * For now we will have a side-effect of validate in that any errors will be added to the message list.
   */
  validate() {
    let errors = this.model.validate() || new Map();

    /* Allow subclasses to validate as well. */
    errors = mapMerge(errors, this.validateSelf());

    return errors;
  }

  /**
   * visibleWhen(...arguments)
   *
   * return the empty string if the current viewState is listed in the arguments, "off" otherwise.
   */
  visibleWhen() {
    return [...arguments].includes(this.viewState) ? "" : "off";
  }

}
