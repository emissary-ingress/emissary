import { LitElement, html, css} from '../vendor/lit-element.min.js'
import {Snapshot, aes_res_editable, aes_res_changed, aes_res_source} from './snapshot.js'
import {getCookie} from './cookies.js';
import {ApiFetch} from "./api-fetch.js";

/**
 * The classes in this file provide the building blocks we use for
 * displaying, adding, and editing kubernetes resources, most
 * importantly the CRDs that ambassador uses to get input from users and
 * communicate output back to users.
 *
 * There are a couple different goals of using an abstraction here:
 *
 *  - Consistency of experience for cross-cutting aspects of CRDs.
 *
 *    One of the things that makes kubernetes powerful for advanced
 *    users is the ways that they can treat all their resources the
 *    same way. Labels, annotations, selectors, and status are some
 *    examples of these shared concepts that we want to provide a
 *    consistent experience for.
 *
 *  - Gitops workflow.
 *
 *    A particularly important example of the above consistent
 *    experience is the ability to use a gitops workflow to manage
 *    your kubernetes resources. For example, defining your source of
 *    truth declaratively in git and updating your cluster via apply.
 *    We need our UI to work well with this gitops workflow
 *
 * Of course in addition to the above, we also want to be able to
 * customize each resource so that we can display, add, and edit it in
 * the best way for that resource. Navigate quickly to other relevant
 * resources, and in general help new users become advanced users
 * faster!
 *
 * There are two base classes (SingleResource, and ResourceSet) intended to be
 * extended as a way to define two web components that work with each
 * other: a single-resource web component, e.g. <dw-host>, and a
 * many-resource component for displaying lots of resources,
 * e.g. <dw-hosts>.
 *
 *
 * There are a number of future features we expect to be adding to
 * the base components in this file:
 *
 *  - For ResourceSets we can provide searching/sorting/filtering
 *    based on the kubernetes metadata that is common to all resources
 *    (labels, annotations, names, namespaces), with extension points
 *    for custom searching/sorting/filtering for specific Kinds.
 *
 *  - Provide ways to select a number of resources and export the yaml.
 *
 *  - Provide a way to edit a specific resource, but instead of saving
 *    directly to kubernetes, download the yaml.
 *
 *  - Leverage kubernetes generate-id to avoid read/modify/write
 *    hazzards when you edit/save a resource.
 *
 *  - Provide a way to rollup all resources with a non-green status and
 *    show them prominently in the dashboard.
 *
 *  - Make sure we label any resources that were created by the UI, and
 *    disallow editing of resources that were not created by the UI so
 *    that we never try to write to resources maintained in git.
 *
 *  - If a resource has an annotation with a git repo, show a link to
 *    that git repo so people can go there and edit it
 *
 *  - Leverage the git repo annotation to allow edit/save to work
 *    on those resources by filing a PR.
 */

/**
 * The SingleResource class is an abstract base class that is extended
 * in order to create a widget for a kubernetes resource. Every
 * kubernetes resource widget supports the following display modes:
 *
 *   - list (a compact representation suitable for use in displaying many resources)
 *   - detail (an expanded representation that includes information omitted from the list display)
 *   - edit (a view that displays the resources values in editable fields)
 *   - add (a view that displays the (defaulted) fields necessary to create a resource)
 *
 * The base SingleResource class provides the state machinery that
 * tracks the current view of a resource and renders the controls that
 * allow switching between them (e.g. the edit button).
 *
 * When extended to display a given resource, each mode can be
 * customized as appropriate to provide the optimal experience for
 * that resource.
 *
 * When you extend a SingleResource, you MUST override the following
 * methods. See each method for a more detailed description:
 *
 *  kind() --> return the kubernetes "Kind" of the resource
 *
 *  spec() --> define how yaml is rendered for the kubectl apply that happens on add/save
 *
 *  renderResource() --> define how the customizable portions of the widget look for each view
 *
 * When you extend a SingleResource, you probably SHOULD override the
 * following methods. See each method for a more detailed description:
 *
 *   validate() --> define how to perform custom validation prior to save
 *
 *   reset() --> reset any ui state related to add/edit on cancel
 *
 * When you extend a SingleResource, you MAY override the following
 * methods. See each method for a more detailed description:
 *
 *   init() --> initialize any ui state when a widget is first rendered
 */
