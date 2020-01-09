/*
 * ResourceView
 * A View subclass that implements a generic view on a Resource model object.  It contains cached values of
 * its Resource model object, as well as state used for the different view variants (edit, add, etc.)
 */

/* Map merge operation */
import { mapMerge } from "./map.js"

/* View superclass */
import { View } from './view.js'

export class ResourceView extends View {

  /* properties
   * These are the properties of the ResourceView, which reflect the properties of the underlying Resource,
   * and also include transient state (e.g. viewState). LitElement manages these declared properties and
   * provides various services depending on how they are used.  For further details on LitElement, see
   * https://lit-element.polymer-project.org/guide/properties
   */

  static get properties() {
    let myProperties =  {
      kind:      {type: String},  // Resource state
      name:      {type: String},  // Resource state
      namespace: {type: String},  // Resource state
      showYAML:  {type: Boolean}  // ResourceView
    };

    /* Merge ResourceView properties with the View's properties. */
    return mapMerge(myProperties, View.properties);
  }

  /* constructor
   * The ResourceView constructor, which takes a Resource (model) as its parameter.
   * We cache the state from the model in the view its itself, as properties.
   * Because this is a web component, the property updates queue the appropriate re-rendering at the correct time.
   */

  constructor(model) {
    super(model);

    /* Cache state from the model. */
    this.kind      = model.kind;
    this.name      = model.name;
    this.namespace = model.namespace;
    this.status    = model.status;

    /* Since we are managing a Resource view, we may have messages to display, and optional YAML */
    this.messages = [];
    this.showYAML = false;
  }

  /* addMessage(message)
 * Add a message to the messages list.  This list can be rendered along with the Resource information to
 * display errors, warnings, or other information.  Typically, messages will be added during validation,
 * when the Save operation is performed.  If any messages have been added they are displayed to the user
 * rather than allowing the save action to proceed.
 */
  addMessage(message) {
    this.messages.push(message)
  }

  /* clearMessages()
   * This method is called to clear the message list.
   */
  clearMessages() {
    this.messages  = [];
  }

  /* nameInput()
   * This method returns the name input field, referenced below in the render() HTML.
   */

  nameInput() {
    return this.shadowRoot.querySelector(`input[name="name"]`);
  }

  /* namespaceInput()
   * This method returns the namespace input field, referenced below in the render() HTML.
   */

  namespaceInput() {
    return this.shadowRoot.querySelector(`input[name="namespace"]`);
  }

  /* onEdit()
   * This method is called on the View when the View needs to change to its Edit mode.  The View needs
   * to create a new copy of its Model for editing, and stop listening to any updates to the old Model.
   */

  onEdit() {
    throw Error("Not Yet Implemented");

    /* Save the View's existing model and stop listening to it. */
    this.savedModel = this.model;
    this.model.removeListener(this);

    /* Create a new model for editing, based on the state in the existing model, and start listening to it. */
    this.model = this.model.copySelf();
    this.model.addListener(this);

    /* Change to "edit" state. */
    this.viewState = "edit";
  }

  /* onSave()
   * This method is called on the View when the View is in Edit mode, and the user clicks on the
   * Save button to save the changes.  Ask the modified Model to save its state, however it needs to do that.
   * in the case of a Resource it will write back to Kubernetes with kubectl apply.
   */

  onSave(cookie) {
    throw Error("Not Yet Implemented");

    if (this.viewState === "add") {
      /* Add the new resource to the system. */
      this.model.doAdd(cookie);

      /* Remove the view.  The next snapshot will create a new Resource which then will create a corresponding
       * View that represents the added resource.  In the future, we will not remove the view but change it to a
       * "pending" state.
       */
      this.parentElement.removeChild(this);
    }
    else
    if (this.viewState === "edit") {
      /* Save the changes in the resource. */
      this.model.doSave(cookie);

      /* Swap models back, restoring listeners to the saved Model. */
      this.model.removeListener(this);
      this.model = this.savedModel;
      this.model.addListener(this);

      /* Now wait for the system to come back and update the Model, which will update the View.
       * In the future we will leave the View state as it was when edited, and watch for the
       * snapshot data to confirm the change.
       */
    }
  }

