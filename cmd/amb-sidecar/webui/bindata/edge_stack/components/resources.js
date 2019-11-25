import { LitElement, html, css} from '/edge_stack/vendor/lit-element.min.js'
import {Snapshot} from '/edge_stack/components/snapshot.js'
import {getCookie} from '/edge_stack/components/cookies.js';

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

  /**
   * Override this to add custom styles. Make sure you include the
   * base class styles by including the result of super.styles() in
   * your result.
   */
  static get styles() {
    return css`
.error {
  color: red;
}
div {
/*  margin: 0.4em; */
}
div.frame {
  display: grid;
  margin: 0.4em;
  grid-template-columns: repeat(10, 10%);
  grid-gap: 0 0;
  border: 2px solid var(--dw-item-border);
  border-radius: 0.4em;
  position: relative;
  padding-bottom: 0.2em;
}
div.title {
  grid-column: 1 / 11;
  background: var(--dw-item-background-fill);
  margin: 0;
  padding: 0.5em;
  max-height: 1.3em;
}
div.title-button {
  position: absolute;
  top: 0.4em;
  right: 0.5em;
}
div.edit-buttons {
  position: absolute;
  right: 0.5em;
  top: 38px;
}
div.edit-buttons-column {
  display: grid;
  grid-template-columns: 3em;
  grid-gap:0.2em;
}
div.edit-buttons button {
  grid-column: 1 / 2;
}
div.edit-buttons button.delete-button {
  margin-top: 0.9em;
}
div.attribute-name {
  grid-column: 1 / 4;
  padding-right: 0.5em;
  text-align: right;
  font-variant: small-caps;
}
div.attribute-value {
  grid-column: 4 / 11;
}
div.error-value {
  grid-column: 3 / 11;
}
div.error-value ul {
  margin-block-start: 0.3em;
  padding-inline-start: 0;
}
div.error-value li {
  list-style-type: none;
}
.crd-name {
  font-weight: bold;
}
.crd-namespace {
  color: #888888;
  font-size: 80%;
}
.off { 
  display: none; 
}
span.code { 
  font-family: Monaco, monospace;
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
    this.state = new UIState()
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
  onDelete() {
    if( this.readOnly() ) {
      this.state.mode = "list";
      return; // we shouldn't be able to get here because there is no edit button,
              // and thus no delete button, but if we do, don't do anything.
    }
    fetch('/edge_stack/api/delete',
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
          if (r.ok) {
            // happy path
          } else {
            alert("BAD\n" + t) //TODO show the error in the UI somehow, this alert is not the best UI
          }
          if (this.state.mode === "add") {
            this.state.mode = "off"
          } else {
            this.state.mode = "list"
          }
          this.reset()
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
  validate() {}

  // internal
  name() {
    return this.resource.metadata.name;
  }

  // internal
  namespace() {
    return this.resource.metadata.namespace;
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

    let yaml = `
---
apiVersion: getambassador.io/v2
kind: ${this.kind()}
metadata:
  name: "${this.nameInput().value}"
  namespace: "${this.namespaceInput().value}"
spec: ${JSON.stringify(this.spec())}
`;

    fetch('/edge_stack/api/apply',
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
          } else {
            alert("BAD\n\n" + yaml + "\n\n" + t) //TODO show the error in the UI somehow, this alert is not the best UI
          }
          if (this.state.mode === "add") {
            this.state.mode = "off"
          } else {
            this.state.mode = "list"
          }
          this.reset()
        })
      })
  }

  // deprecated, use <visible-modes>...</visible-modes> instead
  visible() {
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
    return html`
<slot class="${this.state.mode === "off" ? "" : "off"}" @click=${this.onAdd.bind(this)}></slot>
<div class="${this.state.mode === "off" ? "off" : "frame"}">
  <div class="title-button ${this.visible("list" + (this.readOnly()?"/ro":""))}">
    <button @click=${()=>this.onEdit()}>Edit</button>
  </div>
  <div class="title">
    ${this.kind()}: <span class="crd-name ${this.visible("list", "edit")}">${this.name()}</span>
          <input class="${this.visible("add")}" name="name" type="text" value="${this.name()}"/>
      <span class="crd-namespace">(<span class="${this.visible("list", "edit")}">${this.namespace()}</span><input class="${this.visible("add")}" name="namespace" type="text" value="${this.namespace()}"/>)</span></div>

  ${this.renderResource()}

  ${((this.state.mode === "add") && (this.minimumNumberOfAddRows() < 2)) ? 
      html`<div class="attribute-value">&nbsp;</div>` : ""}
  ${((this.state.mode === "edit") && (this.minimumNumberOfEditRows() < 2)) ? 
      html`<div class="attribute-value">&nbsp;</div>` : ""}
  ${((this.state.mode === "edit") && (this.minimumNumberOfEditRows() < 3)) ? 
      html`<div class="attribute-value">&nbsp;</div>` : ""}
  ${((this.state.mode === "edit") && (this.minimumNumberOfEditRows() < 4)) ? 
      html`<div class="attribute-value">&nbsp;</div>` : ""}
  
  <div class="edit-buttons ${this.visible("edit", "add")}">
    <div class="edit-buttons-column">
      <button class="${this.visible("edit", "add")}" @click=${()=>this.onSave()}>Save</button>
      <button class="${this.visible("edit", "add")}" @click=${()=>this.onCancel()}>Cancel</button>
      <button class="${this.visible("edit")} delete-button" @click=${()=>this.onDelete()}>Delete</button>
    </div>
  </div>

  ${this.state.renderErrors()}
</div>`
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
   */
  readOnly() {
    return false;
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
   * This method is invoked with the snapshot of server state (aka the
   * watt snapshot). Said snapshot comes from the
   * /edge_stack/api/snapshot endpoint which can be found in webui.go
   *
   * This method can be overridden so long as the overriden
   * implementation uses super to invoke the original implementation.
   */
  onSnapshotChange(snapshot) {
    this.resources = this.getResources(snapshot)
  }

  // internal
  state(resource) {
    let key = resource.metadata.namespace + ":" + resource.metadata.name;
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
   * Override this to show control how the collection renders. Most of the time this should look like this:
   *
   *    render() {
   *      let addHost = {
   *        metadata: {
   *          namespace: "default",
   *          name: window.location.hostname
   *        },
   *        spec: {
   *          hostname: window.location.hostname,
   *          acmeProvider: {
   *            authority: "https://acme-v02.api.letsencrypt.org/directory",
   *            email: ""
   *          }
   *        },
   *        status: {}}
   *      return html`
   *  <dw-host .resource=${addHost} .state=${this.addState}><add-button></add-button></dw-host>
   *  <div>
   *    ${this.resources.map(h => html`<dw-host .resource=${h} .state=${this.state(h)}></dw-host>`)}
   *  </div>`
   *    }
   *
   * The key elements being:
   *
   *   a) define the default resource for when you click add
   *   b) include a single resource component (the <dw-host...>)
   *      for where you want add to be
   *   c) render the <dw-host> elements that form the existing
   *      resources and pass in the resource data and ui state
   *
   */
  render() {
    throw new Error("please implement render()")
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
<div class="error-value">
  <ul>
    ${this.messages.map(m=>html`<li><span class="error">${m}</span></li>`)}
  </ul>
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