export class SingleResource extends LitElement {

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
    
    .card .row .line{
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

  // internal
  static get properties() {
    return {
      resource: {type: Map},
      state: {type: UIState},
    }
  }

  // internal
  constructor() {
    super();
    this.resource = {};
    this.state = new UIState();
  }

  /**
   * This is invoked whenever a resource is first displayed on the
   * page. You can use this to initialize new UI state. When we get
   * repeat, this will be able to go away and we can just use
   * constructors.
   */
  init() {}

  // internal
  update() {
    if (this.state instanceof UIState) {
      this.state.init(this)
    }
    super.update()
  }

  // internal
  onAdd() {
    if( this.readOnly() ) {
      return; // we shouldn't be able to get here because there is no add button,
              // but if we do, don't do anything.
    }
    this.requestUpdate();
    this.reset();
    this.state.mode = "add"
  }

  // internal
  onEdit() {
    if( this.readOnly() ) {
      return; // we shouldn't be able to get here because there is no edit button,
              // but if we do, don't do anything.
    }
    this.requestUpdate();
    this.reset();
    if (this.state.mode !== "edit") {
      this.state.mode = "edit"
    } else {
      this.state.mode = "list"
    }
  }


  // internal
  /* Open the window on the source URI */
  onSource(mouseEvent) {
    window.open(this.sourceURI())

    /* Defocus the button */
    mouseEvent.currentTarget.blur()
  }

  // internal
  onDelete() {
    if (this.readOnly()) {
      this.state.mode = "list";
      return; // we shouldn't be able to get here because there is no edit button,
              // and thus no delete button, but if we do, don't do anything.
    }

    let proceed = confirm(`You are about to delete the ${this.kind()} named '${this.name()}' in the '${this.namespace()}' namespace.\n\nAre you sure?`);
    if (!proceed) {
      return; // user canceled the action
    }

    ApiFetch('/edge_stack/api/delete',
          {
            method: "POST",
            headers: new Headers({
              'Authorization': 'Bearer ' + getCookie("edge_stack_auth")
            }),
            body: JSON.stringify({
              Namespace: this.namespace(),
              Names: [`${this.kind()}/${this.name()}`]
            })
          })
      .then(r=>{
        r.text().then(t=>{
          this.reset();
          if (r.ok) {
            // happy path
          } else {
            console.error(t);
            this.addError(`Unexpected error while deleting resource: ${r.statusText}`); // Make sure we add this error to the stack after calling this.reset();
          }
          if (this.state.mode === "add") {
            this.state.mode = "off"
          } else {
            this.state.mode = "list"
          }
        })
      })
  }

  // internal
  onCancel() {
    this.requestUpdate();

    if (this.state.mode === "add") {
      this.state.mode = "off"
    } else {
      this.state.mode = "list"
    }

    this.reset()
  }

  // Override to customize merge strategy for a specific Kind
  mergeStrategy(pathName) {
    return undefined;
  }

  // Default merge strategies. Do not override.
  defaultMergeStrategy(pathName) {
    switch (pathName) {
    case "metadata.annotations.kubectl.kubernetes.io/last-applied-configuration":
    case "metadata.creationTimestamp":
    case "metadata.generation":
    case "metadata.resourceVersion":
    case "metadata.selfLink":
    case "metadata.uid":
    case "status":
      return "ignore";
    case "":
    default:
      return "merge";
    }
  }

  // internal
  _mergeStrategy(path) {
    let pathName = path.join('.');
    let strategy = this.mergeStrategy(pathName);
    switch (strategy) {
    case "ignore":
    case "merge":
    case "replace":
      return strategy;
    default:
      return this.defaultMergeStrategy(pathName);
    }
  }

  /**
   * Merge the original and updated values based on the result of this._mergeStrategy(pathName).
   *
   * We also track the changes that we make to the original yaml. This
   * was originally for debugging, but it's also useful feedback for
   * users.
   */
  merge(original, updated, path=[]) {
    let pathName = path.join('.');
    let strategy = this._mergeStrategy(path);

    switch (strategy) {
    case "ignore":
      this.state.diff.set(pathName, "ignored");
      return undefined;
    case "replace":
      this.state.diff.set(pathName, "replaced");
      return updated;
    }

    // the rest of this function is the "merge" case:

    // handle null as a special case here because typeof null returns "object"
    if (original === null) {
      this.state.diff.set(pathName, "updated");
      return updated;
    }

    let originalType = typeof original;
    switch (originalType) {
    case "undefined":
      let updatedType = typeof updated;
      switch (updatedType) {
      case "object":
        if (Array.isArray(updated)) {
          this.state.diff.set(pathName, "updated");
          return updated;
        } else {
          return this.mergeObject(original, updated, path);
        }
      default:
        this.state.diff.set(pathName, "updated");
        return updated;
      }
    case "object":
      if (Array.isArray(original)) {
        if (updated === undefined) {
          return original;
        } else {
          this.state.diff.set(pathName, "updated");
          return updated;
        }
      } else {
        return this.mergeObject(original, updated, path);
      }
    case "string":
    case "number":
    case "bigint":
    case "boolean":
      if (original === updated || updated === undefined) { return original; }
      this.state.diff.set(pathName, "updated");
      return updated;
    default:
      throw new Error(`don't know how to merge ${originalType}`);
    }
  }

  mergeObject(original, updates, path) {
    if (original === undefined) {
      original = {};
    }
    if (updates === undefined) {
      updates = {};
    }
    let originalHas = Object.prototype.hasOwnProperty.bind(original);
    let updatesHas = Object.prototype.hasOwnProperty.bind(updates);
    let result = {};
    let keys = new Set(Object.keys(original).concat(Object.keys(updates)));
    keys.forEach(key=>{
      var merged;
      if (originalHas(key) && updatesHas(key)) {
        merged = this.merge(original[key], updates[key], path.concat([key]));
      } else if (originalHas(key) && !updatesHas(key)) {
        merged = this.merge(original[key], undefined, path.concat([key]));
      } else if (!originalHas(key) && updatesHas(key)) {
        merged = this.merge(undefined, updates[key], path.concat([key]));
      } else {
        throw new Error("this should be impossible");
      }
      if (merged !== undefined) {
        result[key] = merged;
      }
    });

    return result;
  }

  // internal
  onYaml() {
    this.state.showingYaml = !this.state.showingYaml;
    this.requestUpdate();
  }

  mergedYaml() {
    this.state.diff = new Map();
    var spec;
    var mergeInput = {metadata: {annotations: {}}};
    if (this.state.mode === "edit") {
      mergeInput.spec = this.spec();
    } else if (this.state.mode === "add") {
      mergeInput.kind = this.kind();
      mergeInput.apiVersion = "getambassador.io/v2";
      mergeInput.metadata.name = this.nameInput().value
      mergeInput.metadata.namespace = this.namespaceInput().value
      mergeInput.spec = this.spec();
    }
    mergeInput.metadata.annotations[aes_res_changed] = "true";
    let merged = this.merge(this.resource, mergeInput);
    if (typeof(jsyaml) === "undefined") {
      return "";
    } else {
      return jsyaml.safeDump(merged);
    }
  }

  renderMergedYaml() {
    try {
      let yaml = this.mergedYaml();
      let entries = [];
      let changes = false;
      this.state.diff.forEach((v, k) => {
        if (v !== "ignored") {
          changes = true;
          entries.push(html`
<li><span class="yaml-path">${k}</span> <span class="yaml-change">${v}</span></li>
`);
        }
      });

      return html`
<div class="yaml" style="display: ${this.state.showingYaml ? "block" : "none"}">
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

  /**
   * This method is invoked to reset the add/edit state of the widget
   * when the cancel button is pressed. If you add to this state,
   * which is super likely, you should override this method and reset
   * the state you add. You should also remember to call
   * super.reset() so the common state is also reset.
   */
  reset() {
    this.state.messages.length = 0;
    this.nameInput().value = this.nameInput().defaultValue;
    this.namespaceInput().value = this.namespaceInput().defaultValue
  }

  /**
   * This method is invoked from inside the validate() method to
   * indicate there is an error. If any errors have been added by
   * validate(), they are displayed to the user rather than allowing
   * the save action to proceed.
   */
  addError(message) {
    this.state.messages.push(message)
  }

  /**
   * This method is invoked on save in order to validate input prior
   * to proceeding with the save action. Use the addError() method to
   * supply error messages if any input is invalid. This method does
   * not return a value. If this.addError(message) is not invoked in
   * the implementation, then the data is assumed valid. If
   * this.addError(message) *is* invoked one or more times, then the
   * data is assumed invalid.
   */
  validate() {
    /*
     * name and namespaces rules as defined by
     * https://kubernetes.io/docs/concepts/overview/working-with-objects/names/
     */
    var nameFormat = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$/; // lower-case letters, numbers, dash, and dot

    let nameInputValue = this.nameInput().value;
    if(!( nameInputValue.match(nameFormat)
       && nameInputValue.length <= 253 )) {
      this.state.messages.push("Name must be {a-z0-9-.}, length <= 253")
    }

    var namespaceFormat = nameFormat;
    let namespaceInputValue = this.namespaceInput().value;
    if(!( namespaceInputValue.match(namespaceFormat)
      && namespaceInputValue.length <= 253 )) {
      this.state.messages.push("Namespace must be {a-z0-9-.}, length <= 253")
    }
  }

  // internal
  name() {
    return this.resource.metadata.name;
  }


  // internal
  namespace() {
    return this.resource.metadata.namespace;
  }

  // internal
  annotations() {
    return this.resource.metadata.annotations;
  }


  // internal
  nameInput() {
    return this.shadowRoot.querySelector(`input[name="name"]`)
  }

  // internal
  namespaceInput() {
    return this.shadowRoot.querySelector(`input[name="namespace"]`)
  }

  // internal
  onSave() {
    if( this.readOnly() ) {
      this.state.mode = "list";
      return; // we shouldn't be able to get here because there is no edit button,
              // and thus no save button, but if we do, don't do anything.
    }
    this.requestUpdate();

    this.state.messages.length = 0;
    this.validate();
    if (this.state.messages.length > 0) {
      return
    }

    if (typeof(jsyaml) === "undefined") {
      console.error('unable to save because jsyaml not defined');
      this.addError(`Unable to ${this.state.mode === "add" ? "create" : "save"} because of an internal error (ref: jsyaml)`);
      return;
    }
    let yaml = this.mergedYaml();

    ApiFetch('/edge_stack/api/apply',
          {
            method: "POST",
            headers: new Headers({
              'Authorization': 'Bearer ' + getCookie("edge_stack_auth")
            }),
            body: yaml
          })
      .then(r=>{
        r.text().then(t=>{
          if (r.ok) {
            // happy path
            this.reset();
            if (this.state.mode === "add") {
              this.state.mode = "off"
            } else {
              this.state.mode = "list"
            }
          } else {
            console.error(t);
            this.addError(`Unable to ${this.state.mode === "add" ? "create" : "save"} because: ${t}`); // Make sure we add this error to the stack after calling this.reset();
          }
        })
      })
  }

  // deprecated, use <visible-modes>...</visible-modes> instead
  visible() {
    if( [...arguments].includes("!readOnly") ) {
      if( this.readOnly() ) {
        return "off";
      }
    }
    return [...arguments].includes(this.state.mode) ? "" : "off"
  }

  // internal
  updated() {
    this.shadowRoot.querySelectorAll("visible-modes").forEach((vm)=>{
      vm.mode = this.state.mode
    })
  }

  // internal
  render() {
    let xyz = html`
${this.modifiedStyles() ? this.modifiedStyles() : ""}
<form>
  <div class="card ${this.state.mode === "off" ? "off" : ""}">
    <div class="col">
      <div class="row line">
        <div class="row-col margin-right">${this.kind()}:</div>
      </div>
      <div class="row line">
        <label class="row-col margin-right justify-right">name:</label>
        <div class="row-col">
          <b class="${this.visible("list", "edit")}">${this.name()}</b>
          <input class="${this.visible("add")}" name="name" type="text" value="${this.name()}"/>
        </div>
      </div>
      <div class="row line">
        <label class="row-col margin-right justify-right">namespace:</label>
        <div class="row-col">
          <div class="namespace${this.visible("list", "edit")}">(${this.namespace()})</div>
          <div class="namespace-input ${this.visible("add")}"><div class="pararen">(</div><input class="${this.visible("add")}" name="namespace" type="text" value="${this.namespace()}"/><div class="pararen">)</div></div>
        </div>
      </div>

    ${this.renderResource()}

${this.state.renderErrors()}
${this.renderMergedYaml()}

    </div>
    <div class="col2">
      <a class="cta source ${typeof this.sourceURI() == 'string' ? "" : "off"}" @click=${(x)=>this.onSource(x)}>
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 18.83 10.83"><defs><style>.cls-2{fill:none;stroke:#000;stroke-linecap:square;stroke-miterlimit:10;stroke-width:2px;}</style></defs><title>source_2</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><polyline class="cls-2" points="5.41 1.41 1.41 5.41 5.41 9.41"/><polyline class="cls-2" points="13.41 1.41 17.41 5.41 13.41 9.41"/></g></g></svg>
        <div class="label">source</div>
      </a>
      <a class="cta edit ${this.visible("list")}" @click=${()=>this.onEdit()}>
        <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24"><path d="M14.078 7.061l2.861 2.862-10.799 10.798-3.584.723.724-3.585 10.798-10.798zm0-2.829l-12.64 12.64-1.438 7.128 7.127-1.438 12.642-12.64-5.691-5.69zm7.105 4.277l2.817-2.82-5.691-5.689-2.816 2.817 5.69 5.692z"/></svg>
        <div class="label">edit</div>
      </a>
      <a class="cta save ${this.visible("edit", "add")}" @click=${()=>this.onSave()}>
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><title>Asset 1</title><g id="Layer_2" data-name="Layer 2"><g id="iconmonstr"><path id="save-2" d="M13,3h3V8H13ZM24,4V24H0V0H20ZM7,9H17V2H7ZM22,4.83,19.17,2H19v9H5V2H2V22H22Z"/></g></g></svg>
        <div class="label">save</div>
      </a>
      <a class="cta cancel ${this.visible("edit", "add")}" @click=${()=>this.onCancel()}>
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><title>cancel</title><g id="Layer_2" data-name="Layer 2"><g id="iconmonstr"><polygon id="x-mark-2" points="24 21.08 14.81 11.98 23.91 2.81 21.08 0 11.99 9.18 2.81 0.09 0 2.9 9.19 12.01 0.09 21.19 2.9 24 12.01 14.81 21.19 23.91 24 21.08"/></g></g></svg>
        <div class="label">cancel</div>
      </a>
      <a class="cta delete ${this.visible("edit")}" @click=${()=>this.onDelete()}>
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 16"><defs><style>.cls-1{fill-rule:evenodd;}</style></defs><title>delete</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><path class="cls-1" d="M24,16H7L0,8,7,0H24V16ZM7.91,2,2.66,8,7.9,14H22V2ZM14,6.59,16.59,4,18,5.41,15.41,8,18,10.59,16.59,12,14,9.41,11.41,12,10,10.59,12.59,8,10,5.41,11.41,4,14,6.59Z"/></g></g></svg>
        <div class="label">delete</div>
      </a>
      <a class="cta edit ${this.visible("list", "edit", "add")}" @click=${(e)=>this.onYaml(e.target.checked)}>
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64" width="64" height="64"><title>zoom</title><g class="nc-icon-wrapper" stroke-linecap="square" stroke-linejoin="miter" stroke-width="2" fill="#000000" stroke="#000000"><line data-color="color-2" x1="59" y1="59" x2="42.556" y2="42.556" fill="none" stroke-miterlimit="10"/><circle cx="27" cy="27" r="22" fill="none" stroke="#000000" stroke-miterlimit="10"/></g></svg>
        <div class="label">yaml</div>
      </a>
    </div>
  </div>
</form>
`;
    return xyz;
  }

  /**
   * Override this method to return the kubernetes Kind of the
   * resource, .e.g 'Host', or 'Mapping'.
   */
  kind() {
    throw new Error("please implement kind()")
  }

  /**
   * Override this method to make this object be read-only.
   * Default functionality is to check for an annotation that
   * allows editing.  Default is editable unless the annotation
   * is set to false. [NOTE: may want to switch this?]
   */
  readOnly() {
    let annotations = this.annotations;
    if (aes_res_editable in annotations) {
      return !annotations[aes_res_editable];
    }
    else {
      return false;
    }
  }

  /**
   * Override to extend the styles of this resource (see yaml download tab).
   */
  modifiedStyles() {
    return null;
  }

  /**
   * Return the source URI for this resource, if one exists.
   * In the case we have a source URI, provide a button next to the
   * Edit button which, when clicked, opens a window on that source URI.
   * Basically this is useful for tracking resources as they are applied
   * using GitOps, though the annotation must be applied in the GitOps
   * pipeline for this to work.
   */
  sourceURI() {
    /* Make sure we have annotations, and return the aes_res_source, or undefined */
    let annotations = this.annotations;
    if (aes_res_source in annotations) {
      return annotations[aes_res_source];
    }
    else {
      /* Return undefined (same as nonexistent property, vs. null) */
      return undefined;
    }
  }

  /**
   * Override this method to implement the save behavior of a
   * resource.  This method must return an object that will get
   * rendered with JSON.stringify and supplied as the 'spec:' portion
   * of the kubernetes yaml that is passed to 'kubectl apply'. For example:
   *
   *   class Host extends SingleResource {
   *     ...
   *     spec() {
   *       return {
   *         hostname: this.hostname().value,
   *         acmeProvider: {
   *           authority: this.provider().value,
   *           email: this.email().value
   *         }
   *       }
   *     ...
   *   }
   *
   * The above spec will result in the following yaml being applied:
   *
   *    ---
   *    apiVersion: getambassador.io/v2
   *    kind: Host
   *    metadata:
   *      name: rhs.bakerstreet.io
   *      namespace: default
   *    spec:
   *      hostname: rhs.bakerstreet.io
   *      acmeProvider:
   *        authority: https://acme-v02.api.letsencrypt.org/directory
   *        email: rhs@alum.mit.edu
   *
   */
  spec() {
    throw new Error("please implement spec()")
  }

  /**
   * Override this method to control how a resource renders everything
   * but the kubernetes metadata. This method needs to do the right
   * thing depending on the value of 'this.state.mode'. For example,
   * if the mode is detail, this should render all/most of the
   * contents of the spec and status portion of the resource. If it is
   * edit, it should render the contents as form inputs. If it is
   * list, it should show a compact summary of just the relevant
   * stuff.
   *
   * The <visible-modes> component provides a convenient way to
   * control the visibility of elements that you render. It will
   * display its contents when the current mode matches *any* of the
   * specified modes. For example:
   *
   *   <visible-modes list detail>Summary: ${summary}</visible-modes>
   *   <visible-modes detail>${long-explanation}</visible-modes>
   *
   * The summary will show if the current mode is list *or* if the
   * current mode is detail. The ${long-explanation} will only show if
   * the current mode is detail.
   *
   * You can use this to provide a detail view whose fields change
   * in-place to become editable:
   *
   *   Field: <visible-modes list>${value}</visible-modes>
   *          <visible-modes add edit><input type=text value=${value}/></visible-modes>
   *
   * You can also use this at a coarser granularity to render each
   * mode entirely differently:
   *
   *   renderResource() {
   *     return html`
   *       <visible-modes list>${this.renderList()}</visible-modes>
   *       <visible-modes detail>${this.renderDetail()}</visible-modes>
   *       <visible-modes edit>${this.renderEdit()}</visible-modes>
   *       <visible-modes add>${this.renderAdd()}</visible-modes>
   *     `
   *   }
   *
   */
  renderResource() {
    throw new Error("please implement renderResource()")
  }

  /**
   * To help the UI place buttons within the rectangle border (or more
   * precisely, to help the UI grow the rectangle border to fit all the
   * buttons, these two functions should be overridden if the renderResource
   * has fewer than four rows in edit mode and/or fewer than two rows in
   * add mode.
   * (Override these functions if the add and edit buttons on the right
   * side of the frame are extending below the bottom of the frame.)
   */
  minimumNumberOfAddRows() {
    return 2;
  }
  minimumNumberOfEditRows() {
    return 4;
  }

}

/**
 * The ResourceSet class is an abstract base class that is extended in
 * order to create a container widget for listing kubernetes resources
 * of a single Kind. The base class provides machinery to manage the
 * UI state of all the contained widgets, so we can have compact list
 * displays, expand/edit individual items, etc. (That particular
 * aspect of this component will be much less central once we have
 * repeat and we don't need to explicitly manage ephemeral UI state.)
 *
 * Another important aspect of the ResourceSet class is that it
 * asynchronously updates the data behind all of the contained
 * SingleResource widgets whenever the data changes on the server. The
 * data comes from the <aes-snapshot-provider> element defined in the
 * snapshots.js class. ResourceSet elements need to appear on a page
 * that includes the <aes-snapshot-provider> element.
 *
 * To implement a ResourceSet container element, you must extend this
 * class and override the following methods. See individual methods
 * for more details:
 *
 *   getResources(snapshot) --> extracts the right resources from the backend snapshot
 *
 *   render() --> tell us how to display the collection
 *
 */
export class ResourceSet extends LitElement {

  static get styles() {
    return css``;
  }
  // internal
  static get properties() {
    return {
      resources: {type: Array},
      _states: {type: Map},
      addState: {type: Object}
    };
  }

  // internal
  constructor() {
    super();
    this.resources = [];
    this._states = {};
    this.addState = new UIState();
    this.addState.mode = "off";
    Snapshot.subscribe(this.onSnapshotChange.bind(this))
  }

  /**
   * Override to true to prevent the Add button from showing up.
   */
  readOnly() {
    return false;
  }

  /**
   * This method is invoked with the snapshot of server state (aka the
   * watt snapshot). Said snapshot comes from the
   * /edge_stack/api/snapshot endpoint which can be found in webui.go
   *
   * This method can be overridden so long as the overriden
   * implementation uses super to invoke the original implementation.
   */
  onSnapshotChange(snapshot) {
    this.resources = this.getResources(snapshot);
    // sort so that we don't randomly change the order whenever we get an update
    this.resources.sort((a, b) => {
      return this.key(a).localeCompare(this.key(b));
    });
  }

  // internal
  key(resource) {
    return resource.kind + ":" + resource.metadata.namespace + ":" + resource.metadata.name;
  }

  // internal
  state(resource) {
    let key = this.key(resource);
    if (this._states[key] === undefined) {
      this._states[key] = new UIState()
    }
    return this._states[key]
  }

  /**
   * Override this method to extract the correct set of resources for
   * the ResourceSet to have. For example, if you wanted to display
   * all Host, resources, you would implement the following
   * getResource(snapshot) method:
   *
   *   getResources(snapshot) {
   *     return snapshot.getResources('Host')
   *   }
   *
   * See the SnapshotWrapper class in snapshot.js for all the APIs you
   * can use to extract resources from a snapshot.
   */
  getResources() {
    throw new Error("please implement getResources(snapshot)")
  }

  /**
   * Override to extend the styles of this resource (see yaml download tab).
   */
  modifiedStyles() {
    return null;
  }

  /**
   * Override renderInner to show control how the collection renders. Most of the time this should look like this:
   * See hosts.js for an example.
   */
  render() {
    return html`
<link rel="stylesheet" href="../styles/resources.css">
${this.modifiedStyles() ? this.modifiedStyles() : ""}
  ${this.renderInner()}
`;
  }

  renderInner() {
    throw new Error("please implement renderInner()")
  }

}

/**
 * The SortableResourceSet class is an abstract base class that is extended in
 * order to create a container widget for listing sorted kubernetes resources
 * of a single Kind. The SortableResourceSet extends ResourceSet, adding an
 * HTML selector for picking a "sort by" attribute.
 *
 * See ResourceSet.
 *
 * To implement a SortableResourceSet container element, you must extend this
 * class and override the following methods. See individual methods
 * for more details:
 *
 *   sortFn(sortByAttribute) --> must return a `compare` function, for the collection.sort()
 *     https://www.w3schools.com/js/js_array_sort.asp
 *
 *     sortFn(sortByAttribute) {
 *       return function(a, b) { return a[sortByAttribute] - b[sortByAttribute] };
 *     }
 *
 *   renderSet() --> tell us how to display the collection
 *
 */
export class SortableResourceSet extends ResourceSet {

  // internal
  static get properties() {
    return {
      sortFields: { type: Array },
      sortBy: { type: String },
    };
  }

  /**
   * @param sortFields: A non-empty Array of {value, label} Objects by which it is possible to sort the ResourceSet. eg.:
   *   super([
   *     {
   *       value: "name",         // String. When selected, the value will be passed as argument in `sortFn`.
   *       label: "Mapping Name"  // String. Display label for the HTML component.
   *     },
   *     ...
   *   ]);
   */
  constructor(sortFields) {
    super();
    if (!sortFields || sortFields.length === 0) {
      throw new Error('please pass `sortFields` to constructor');
    }
    this.sortFields = sortFields;
    this.sortBy = this.sortFields[0].value;
  }

  onChangeSortByAttribute(e) {
    this.sortBy = e.target.options[e.target.selectedIndex].value;
  }

  static get styles() {
    return css`
div.sortby {
    text-align: right;
}
div.sortby select {
  font-size: 0.85rem;
  border: 2px #c8c8c8 solid;
  text-transform: uppercase;
}
div.sortby select:hover {
  color: #5f3eff;
  transition: all .2s ease;
  border: 2px #5f3eff solid;
}
    `
  }

  renderInner() {
    return html`
${this.resources.sort(this.sortFn(this.sortBy)) && this.renderSet()}`
  }

  renderSet() {
    throw new Error("please implement renderSet()");
  }

  sortFn(sortByAttribute) {
    throw new Error("please implement sortFn(sortByAttribute)");
  }
}

/**
 * The UIState class holds the transient UI state of a kubernetes
 * resource widget, for example whether the widget is in detail or
 * list view, or whether we are editing it, or any error messages were
 * discovered when validating prior to save.
 *
 * The reason all this state needs to be kept in a separate class is
 * that the data associated with the resource itself (e.g. the labels,
 * spec, status, etc.), is all asynchronously updated whenever it
 * changes in kubernetes, and we don't want the UI state to reset
 * whenever this change happens and we need to rerender our widgets.
 *
 * Normally we would ensure this by using the repeat directive in our
 * html templates, and we could just hold this state as regular
 * properties inside our SingleResource class, but for now we need to
 * keep all that state here, and have our ResourceSet component
 * carefully manage the UIState objects for us.
 *
 * To add your own transient UI state:
 *
 * 1. In your SingleResource subclass, override the init() method and
 *    initialize any fields you would like:
 *
 *    class Mapping extends SingleResource {
 *       ...
 *       init() {
 *          this.state.mapping_selected = false
 *       }
 *       ...
 *
 * 2. In your renderResource() method, make use of this state:
 *
 *       ...
 *       renderResource() {
 *         ...
 *         return html`
 *         ...
 *         <button @click=${()=>this.state.mapping_selected=true}>Select</button>
 *         ...
 *         `
 *       }
 *       ...
 *
 */
export class UIState {

  // internal
  constructor() {
    this.mode = "list"; // one of add, edit, list, detail, off
    this.messages = [];
    this._init = false
  }

  // internal
  init(resource) {
    if (!this._init) {
      resource.init();
      this._init = true
    }
  }

  // internal
  renderErrors() {
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

}

/**
 * This is a utility component used in conjuction with the SingleResource
 * class to control visibility of elements in different modes. Within
 * a renderResource() method, you can use:
 *
 *    <visible-modes mode1 ... modeN>...</visible-modes>
 *
 * to control visibility. The contents of the element will be
 * displayed if the current mode is any one of the mode names provided
 * as an attribute.
 */

export class VisibleModes extends LitElement {

  // internal
  static get properties() {
    return {
      mode: {type: String}
    }
  }

  // internal
  constructor() {
    super();
    this.mode = "default"
  }

  /**
   * Render the contents of the <visible-modes...>...</visible-modes>
   * element based on the provided attributes and the mode of the
   * containing SingleResource element.
   *
   * The way this works is that the user supplies the list of modes as
   * attribute names. The containing SingleResource element searches
   * for all <visible-modes> elements contained within its shadowRoot
   * and sets the mode attribute to the current mode of the
   * SingleResource widget.
   */
  render() {
    let display = this.attributes.getNamedItem(this.mode) != null ? "inline" : "none";
    return html`<slot style="display:${display}"></slot>`
  }

}

customElements.define('visible-modes', VisibleModes);
