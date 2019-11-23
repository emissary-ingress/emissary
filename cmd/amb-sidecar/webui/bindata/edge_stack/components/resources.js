import { LitElement, html, css} from 'https://cdn.pika.dev/-/lit-element/2.2.1/dist-es2019/lit-element.min.js'
import {registerContextChangeHandler, useContext} from '/edge_stack/components/context.js'
import {getCookie} from '/edge_stack/components/cookies.js';

// holds the UI state of a kubernetes resource widget, this can be merged with Resource when we have repeat
export class UIState {

  constructor() {
    this.mode = "list" // one of add, edit, list, detail, off
    this.messages = []
    this._init = false
  }

  init(resource) {
    if (!this._init) {
      resource.init()
      this._init = true
    }
  }

  renderErrors() {
    if (this.messages.length > 0) {
      return html`
<div class="both">
  <ul>
    ${this.messages.map(m=>html`<li><span class="error">${m}</span></li>`)}
  </ul>
</div>`
    } else {
      return html``
    }
  }
}

export class Resource extends LitElement {

  static get styles() {
    return css`
.error {
  color: red;
}
div {
  margin: 0.4em;
}
div.frame {
  display: grid;
  grid-template-columns: 50% 50%;
  border: 2px solid #ede7f3;
  border-radius: 0.4em;
}
div.title {
  grid-column: 1 / 3;
  background: #ede7f3;
  margin: 0;
  padding: 0.5em;
}

/* -- -- -- -- -- -- -- -- -- -- -- --  
 * These styles are used in mappings.js
 */
/*
 * We separate the frame from the grid so that we can have different grids inside the frame.
 */
div.frame-no-grid {
  border: 2px solid #ede7f3;
  border-radius: 0.4em;
}
/*
 * Collapsed and expanded are used in the read-only list display of the Mappings.
 */
.collapsed div.up-down-triangle {
  float: left;
  margin-left: 0;
  margin-top: 0.25em;
  cursor: pointer;
}
.collapsed div.up-down-triangle::before {
  content: "\\25B7"
}
.expanded div.up-down-triangle {
  float: left;
  margin-left: 0;
  margin-top: 0.25em;
  cursor: pointer; 
}
.expanded div.up-down-triangle::before {
  content: "\\25BD"
}
/*
 * grid is used in the read-only list display of the Mappings
 */
div.grid {
  display: grid;
  grid-template-columns: 50% 50%;
}
div.grid div {
  margin: 0.1em;
}
.namespace {
  color: #989898;
  font-size: 80%;
}
/*
 * three-grid is used in the edit display of the Mappings
 * along with edit-field classes
 */
.edit-field {
  padding-left: 2em;
}
.edit-field-label {
  color: #202020;
}
.three-grid {
  display: grid;
  grid-template-columns: 40% 50% 10%;
}
.three-grid-all {
  grid-column: 1 / 4;
}
.three-grid-one {
  grid-column: 1 / 2;
  text-align: right;
  padding-right: 1em;
  margin: 0 0 0.25em 0;
}
.three-grid-two {
  grid-column: 2 / 3;
  margin: 0 0 0.25em 0;
}
.three-grid-three {
  grid-column: 3 / 4;
  margin: 0 0 0.25em 0;
}
.three-grid-two input[type=text] {
  width: 100%;
}
/*
 * one-grid is used in the edit display for the three action icons
 * on the right side
 */
.one-grid {
  grid-template-columns: 40px;
  margin-top: -0.2em;
}
.one-grid-one {
  grid-column: 1 / 2;
  margin: 0;
  padding: 0;
}
.edit-action-icon {
  cursor: pointer;
  width: 25px;
  height: 25px;
  padding: 0;
  margin: 0;
}
/*
 * End of styles for mappings.js
 *  -- -- -- -- -- -- -- -- -- -- -- --  */
 
div.left {
  grid-column: 1 / 2;
}
div.right {
  grid-column: 2 / 3;
}
div.both {
  grid-column: 1 / 3;
}
.off { display: none; }
span.code { 
  font-family: Monaco, monospace;
}

`
  }

  static get properties() {
    return {
      resource: {type: Map},
      state: {type: UIState},
    }
  }

  constructor() {
    super()
    this.resource = {}
    this.state = new UIState()
  }

  // This is invoked when the UI state is new... when we get repeat,
  // this will be able to go away and we can just use constructors.
  init() {}

  update() {
    if (this.state instanceof UIState) {
      this.state.init(this)
    }
    super.update()
  }

  onAdd() {
    this.requestUpdate()
    this.state.mode = "add"
  }

