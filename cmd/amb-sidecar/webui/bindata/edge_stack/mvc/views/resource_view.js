/*
 * ResourceView
 * A LitElement subclass that implements a generic view on a Resource model object.  It contains cached values of
 * its Resource model object, as well as state used for the different view variants (edit, add, etc.)
 */

import { LitElement, html, css } from '../vendor/lit-element.min.js'
import { getCookie }             from '../components/cookies.js';
import { View }                  from './view.js'

export class ResourceView extends View {

  /* properties
   * These are the properties of the ResourceView, which reflect the properties of the underlying Resource,
   * and also include transient state (e.g. viewState). LitElement manages these declared properties and
   * provides various services depending on how they are used.  For further details on LitElement, see
   * https://lit-element.polymer-project.org/guide/properties
   */

  static get properties() {
    return {
      kind:      {type: String},  // Resource
      name:      {type: String},  // Resource
      namespace: {type: String},  // Resource
      viewState: {type: String},  // View
      showYAML:  {type: Boolean}  // ResourceView
    }
  }

  /* constructor
   * The ResourceView constructor, which takes a Resource (model) as its parameter.
   * We cache the state from the model in the view its itself, as properties.
   * Because this is a web component, the property updates queue the appropriate re-rendering at the correct time.
   */

  constructor(model) {
    super(model);

    /* Cache state from the model. */
    this.kind = model.kind;
    this.name = model.name;
    this.namespace = model.namespace;
    this.status = model.status;

    /* Since we are managing a Resource view, we may have messages to display, and optional YAML */
    this.messages = [];
    this.showYAML = false;
  }

  /**
   * This method is called on the View when the View needs to match the current state of its Model.
   * Generally this happens during initialization and during editing when the Cancel button is pressed and the
   * View reverts to displaying the original Model's state.
   */

  readFromModel() {
    this.clearMessages();

    /* Get the name and namespace from the model */
    this.name = this.model.name;
    this.namespace = this.model.namespace;

    /* Set the edit fields */
    this.nameInput().value = this.name;
    this.namespaceInput().value = this.namespace;

    /* Allow subclasses to read their state from the model. */
    this.readSelfFromModel();
  }

  /**
   * This method is called on the View when the View has new, validated state that should be written back
   * to the Model.  This happens during a Save operation after the user has modified the View.
   */

  writeToModel() {
    this.model.name = this.nameInput().value;
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

