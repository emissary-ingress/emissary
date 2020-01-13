import {LitElement, html} from '/edge_stack/vendor/lit-element.min.js'

/**
 * How envoy's rate limiting works:
 *
 *   When envoy's rate limiting functionality is enabled, envoy will
 *   associated a set of "descriptors" with each request, and pass
 *   that set to the rate limit service in order to get back a yes/no
 *   answer about whether to allow the request through or rate limit
 *   the request by responding with an http status code of 429.
 *
 *   A descriptor is a domain plus an ordered list of key-value pairs.
 *
 *   The domain is a string and right now is constrained to be
 *   "ambassador" since there can (or at the time there could) be only
 *   one for an entire envoy.
 *
 *   Envoy constructs the descriptors based on user supplied
 *   configuration. Each pair in a descriptor can be constructed based
 *   on an oddly specific set of options that envoy provides:
 *
 *    - ("generic_key", <user-supplied-value>)
 *    - (<user-supplied-key>, <value-from-user-specified-header>)
 *    - ("remote_address", <the-remote-ip-address>)
 *
 *   The oddly specific portion here is that even though you can
 *   control the key that is used when you tell envoy to plug in the
 *   value from a request header, you cannot control the key that is
 *   used in the other two cases.
 * 
 * How our rate limiting service (based on the lyft rate limiting
 * service) works:
 *
 *   The descriptor that envoy sends the rate limit service forms a
 *   redis key that is used to track request counts. All requests that
 *   share the same key will count against the same "request bucket"
 *   in redis.
 *
 *   The limits themselves are specified as patterns that match
 *   against a key. For example, if you have the following descriptor:
 *
 *    - ("generic_key", "per-user")
 *    - ("remote_address", "1.2.3.4")
 *
 *   Then a rate limit can be specified like so (along with a rate of course):
 *
 *    - ("generic_key", "per-user")
 *    - ("remote_address", "*")
 *
 *   The rate will apply for any remote_address. You can also override
 *   this with more specific patterns, e.g. the following would let
 *   you give 4.3.2.1 a different rate limit than the default:
 * 
 *    - ("generic_key", "per-user")
 *    - ("remote_address", "4.3.2.1")
 *
 * How we have surfaced the descriptors functionality:
 *
 *   We chose to rename descriptors as request labels. All request
 *   labels need to be under the obligatory ambassador domain:
 *
 *   spec: (of a mapping)
 *     ...
 *     labels:
 *       ambassador:
 *        - ignored_label_name1: [<label_element_1>, ..., <label_element_N>]
 *        - ...
 *        - ignored_label_nameN: [<label_element_1>, ..., <label_element_N>]
 *   
 *   Each mapping can define zero or more request labels. Each request
 *   label has a name that is ignored. Only the value of the label
 *   matters to the system.
 *
 *   The label elements are one of "remote_address", an arbitrary
 *   string, or the following map:
 *
 *    {<arbitrary_key>: {header: <header-name>}}
 *
 * Achieving Rate limiting:
 *
 *   In order to achieve rate limiting, you need to define labels, and
 *   specify patterns that match those labels. This can be difficult
 *   because there is little feedback in this process.
 *
 */


/**
 * This is the UI for everything under the obligatory ambassador
 * domain described above.
 */

export class RequestLabelSet extends LitElement {

  static get properties() {
    return {
      mode: {type: String}, // list or edit
      labels: {type: Array},
      addLabelName: {type: String}
    }
  }

  // internal
  constructor() {
    super();
    this.mode = "list";
    this.labels = [];
    this.addLabelName = "";
  }

  // internal
  changed() {
    this.dispatchEvent(new Event("change"));
  }

  // internal
  addLabel() {
    let label = {};
    label[`${this.addLabelName}`] = [""];
    this.labels.push(label);
    this.addLabelName = "";
    this.requestUpdate("labels");
    this.changed();
  }

  // internal
  labelChanged(index, label) {
    if (label.path.length == 0) {
      this.labels.splice(index, 1);
    } else {
      this.labels[index] = label.yaml();
    }
    this.requestUpdate("labels");
    this.changed();
  }

  // implement
  render() {
    return html`
<visible-modes list>
${this.labels.length == 0 ? html`(none)` : html``}
</visible-modes>
${this.labels.map((label, index)=> {
  let name = Object.keys(label)[0];
  let path = label[name];
  return html`
      <dw-request-label .mode=${this.mode} .name=${name} .path=${path}
                         @change=${(e)=>this.labelChanged(index, e.target)}></dw-request-label>
    `
})}
<visible-modes edit>
  <input type="text" .value=${this.addLabelName} @change=${(e)=>{this.addLabelName = e.target.value}}/>
  <button @click=${this.addLabel.bind(this)}>Add</button>
</visible-modes>
`
  }

  // internal
  updated() {
    // XXX: find a better way to do this
    this.shadowRoot.querySelectorAll("visible-modes").forEach((vm)=>{
      vm.mode = this.mode
    })
  }

}

customElements.define('dw-request-labels', RequestLabelSet);

/**
 * This is an invididual label aka a descriptor, which is in and of
 * itself a list of key value pairs.
 */
export class RequestLabel extends LitElement {

  // internal
  static get properties() {
    return {
      mode: { type: String },
      name: { type: String },
      path: { type: Array }
    }
  }

  // internal
  constructor() {
    super();
    this.mode = "list";
    this.name = "";
    this.path = [];
    this.dragging = null;
  }

