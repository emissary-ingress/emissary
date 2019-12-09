import {LitElement, html, css} from '/edge_stack/vendor/lit-element.min.js'

/**
 * This is the UI representation of the limits field of a rate limit
 * CRD spec. Please see the long comment at the beginning of
 * request-labels.js for an overview.
 *
 * The patterns in a limit set are intended to match against the
 * labels specified in the mappings tab. For that reason we want to
 * keep this UI visually similar to the labels UI that you access when
 * you edit mappings.
 */
export class LimitSet extends LitElement {

  // internal
  static get properties() {
    return {
      mode: {type: String}, // list or edit
      limits: {type: Array} // holds the limit set in yaml form
    }
  }

  // internal
  constructor() {
    super();
    this.mode = "list";
    this.limits = [];
  }

  // internal
  changed() {
    this.dispatchEvent(new Event("change"));
  }

  // internal
  addLimit() {
    console.log("adding", this.limits);
    this.limits.push({
      rate: "",
      unit: "minute",
      pattern: [{"generic_key": ""}]
    });
    this.requestUpdate("limits");
    this.changed();
    console.log("adding after", this.limits);
  }

  // internal
  limitChanged(index, limit) {
    if (limit.pattern.length === 0) {
      this.limits.splice(index, 1);
    } else {
      this.limits[index] = limit.yaml();
    }
    this.requestUpdate("limits");
    this.changed();
  }

  // implement
  render() {
    return html`
<visible-modes list>
${this.limits.length == 0 ? html`(none)` : html``}
</visible-modes>
${this.limits.map((limit, index)=> {
  return html`
      <dw-limit-pattern
        .mode=${this.mode}
        .pattern=${limit.pattern}
        .rate=${limit.rate}
        .unit=${limit.unit}
        @change=${(e)=>this.limitChanged(index, e.target)}></dw-limit-pattern>
    `
})}
<visible-modes add edit>
  <button @click=${this.addLimit.bind(this)}>Add</button>
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

customElements.define('dw-limit-set', LimitSet);

/**
 * This is the UI representation of a limit pattern which is intended
 * to be a pattern that maps against the request lables specified on a
 * mapping. We want to keep this visually similar to the UI for
 * editing labels so that the parallel is obvious to users.
 */
export class LimitPattern extends LitElement {

  // internal
  static get properties() {
    return {
      mode: { type: String },
      pattern: { type: Array },
      rate: { type: Number },
      unit: { type: String }
    }
  }

  // internal
  constructor() {
    super();
    this.mode = "list";
    this.pattern = [];
    this.rate = 0;
    this.unit = "minute";
    this.dragging = null;
  }

  // produce the kubernetes yaml from our state
  yaml() {
    return {
      pattern: this.pattern,
      rate: this.rate,
      unit: this.unit
    };
  }

  // convert from kubernetes yaml to our state
  splitElement(element) {
    let key = Object.keys(element)[0];
    let value = element[key];
    var type
    if (key === "remote_address") {
      type = "client"
    } else if (key === "generic_key") {
      type = "global"
    } else {
      type = "header"
    }
    return [type, key, value]
  }

  // internal
  changed() {
    this.dispatchEvent(new Event("change"));
  }

  // internal
  addElement() {
    this.pattern.push({"": ""});
    this.requestUpdate("pattern");
    this.changed();
  }

  // internal
  removeElement(index) {
    this.pattern.splice(index, 1);
    this.requestUpdate("pattern");
    this.changed();
  }

  // internal
  swapElements(a, b) {
    [this.pattern[a], this.pattern[b]] = [this.pattern[b], this.pattern[a]]
    this.requestUpdate("pattern");
    this.changed();
  }

  // internal
  elementChanged(index, element) {
    this.pattern[index] = element.yaml();
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
  <dw-pattern-element
    .mode=${this.mode}
    .type=${type}
    .key=${key}
    .value=${value}
    @change=${(e)=>this.elementChanged(index, e.target)}
  ></dw-pattern-element>
  <visible-modes add edit>
    <button @click=${()=>this.removeElement(index)}>-</button>
    ${index === this.pattern.length-1 ? html`<button @click=${this.addElement.bind(this)}>+</button>` : html``}
  </visible-modes>
</div>
`
  }

  // implement
  render() {
    return html`
<div style="margin-bottom: 1em">
  <div>
    <visible-modes list>
    <span>${this.rate} requests per ${this.unit}</span> <span style="margin-left: 0.5em">when:</span>
    </visible-modes list>

    <visible-modes add edit>
    <input type="text" .value=${this.rate} @change=${(e)=>{this.rate=parseInt(e.target.value); this.changed()}}/>
    requests per
    <select @change=${(e)=>{this.unit = e.target.value; this.changed()}}>
      <option .selected=${this.unit === "second"} value="second">second</option>
      <option .selected=${this.unit === "minute"} value="minute">minute</option>
      <option .selected=${this.unit === "hour"} value="hour">hour</option>
      <option .selected=${this.unit === "day"} value="day">day</option>
    </select>
    <span style="margin-left: 0.5em">when:</span>
    </visible-modes add edit>
  </div>

  <div style="margin-left: 1em">
    ${this.pattern.map(this.renderElement.bind(this))}
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

customElements.define('dw-limit-pattern', LimitPattern);

/**
 * This is the UI for specifying a pattern element. A pattern element
 * matches against the request label element specified in the mappings
 * tab when you edit labels. We want to keep this visually similar to
 * that, so the parallel is obvious, or at least less obtuse.
 */
class PatternElement extends LitElement {

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
      return {"remote_address": this.value};
    } else if (this.type === "global") {
      return {"generic_key": this.value};
    } else if (this.type === "header") {
      let result = {};
      result[this.key] = this.value;
      return result;
    }
  }

  // internal
  keyDisabled() {
    return this.type === "global" || this.type === "client";
  }

  // internal
  valueDisabled() {
    return false;
  }

  // implement
  render() {
    let key_disabled =  this.keyDisabled() ? "display: none" : "";
    let value_disabled =  this.valueDisabled() ? "display: none" : "";
    // XXX: the extra span in the list portion below seems to be
    // necessary to ensure there is a space between the key and value
    // after we hit cancel or save
    return html`
<visible-modes list add edit>
  <select ?disabled=${this.mode === "list"} @change=${(e)=>{this.type = e.target.value; this.changed()}}>
    <option .selected=${this.type === "global"} value="global">Global</option>
    <option .selected=${this.type === "header"} value="header">Header</option>
    <option .selected=${this.type === "client"} value="client">Client</option>
  </select>
</visible-modes>
<visible-modes list>
  <span style=${key_disabled}>${this.key}</span> <span>=</span> <span style=${value_disabled}>${this.value}</span>
</visible-modes>
<visible-modes add edit>
  <input style=${key_disabled} type="text" .value="${this.key}"
         @change=${(e)=>{this.key=e.target.value; this.changed()}}></input>
  <span>=</span>
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

customElements.define('dw-pattern-element', PatternElement);
