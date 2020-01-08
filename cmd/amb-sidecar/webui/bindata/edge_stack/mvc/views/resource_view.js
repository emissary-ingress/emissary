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
    return mapMerge(myProperties, View.properties());
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
 * Add a message to the messages list.  This list can be rendered
 * indicate there is an error. If any errors have been added by
 * validate(), they are displayed to the user rather than allowing
 * the save action to proceed.
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
    this.model.name      = this.nameInput().value;
    this.model.namespace = this.namespaceInput().value;

    /* Allow subclasses to write their state to the model. */
    this.writeSelfToModel();
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