  /* onCancel()
   * This method is called on the View when the View is in Edit mode, and the user clicks on the
   * Cancel button to discard the changes and return to the original state.
   */

  onCancel() {
    throw Error("Not Yet Implemented");
  }


  /* readFromModel()
   * This method is called on the View when the View needs to match the current state of its Model.
   * Generally this happens during initialization and during editing when the Cancel button is pressed and the
   * View reverts to displaying the original Model's state.
   */

  readFromModel() {
    this.clearMessages();

    /* Get the name and namespace from the model */
    this.name      = this.model.name;
    this.namespace = this.model.namespace;

    /* Set the edit fields */
    this.nameInput().value      = this.name;
    this.namespaceInput().value = this.namespace;

    /* Allow subclasses to read their state from the model. */
    this.readSelfFromModel();
  }

  /**
   * This method is called on the View when the View has new, validated state that should be written back
   * to the Model.  This happens during a Save operation after the user has modified the View.
   */

  writeToModel() {
    /* Get the new values from the form */
    this.name      = this.nameInput().value;
    this.namespace = this.namespaceInput().value;

    /* Write back to the model. */
    this.model.name      = this.name;
    this.model.namespace = this.namespace;

    /* Allow subclasses to write their state to the model. */
    this.writeSelfToModel();
  }

  /* render()
  * This renders the entire ResourceView.  It requires five callback methods for buttons:
  * - onSource
  * - onEdit
  * - onSave
  * - onCancel
  * - onDelete
  * - onYAML
  *
  * Subclasses need only define renderSelf() for the specific rendering of the individual resource class.
  *
   */
  render() {
    return html`
      <link rel="stylesheet" href="../styles/oneresource.css">
      ${this.modifiedStyles() ? this.modifiedStyles() : ""}
      <form>
        <div class="card ${this.state.mode === "off" ? "off" : ""}">
          <div class="col">
            <div class="row line">
              <div class="row-col margin-right">${this.kind}:</div>
            </div>
            <div class="row line">
              <label class="row-col margin-right justify-right">name:</label>
              <div class="row-col">
                <b class="${this.visibleWhen("list", "edit")}">${this.name}</b>
                <input class="${this.visibleWhen("add")}" name="name" type="text" value="${this.name}"/>
              </div>
            </div>
            <div class="row line">
              <label class="row-col margin-right justify-right">namespace:</label>
              <div class="row-col">
                <div class="namespace${this.visibleWhen("list", "edit")}">(${this.namespace})</div>
                <div class="namespace-input ${this.visibleWhen("add")}"><div class="pararen">(</div><input class="${this.visibleWhen("add")}" name="namespace" type="text" value="${this.namespace}"/><div class="pararen">)</div></div>
              </div>
            </div>
      
          ${this.renderSelf()}
          ${this.renderMessages()}
          <!-- ${this.renderMergedYaml()} -->
      
          </div>
          <!-- Disable buttons for now
          
          <div class="col2">
            <a class="cta source ${typeof this.sourceURI() == 'string' ? "" : "off"}" @click=${(x)=>this.onSource(x)}>
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 18.83 10.83"><defs><style>.cls-2{fill:none;stroke:#000;stroke-linecap:square;stroke-miterlimit:10;stroke-width:2px;}</style></defs><title>source_2</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><polyline class="cls-2" points="5.41 1.41 1.41 5.41 5.41 9.41"/><polyline class="cls-2" points="13.41 1.41 17.41 5.41 13.41 9.41"/></g></g></svg>
              <div class="label">source</div>
            </a>
            <a class="cta edit ${this.visibleWhen("list", "detail", "!readOnly")}" @click=${()=>this.onEdit()}>
              <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24"><path d="M14.078 7.061l2.861 2.862-10.799 10.798-3.584.723.724-3.585 10.798-10.798zm0-2.829l-12.64 12.64-1.438 7.128 7.127-1.438 12.642-12.64-5.691-5.69zm7.105 4.277l2.817-2.82-5.691-5.689-2.816 2.817 5.69 5.692z"/></svg>
              <div class="label">edit</div>
            </a>
            <a class="cta save ${this.visibleWhen("edit", "add", "!readOnly")}" @click=${()=>this.onSave()}>
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><title>Asset 1</title><g id="Layer_2" data-name="Layer 2"><g id="iconmonstr"><path id="save-2" d="M13,3h3V8H13ZM24,4V24H0V0H20ZM7,9H17V2H7ZM22,4.83,19.17,2H19v9H5V2H2V22H22Z"/></g></g></svg>
              <div class="label">save</div>
            </a>
            <a class="cta cancel ${this.visibleWhen("edit", "add")}" @click=${()=>this.onCancel()}>
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><title>cancel</title><g id="Layer_2" data-name="Layer 2"><g id="iconmonstr"><polygon id="x-mark-2" points="24 21.08 14.81 11.98 23.91 2.81 21.08 0 11.99 9.18 2.81 0.09 0 2.9 9.19 12.01 0.09 21.19 2.9 24 12.01 14.81 21.19 23.91 24 21.08"/></g></g></svg>
              <div class="label">cancel</div>
            </a>
            <a class="cta delete ${this.visibleWhen("edit", "!readOnly")}" @click=${()=>this.onDelete()}>
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 16"><defs><style>.cls-1{fill-rule:evenodd;}</style></defs><title>delete</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><path class="cls-1" d="M24,16H7L0,8,7,0H24V16ZM7.91,2,2.66,8,7.9,14H22V2ZM14,6.59,16.59,4,18,5.41,15.41,8,18,10.59,16.59,12,14,9.41,11.41,12,10,10.59,12.59,8,10,5.41,11.41,4,14,6.59Z"/></g></g></svg>
              <div class="label">delete</div>
            </a>
            <a class="cta edit ${this.visibleWhen("list", "detail", "edit", "add")}" @click=${(e)=>this.onYaml(e.target.checked)}>
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64" width="64" height="64"><title>zoom</title><g class="nc-icon-wrapper" stroke-linecap="square" stroke-linejoin="miter" stroke-width="2" fill="#000000" stroke="#000000"><line data-color="color-2" x1="59" y1="59" x2="42.556" y2="42.556" fill="none" stroke-miterlimit="10"/><circle cx="27" cy="27" r="22" fill="none" stroke="#000000" stroke-miterlimit="10"/></g></svg>
              <div class="label">yaml</div>
            </a>
          </div>
          -->
        </div>
      </form>
      `
  }