  // produce the kubernetes yaml from our state
  yaml() {
    let result = {};
    result[this.name] = this.path;
    return result;
  }

  // internal
  changed() {
    this.dispatchEvent(new Event("change"));
  }

  // parse kubernetes yaml into the state as we would like to
  // represent it
  splitElement(e) {
    if (e === "remote_address") {
      return ["client", "", ""]
    } else if (typeof e === "string") {
      return ["global", "", e]
    } else {
      let key = Object.keys(e)[0]
      return ["header", key, e[key].header]
    }
  }

  // internal
  addElement() {
    this.path.push("");
    this.requestUpdate("path");
    this.changed();
  }

  // internal
  removeElement(index) {
    this.path.splice(index, 1);
    this.requestUpdate("path");
    this.changed();
  }

  // internal
  swapElements(a, b) {
    if (!this.path[a] || !this.path[b]) {
      console.error('At least one element is not part of the current label; skip the update');
      return;
    }
    [this.path[a], this.path[b]] = [this.path[b], this.path[a]];
    this.requestUpdate("path");
    this.changed();
  }

  // internal
  elementChanged(index, element) {
    this.path[index] = element.yaml();
    this.changed();
  }

  // internal
  renderElement(element, index) {
    let [type, key, value] = this.splitElement(element);

    return html`
<div draggable="true"
     @dragstart=${(e)=>{
       this.dragging = index;
     }}
     @dragover=${(e)=>{
       e.preventDefault();
     }}
     @drop=${(e)=>{
       this.swapElements(index, this.dragging);
     }}>
  <dw-label-element
    .mode=${this.mode}
    .type=${type}
    .key=${key}
    .value=${value}
    @change=${(e)=>this.elementChanged(index, e.target)}
  ></dw-label-element>
  <visible-modes edit>
    <button @click=${()=>this.removeElement(index)}>-</button>
    ${index === this.path.length-1 ? html`<button @click=${this.addElement.bind(this)}>+</button>` : html``}
  </visible-modes>
</div>
`
  }

  // implement
  render() {
    return html`
<div>

  <visible-modes list>
  <span>${this.name}</span>
  </visible-modes list>

  <visible-modes edit>
  <input type="text" .value=${this.name} @change=${(e)=>{this.name=e.target.value; this.changed()}}/>
  </visible-modes edit>

  <div style="margin-left: 1em">
    ${this.path.map(this.renderElement.bind(this))}
  </div>

</div>
`;
  }

  // internal
  updated() {
    // XXX: find a better way to do this
    this.shadowRoot.querySelectorAll("visible-modes").forEach((vm)=>{
      vm.mode = this.mode
    })
  }

}

customElements.define('dw-request-label', RequestLabel);

/**
 * A component for the label element. Each label element can be one of
 * Client, Header, or Global. A Client element is not user
 * customizable. A Global element requires you to supply a value. A
 * Header element requires you to supply a name and the header from
 * which to extract the value. You can flip them back and forth and
 * they remember what they were in case you change your mind.
 */
class LabelElement extends LitElement {

  // internal
  static get properties() {
    return {
      mode: { type: String },
      type: { type: String },
      key: { type: String },
      value: { type: String }
    }
  }

  // internal
  constructor() {
    super();
    this.mode = "list";
    this.type = "global";
    this.key = "";
    this.value = "";
  }

  // internal
  changed() {
    this.dispatchEvent(new Event("change"));
  }

  // construct kubernetes yaml from our internal state
  yaml() {
    if (this.type === "client") {
      return "remote_address";
    } else if (this.type === "global") {
      return this.value;
    } else if (this.type === "header") {
      let result = {};
      result[this.key] = {header: this.value};
      return result;
    }
  }

  // internal
  keyDisabled() {
    return this.type === "global" || this.type === "client";
  }

  // internal
  valueDisabled() {
    return this.type === "client";
  }

  // implement
  render() {
    let key_disabled =  this.keyDisabled() ? "display: none" : "";
    let value_disabled =  this.valueDisabled() ? "display: none" : "";
    // XXX: the extra span in the list portion below seems to be
    // necessary to ensure there is a space between the key and value
    // after we hit cancel or save
    return html`
<visible-modes list edit>
  <select ?disabled=${this.mode === "list"} @change=${(e)=>{this.type = e.target.value; this.changed()}}>
    <option .selected=${this.type === "global"} value="global">Global</option>
    <option .selected=${this.type === "header"} value="header">Header</option>
    <option .selected=${this.type === "client"} value="client">Client</option>
  </select>
</visible-modes>
<visible-modes list>
  <span style=${key_disabled}>${this.key}</span> <span> </span> <span style=${value_disabled}>${this.value}</span>
</visible-modes>
<visible-modes edit>
  <input style=${key_disabled} type="text" .value="${this.key}"
         @change=${(e)=>{this.key=e.target.value; this.changed()}}></input>
  <input style=${value_disabled} type="text" .value="${this.value}"
         @change=${(e)=>{this.value=e.target.value; this.changed()}}></input>
</visible-modes>
`;
  }

  // internal
  updated() {
    // XXX: find a better way to do this that doesn't involve copying this updated thing around
    this.shadowRoot.querySelectorAll("visible-modes").forEach((vm)=>{
      vm.mode = this.mode
    })
  }

}

customElements.define('dw-label-element', LabelElement);
