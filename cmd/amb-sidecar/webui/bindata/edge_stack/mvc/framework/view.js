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
      viewState: {type: String}  // View
    }
  }

  /* constructor(model)
   * The View constructor, which takes a Model as its parameter.
   */

  constructor(model) {
    super();
    this.model     = model;
    this.viewState = "list";

    /* listen to model changes for updates. Model will call this.onModelNotification with
     * the model itself, a message, and an optional parameter.
     */

    model.addListener(this);
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
            <a class="cta edit ${this.visibleWhen("list", "edit", "add")}" @click=${(e)=>this.onYaml(e.target.checked)}>
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64" width="64" height="64"><title>zoom</title><g class="nc-icon-wrapper" stroke-linecap="square" stroke-linejoin="miter" stroke-width="2" fill="#000000" stroke="#000000"><line data-color="color-2" x1="59" y1="59" x2="42.556" y2="42.556" fill="none" stroke-miterlimit="10"/><circle cx="27" cy="27" r="22" fill="none" stroke="#000000" stroke-miterlimit="10"/></g></svg>
              <div class="label">yaml</div>
            </a>
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
