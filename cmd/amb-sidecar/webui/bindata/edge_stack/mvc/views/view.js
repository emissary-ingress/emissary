/**
 * View
 * This is the framework class for Views, which are Web Component elements to be rendered in a browser.
 * The View implements listener behavior, receiving notifications from Model objects when they change state,
 * and update their properties to be redisplayed.
 *
 * This View implementation also assumes quite a bit about the HTML environment it renders in.  There may be
 * modifiedStyles
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
      viewState: {type: String},  // View
    }
  }

  /* constructor(model)
   * The View constructor, which takes a Model (model) as its parameter.
   */

  constructor(model) {
    super();
    this.model     = model;
    this.viewState = "list";

    /* and listen to model changes for updates. Model will call this.onModelNotification with
     * the model itself, a message, and an optional parameter.
     */

    model.addListener(this);
  }

   /* minimumNumberOfAddRows()
   * minimumNumberOfEditRows()
   *
   * To help the UI place buttons within the rectangle border (or more precisely, to help the UI grow the rectangle
   * border to fit all the buttons), these two functions should be overridden if the renderSelf has fewer than four
   * rows in edit mode and/or fewer than two rows in add mode.
   * (Override these functions if the add and edit buttons on the right side of the frame are extending below the
   * bottom of the frame.)
   */

  minimumNumberOfAddRows() {
    return 2;
  }
  minimumNumberOfEditRows() {
    return 4;
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
        break;
      /*
       * And if we are notified that our model has been deleted, we remove ourselves from our parent.  Again, because
       * this is a web component, that will queue the appropriate re-render at the correct time.
       */
      case 'deleted':
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
                <b class="${this.visible("list", "edit")}">${this.name}</b>
                <input class="${this.visible("add")}" name="name" type="text" value="${this.name}"/>
                <div class="namespace${this.visible("list", "edit")}">(${this.namespace})</div>
                <div class="namespace-input ${this.visible("add")}"><div class="pararen">(</div><input class="${this.visible("add")}" name="namespace" type="text" value="${this.namespace}"/><div class="pararen">)</div></div>
              </div>
            </div>
      
          ${this.renderSelf()}
         
          </div>
          <div class="col2">
            <a class="cta edit ${this.visible("list", "detail", "edit", "add")}" @click=${(e)=>this.onYaml(e.target.checked)}>
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64" width="64" height="64"><title>zoom</title><g class="nc-icon-wrapper" stroke-linecap="square" stroke-linejoin="miter" stroke-width="2" fill="#000000" stroke="#000000"><line data-color="color-2" x1="59" y1="59" x2="42.556" y2="42.556" fill="none" stroke-miterlimit="10"/><circle cx="27" cy="27" r="22" fill="none" stroke="#000000" stroke-miterlimit="10"/></g></svg>
              <div class="label">yaml</div>
            </a>
          </div>
        </div>
      </form>`
  }
}
