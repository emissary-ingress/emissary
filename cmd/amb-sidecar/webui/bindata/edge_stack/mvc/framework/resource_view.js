/*
 * ResourceView
 * A View subclass that implements a generic view on a Resource model object.  It contains cached values of
 * its Resource model object, as well as state used for the different view variants (edit, add, etc.)
 */

import { html, css } from '../../vendor/lit-element.min.js'

/* Object merge operation */
import { objectMerge } from "../framework/utilities.js"

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
    return objectMerge(myProperties, View.properties);
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
    }`
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

    /* For editing, we will save the existing Model while we edit a new one that will replace the old.
    /* For editing, we will save the existing Model while we edit a new one that will replace the old.
     * If there is a savedModel, then there must be an edit in progress using this view.
     */
    this._savedModel = null;
    this._timeout    = null;
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

  /* onAdd()
   * This method is called on the View when the View has been newly-added to a ResourceCollectionView
   * and needs to change to its Add mode.  This is different than the normal process when editing an
   * existing Resource since onEdit() is called when the Edit button is pressed, and doAdd is called
   * by the ResourceCollectionView to begin the add process.
   */

  onAdd() {
    /* Change view to "add" state, and request to focus */
    this.viewState   = "add";
    this._needsFocus = true;
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

  /* onCancel()
   * This method is called on the View when the View is in Edit mode, and the user clicks on the
   * Cancel button to discard the changes and return to the original state.
   */

  onCancel() {
    /* Adding?  Remove the view--the Resource is not being added to the system. */
    if (this.viewState === "add") {
      this.parentElement.removeChild(this);
    }

    /* Editing? Swap models back, restoring listeners to the saved Model. */
    if (this.viewState === "edit") {
      this.model.removeListener(this);
      this.model = this._savedModel;
      this.model.addListener(this);
      this._savedModel = null;

      /* Restore the fields to the previous model's. */
      this.readFromModel();

      /* Restore to "list" state. */
      this.viewState = "list";
    }
  }

  /* onDelete()
   * This method is called on the View when the user has clicked the Delete button to delete the Resource.
   * Like saving, this switches to a pending mode until the Resource has been observed to be deleted.
   */

  onDelete() {
    let proceed = confirm(`You are about to delete the ${this.kind} named '${this.name}' in the '${this.namespace}' namespace.\n\nAre you sure?`);

    if (proceed) {
      /* Ask the Resource to delete itself. */
      let error = this.model.doDelete();

      if (error === null) {
        /* Note that the resource is pending an update (in this case, to be deleted),
         * and update the viewState to show the pending-delete button and pending visualization */
        this.model.setPending("delete");
        this.viewState = "pending-delete";

        /* Start the timeout for 5 seconds to make sure that the pending delete is reset even if the backend fails */
        this._timeout = setTimeout(this.verifyDelete.bind(this), 5000);
      }
      else {
        this.addMessage("Delete failed - backend not available?");
        console.log("ResourceView.onDelete() returned error ${error");
      }
    }
  }

  /* verifyDelete()
   * This method is called when the timeout finishes, to check whether the resource being deleted has in fact
   * been successfully removed and is no longer in the snapshot.
   */

  verifyDelete() {
  }


    /* onEdit()
    * This method is called on the View when the View needs to change to its Edit mode.  The View needs
    * to create a new copy of its Model for editing, and stop listening to any updates to the old Model.
    */

  onEdit() {
    /* Clear any error messages prior to editing. */
    this.clearMessages();

    /* Save the View's existing model and stop listening to it. */
    this._savedModel = this.model;
    this.model.removeListener(this);

    /* Create a new model for editing, based on the state in the existing model, and start listening to it. */
    this.model = this.model.copySelf();
    this.model.addListener(this);

    /* Change view to "edit" state, and request to focus */
    this.viewState   = "edit";
    this._needsFocus = true;
  }

  /* onSave()
    * This method is called on the View when the View is in Edit mode, and the user clicks on the
    * Save button to save the changes.  There are two circumstances in which the Save button will be clicked:
    * 1) a new Resource (model) has been added and Save creates a new Resource in the backend.  In this case,
    * after the model executes doSave(), the Resource will appear (or not) in the snapshot and will be updated
    * at that time.
    * 2) an existing Resource is being edited and Save writes back any changes.
    */

  onSave() {
    /* Validate the data in the model. */
    let validationErrors = this.validate();

    if (validationErrors.size === 0) {
      /* Save the changes in the resource. */
      let error = this.model.doSave();

      if (error === null) {
        /* Save is invoked in two cases:
         * 1) after a new Resource has been added; doSave then creates a new Resource object in Kubernetes.
         * 2) after an existing Resource has been edited; doSave then updates that existing Resource in Kubernetes.
         *
         * The resource's pending state indicates which case is to be run.
         */

        if (this.model.isPending("add")) {
          /* Nothing needs to be done.  The model will be updated by the ResourceCollection at some point
           * in the future.  The timeout, which is set on all Save operations, could cause the add to fail,
           * in which case the snapshot will not include the object and it will be removed.  The ResourceCollection
           * will not remove from the collection any Resources with pending state.
           */
        }
        else if (this.model.isPending("edit")) {
          /*  Copy our YAML to the saved model so that it can be identified as existing in the
           * ResourceCollectionView (since it is the "same" model as before, with different state).
           * Note: the savedModel has no listeners so this will just update the model's attribute values.
           */

          this._savedModel.updateFrom(this.model.getYAML());

          /* Stop listening to the new model, swap, and start listening again to the savedModel (with new state),
           * which was previously in the ResourceCollection and thus will be identified as having changed.
           * If the resource name has changed, then the old model will disappear and a new one will take its place.
           */

          this.model.removeListener(this);
          this.model = this._savedModel;
          this.model.addListener(this);

          /* We believe that the save was successful.  Keep the new model and note that its state is pending. */
          this._savedModel = null;
        }

        /* Note that the resource is pending save, and change the view to "pending-save" to show that the
         * view is pending the result of the save operation.
         */
        this.model.setPending("save");
        this.viewState = "pending-save";

        /* Start the timeout for 5 seconds to make sure that the pending save is reset even if the backend fails */
        this._timeout = setTimeout(this.verifySave.bind(this), 5000);

      } else {
        this.addMessage("Save failed -- backend not available?");
        console.log(`ResourceView.onSave() returned error ${error}`);
      }
    }
    /* Have validation errors.  Add to the message list. */
    else {
      for (let [field, message] of validationErrors) {
        this.addMessage(`${field}: ${message}`);
      }

      /* Messages are not yet in their own component, so request update of the entire view. */
      this.requestUpdate();
    }
  }

  /* verifySave()
   * This method is called when the timeout finishes, to check whether the resource being saved has in fact
   * been successfully saved and has been updated by the snapshot.
   */

  verifySave() {
  }

  /* onSource()
   * This method opens a window on the Resource's source URI.
   */

    onSource(mouseEvent) {
      window.open(this.model.sourceURI());

      /* Defocus the button */
      mouseEvent.currentTarget.blur();
    }

  /* onYaml()
   * This method hides and shows the current YAML of the Resource.
   */

  onYaml(mouseEvent) {
    /* Toggle showYAML */
    this.showYAML = !this.showYAML;

    /* Defocus the button */
    mouseEvent.currentTarget.blur();
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
    /* If pending, show a crosshatch over the view content. */
    let pending = this.model.pending("add", "delete", "save");

    /* Return the HTML, including calls to renderSelf() to allow subclasses to specialize. */
    return html`
      <link rel="stylesheet" href="../styles/oneresource.css">
      ${this.modifiedStyles() ? this.modifiedStyles() : ""}
      <form>
        <div class="card ${this.viewState === "off" ? "off" : ""}">
          <div class="col">
            <div class="${pending ? "pending" : ""}">
            
              <!-- Render common Resource fields: kind, name, namespace, as well as input fields when editing.   -->
              <div class="row line">
                <div class="row-col margin-right">${this.kind}:</div>
              </div>
              
              <div class="row line">
                <label class="row-col margin-right justify-right">name:</label>
                <div class="row-col">
                  <b class="${this.visibleWhen("list", "pending-save", "pending-delete")}">${this.name}</b>
                  
                  <input class="${this.visibleWhen("add", "edit")}" name="name" type="text" value="${this.name}"/>
                </div>
              </div>
              
              <div class="row line">
                <label class="row-col margin-right justify-right">namespace:</label>
                <div class="row-col">
                  <div class="namespace${this.visibleWhen("list", "pending-save", "pending-delete")}">(${this.namespace})</div>
                  
                  <div class="namespace-input ${this.visibleWhen("add", "edit")}">
                    <div class="pararen">(</div>
                    <input class="${this.visibleWhen("add", "edit")}" name="namespace" type="text" value="${this.namespace}"/>
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
          
            <a class="cta source ${typeof this.model.sourceURI() == 'string' ? "" : "off"}" @click=${(x)=>this.onSource(x)}>
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 18.83 10.83"><defs><style>.cls-2{fill:none;stroke:#000;stroke-linecap:square;stroke-miterlimit:10;stroke-width:2px;}</style></defs><title>source_2</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><polyline class="cls-2" points="5.41 1.41 1.41 5.41 5.41 9.41"/><polyline class="cls-2" points="13.41 1.41 17.41 5.41 13.41 9.41"/></g></g></svg>
              <div class="label">source</div>
            </a>
            
            <a class="cta pending ${this.visibleWhen("pending-save")}">
              <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24"><path d="M14.078 7.061l2.861 2.862-10.799 10.798-3.584.723.724-3.585 10.798-10.798zm0-2.829l-12.64 12.64-1.438 7.128 7.127-1.438 12.642-12.64-5.691-5.69zm7.105 4.277l2.817-2.82-5.691-5.689-2.816 2.817 5.69 5.692z"/></svg>
              <div class="label">pending</div>
            </a>
            
            <a class="cta pending ${this.visibleWhen("pending-delete")}">
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 16"><defs><style>.cls-1{fill-rule:evenodd;}</style></defs><title>delete</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><path class="cls-1" d="M24,16H7L0,8,7,0H24V16ZM7.91,2,2.66,8,7.9,14H22V2ZM14,6.59,16.59,4,18,5.41,15.41,8,18,10.59,16.59,12,14,9.41,11.41,12,10,10.59,12.59,8,10,5.41,11.41,4,14,6.59Z"/></g></g></svg>
              <div class="label">pending</div>
            </a>
            
            <a class="cta edit ${this.visibleWhen("list", "detail")}" @click=${()=>this.onEdit()}>
              <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24"><path d="M14.078 7.061l2.861 2.862-10.799 10.798-3.584.723.724-3.585 10.798-10.798zm0-2.829l-12.64 12.64-1.438 7.128 7.127-1.438 12.642-12.64-5.691-5.69zm7.105 4.277l2.817-2.82-5.691-5.689-2.816 2.817 5.69 5.692z"/></svg>
              <div class="label">edit</div>
            </a>
            
            <a class="cta save ${this.visibleWhen("edit", "add")}" @click=${()=>this.onSave()}>
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><title>Asset 1</title><g id="Layer_2" data-name="Layer 2"><g id="iconmonstr"><path id="save-2" d="M13,3h3V8H13ZM24,4V24H0V0H20ZM7,9H17V2H7ZM22,4.83,19.17,2H19v9H5V2H2V22H22Z"/></g></g></svg>
              <div class="label">save</div>
            </a>
            
            <a class="cta cancel ${this.visibleWhen("edit", "add")}" @click=${()=>this.onCancel()}>
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><title>cancel</title><g id="Layer_2" data-name="Layer 2"><g id="iconmonstr"><polygon id="x-mark-2" points="24 21.08 14.81 11.98 23.91 2.81 21.08 0 11.99 9.18 2.81 0.09 0 2.9 9.19 12.01 0.09 21.19 2.9 24 12.01 14.81 21.19 23.91 24 21.08"/></g></g></svg>
              <div class="label">cancel</div>
            </a>
            
            <a class="cta delete ${this.visibleWhen("list", "detail")}" @click=${()=>this.onDelete()}>
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 16"><defs><style>.cls-1{fill-rule:evenodd;}</style></defs><title>delete</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><path class="cls-1" d="M24,16H7L0,8,7,0H24V16ZM7.91,2,2.66,8,7.9,14H22V2ZM14,6.59,16.59,4,18,5.41,15.41,8,18,10.59,16.59,12,14,9.41,11.41,12,10,10.59,12.59,8,10,5.41,11.41,4,14,6.59Z"/></g></g></svg>
              <div class="label">delete</div>
            </a>
            
            <a class="cta edit ${this.visibleWhen("list", "detail", "edit", "add")}" @click=${(e)=>this.onYaml(e.target.checked)}>
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64" width="64" height="64"><title>zoom</title><g class="nc-icon-wrapper" stroke-linecap="square" stroke-linejoin="miter" stroke-width="2" fill="#000000" stroke="#000000"><line data-color="color-2" x1="59" y1="59" x2="42.556" y2="42.556" fill="none" stroke-miterlimit="10"/><circle cx="27" cy="27" r="22" fill="none" stroke="#000000" stroke-miterlimit="10"/></g></svg>
              <div class="label">yaml</div>
            </a>
            
          </div>
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
    if (this.showYAML) {
      try {
        let yaml = jsyaml.safeDump(this.model.getYAML())
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

  /* updated()
   * This method is a LitElement callback which is invoked when the element's DOM has been updated
   * and rendered.  Here it is used to focus and select the resource's name field when being edited.
  */
  updated(changedProperties) {
    if (this._needsFocus) {
      this.nameInput().focus();
      this.nameInput().select();
      this._needsFocus = false;
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

  /* yamlElement()
 * This method returns the element that renders the YAML of the resource, either the pending changes
 * if shown during editing, or the current values.
 */

  yamlElement() {
    return this.shadowRoot.getElementById("merged-yaml");
  }
}

