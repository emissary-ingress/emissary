/**
 * View
 * This is the framework class for Views, which are Web Component elements to be rendered in a browser.
 * The View implements listener behavior, receiving notifications from Model objects when they change state,
 * and update their properties to be redisplayed.
 *
 * This View implementation also assumes quite a bit about the HTML environment it renders in.  There may be
 * modifiedStyles as well.  This View is for rendering Resources and Resource subclasses.
 *
 */

import { LitElement, html } from '../../vendor/lit-element.min.js'

export class View extends LitElement {

  /* properties
   * These are the properties of the View, which reflect the properties of the underlying Model,
   * and also include transient state (e.g. viewState).
   *
   * LitElement manages these declared properties and provides various services depending on how they are used.
   * https://lit-element.polymer-project.org/guide/properties
   */

  static get properties() {
    return {
      model: {type: Object, noAccessor: true},
      viewState: {type: String}  // View
    }
  }

  /* Because this is a web component, the constructor needs to be empty. We just do basic/minimal
   * initialization here. Real initialization needs to happen lazily and/or when the component is
   * first connected to the DOM. (See connectedCallback() and disconnectedCallback()).
   */
  constructor() {
    super();
    this.viewState = "list";
  }

  /**
   * The model property holds the business logic and data for the view. Because it is a property,
   * updates will cause re-rendering to happen at the correct time.
   */
  set model(value) {
    if (this._model) {
      this._model.removeListener(this);
    }

    this._model = value

    /* listen to model changes for updates. Model will call this.onModelNotification with
     * the model itself, a message, and an optional parameter.
     */
    this._model.addListener(this);
  }

  get model() {
    return this._model;
  }

  /**
   * Override to extend the styles of this resource (see yaml download tab).
   */
  modifiedStyles() {
    return null;
  }

  /* onModelNotification(model, message, parameter)
    * When we get a notification from the model that one or more model values have changed, the properties are updated.
    * Because this is a web component, the property updates queue the appropriate re-rendering at the correct time.
    */

  onModelNotification(model, message, parameter) {
    switch (message) {
      /* if updated, then set our properties to those of the model */
      case 'updated':
        this.readFromModel();

        /* Switch back to list view, if we were pending an update. */
        if (this.viewState === "pending") {
          this.viewState = "list";
        }
        break;
      /*
       * And if we are notified that our model has been deleted, we remove ourselves from our parent.  Again, because
       * this is a web component, that will queue the appropriate re-render at the correct time.
       */
      case 'deleted':
        if (this.viewState === "pending") {
          this.viewState = "list";
        }
        this.parentElement.removeChild(this);
        break;
    }
  }

  /* render()
   * Render the view.  This html assumes styles are imported properly and a common layout of the view that has
   * list, edit, detail, and add variants, the current value of which is saved in the viewState instance variable.
   * The concrete subclass implementation of a View is responsible only for rendering itself (renderSelf()) and
   * hiding and showing fields based on the viewState.
   */

  render() {
    return html`
      <link rel="stylesheet" href="../../styles/oneresource.css">
      ${this.modifiedStyles() ? this.modifiedStyles() : ""}
      <form>
        <div class="card ${this.viewState === "off" ? "off" : ""}">
          <div class="col">
            <div class="row line">
              <div class="row-col margin-right">${this.kind}:</div>
              <div class="row-col">
                <b class="${this.visibleWhen("list", "edit")}">${this.name}</b>
                <input class="${this.visibleWhen("add")}" name="name" type="text" value="${this.name}"/>
                <div class="namespace${this.visibleWhen("list", "edit")}">(${this.namespace})</div>
                <div class="namespace-input ${this.visibleWhen("add")}"><div class="pararen">(</div><input class="${this.visibleWhen("add")}" name="namespace" type="text" value="${this.namespace}"/><div class="pararen">)</div></div>
              </div>
            </div>
      
          ${this.renderSelf()}
         
          </div>
          <div class="col2">
          </div>
        </div>
      </form>`
  }

  /* visibleWhen(...arguments)
  * return the empty string if the current viewState is listed in the arguments, "off" otherwise.
  */

  visibleWhen() {
    return [...arguments].includes(this.viewState) ? "" : "off";
  }
}