  onEdit() {
    this.requestUpdate()
    if (this.state.mode != "edit") {
      this.state.mode = "edit"
    } else {
      this.state.mode = "list"
    }
  }

  onDelete() {
    fetch('/edge_stack/api/delete',
          {
            method: "POST",
            headers: new Headers({
              'Authorization': 'Bearer ' + getCookie("edge_stack_auth")
            }),
            body: JSON.stringify({
              Namespace: this.resource.metadata.namespace,
              Names: [`${this.kind()}/${this.resource.metadata.name}`]
            })
          })
      .then(r=>{
        r.text().then(t=>{
          if (r.ok) {
            alert("OK\n" + t)
          } else {
            alert("BAD\n" + t)
          }
          if (this.state.mode == "add") {
            this.state.mode = "off"
          } else {
            this.state.mode = "list"
          }
          this.reset()
        })
      })
  }

  onCancel() {
    this.requestUpdate()

    if (this.state.mode == "add") {
      this.state.mode = "off"
    } else {
      this.state.mode = "list"
    }

    this.reset()
  }

  reset() {
    this.state.messages.length = 0
    this.name().value = this.name().defaultValue
    this.namespace().value = this.namespace().defaultValue
  }

  addError(message) {
    this.state.messages.push(message)
  }

  validate() {}

  name() {
    return this.shadowRoot.querySelector('input[name="name"]')
  }

  namespace() {
    return this.shadowRoot.querySelector('input[name="namespace"]')
  }

  onSave() {
    this.requestUpdate()

    this.state.messages.length = 0
    this.validate()
    if (this.state.messages.length > 0) {
      return
    }

    let yaml = `
---
apiVersion: getambassador.io/v2
kind: ${this.kind()}
metadata:
  name: "${this.name().value}"
  namespace: "${this.namespace().value}"
spec: ${JSON.stringify(this.spec())}
`

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
            alert("OK\n" + t)
          } else {
            alert("BAD\n\n" + yaml + "\n\n" + t)
          }
          if (this.state.mode == "add") {
            this.state.mode = "off"
          } else {
            this.state.mode = "list"
          }
          this.reset()
        })
      })
  }

  visible() {
    return [...arguments].includes(this.state.mode) ? "" : "off"
  }

  render() {
    return html`
<slot class="${this.state.mode == "off" ? "" : "off"}" @click=${this.onAdd.bind(this)}></slot>
<div class="${this.state.mode == "off" ? "off" : "frame"}">
  <div class="title">
    ${this.kind()}: <span class="${this.visible("list", "edit")}">${this.resource.metadata.name}</span>
          <input class="${this.visible("add")}" name="name" type="text" value="${this.resource.metadata.name}"/>


      (<span class="${this.visible("list", "edit")}">${this.resource.metadata.namespace}</span><input class="${this.visible("add")}" name="namespace" type="text" value="${this.resource.metadata.namespace}"/>)</div>

  ${this.renderResource()}

  <div class="both">
    <label>
      <button class="${this.visible("list")}" @click=${()=>this.onEdit()}>Edit</button>
      <button class="${this.visible("list")}" @click=${()=>this.onDelete()}>Delete</button>
      <button class="${this.visible("edit", "add")}" @click=${()=>this.onCancel()}>Cancel</button>
      <button class="${this.visible("edit", "add")}" @click=${()=>this.onSave()}>Save</button>
    </label>
  </div>

  ${this.state.renderErrors()}
</div>`
  }

  kind() {
    throw new Error("please implement kind()")
  }

  spec() {
    throw new Error("please implement spec()")
  }

  renderResource() {
    throw new Error("please implement renderResource()")
  }

}

export class Resources extends LitElement {

  static get properties() {
    return {
      resources: {type: Array},
      _states: {type: Map},
      addState: {type: Object}
    };
  }

  constructor() {
    super();

    const arr = useContext('aes-api-snapshot', null);
    if (arr[0] != null) {
      this.resources = arr[0][this.key()] || []
    } else {
      this.resources = []
    }
    this._states = {}
    this.addState = new UIState()
    this.addState.mode = "off"
    registerContextChangeHandler('aes-api-snapshot', this.onSnapshotChange.bind(this))
  }

  onSnapshotChange(snapshot) {
    let defaults = {}
    defaults[this.key()] = []
    let kube = snapshot['Kubernetes'] || defaults
    this.resources = kube[this.key()] || []
  }

  state(resource) {
    let key = resource.metadata.namespace + ":" + resource.metadata.name
    if (this._states[key] == undefined) {
      this._states[key] = new UIState()
    }
    return this._states[key]
  }

  key() {
    throw new Error("please implement key()")
  }

  render() {
    throw new Error("please implement render()")
  }

}
