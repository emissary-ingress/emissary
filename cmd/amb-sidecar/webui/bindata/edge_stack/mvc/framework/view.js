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

import { LitElement, html, css } from '../../vendor/lit-element.min.js'

export class View extends LitElement {

  /* properties
   * These are the properties of the View, which reflect the properties of the underlying Model,
   * and also include transient state (e.g. viewState).
   *
   * LitElement manages these declared properties and provides various services depending on how they are used.
   * https://lit-element.polymer-project.org/guide/properties
   */

  static get styles() {
    return css`
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
  `
  }

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