  /* renderMessages()
   * This renders the message list, if there are any messages to report for this particular resource.
   *
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

  /* renderYAML()
   * This renders the YAML that would be applied to Kubernetes when the resource is added or saved.
   *
   *
   */

  renderYAML() {
    throw Error("Not Yet Implemented");
    /* TODO: rewrite using current YAML merge code and returned diffs, rather than this.state.* */

    try {
      let yaml = this.mergedYaml();
      let entries = [];
      let changes = false;
      this.state.diff.forEach((v, k) => {
        if (v !== "ignored") {
          changes = true;
          entries.push(html`<li><span class="yaml-path">${k}</span> <span class="yaml-change">${v}</span></li>`);
        }
      });

      return html`
        <div class="yaml" style="display: ${this.showYAML ? "block" : "none"}">
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

  /* validate()
 * This method is invoked on save in order to validate input prior to proceeding with the save action.
 * The model validates its current state, so anything that the View wants to validate must already be in the model.
 *
 * validate() returns a Map of fieldnames and error strings. If the dictionary is empty, there are no errors.
 *
 * For now we will have a side-effect of validate in that any errors will be added to the message list.
  */

  validate() {
    let errors = this.model.validate() || new Map();

    for (let [property, errorStr] of errors) {
      this.addMessage(`Resource property ${property} not valid: ${errorStr}`)
    }

    /* Allow subclasses to validate as well. */
    errors = new Map(...errors, ...this.validateSelf());

    return errors;
  }
}

