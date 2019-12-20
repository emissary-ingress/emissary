import {LitElement, html, css} from '/edge_stack/vendor/lit-element.min.js'

class HeaderFieldSelector extends LitElement {
  static get properties() {
    return {
      mode: {type: String}, // 'list' or 'edit'
      data: {type: Object}, // from YAML
    }
  }

  constructor() {
    super();
    this.mode = "list";
    this.data = {
      name: "",
      value: "",
    };
  }

  set rawValue(newVal) {
    this._value = newVal;
    this.dispatchEvent(new Event("change"));
  }

  get rawValue() {
    return JSON.parse(JSON.stringify(this._value || this.dataToRawValue(this.data || {})));
  }

  set value(val) {
    throw new Error("set .rawValue instead");
  }

  get value() {
    return this.rawValueToData( this.rawValue);
  }

  dataToRawValue(dat) {
    return {
      mode: (dat.name ? (dat.value ? "h_val" : "h_any") : "all"),
      name: (dat.name || ""),
      value: (dat.value || ""),
    };
  }

  rawValueToData(raw) {
    switch (raw.mode) {
    case "all":
      return {name: "", value: ""};
    case "h_any":
      return {name: raw.name, value: ""};
    case "h_val":
      return {name: raw.name, value: raw.value};
    }
  }

  reset() {
    this._value = null;
    this.shadowRoot.querySelectorAll('input').forEach((el)=>{el.value = el.defaultValue;});
    this.shadowRoot.querySelectorAll('select').forEach((el)=>{el.value = el.querySelector('option[selected]').value;});
  }

  static get styles() {
    return css`
* {
  box-sizing: border-box;
}

:host {
  display: block
}

dl > dt {
	font-weight: 600;
}
dl > dt::after {
  content: ":";
}
dl > dd {
	padding: 10px 5px;

  margin-left: 0;
  padding-left: 1.5em;
	border-bottom: 1px solid rgba(0, 0, 0, .1);
}
dl > dd:nth-last-child(2), dl > dd:last-child {
	border-bottom: none;
}
`;
  }

  render() {
    return html`<fieldset>

  <legend><select
      ?disabled=${this.mode !== "add" && this.mode !== "edit"}
      @change=${(ev)=>{this.rawValue = {...this.rawValue, mode: ev.target.value}}}
    >
    <option ?selected=${this.rawValue.mode==="all"} value="all">all requests</option>
    <option ?selected=${this.rawValue.mode==="h_any"} value="h_any">requests with header</option>
    <option ?selected=${this.rawValue.mode==="h_val"} value="h_val">requests with header a certain way</option>
  </select></legend>
  <dl>

    <dt style=${this.rawValue.mode==="all" ? "display: none" : ""}>name</dt>
    <dd style=${this.rawValue.mode==="all" ? "display: none" : ""}>
      <visible-modes .mode=${this.mode} list><span>${(this.data||{}).name}</span></visible-modes>
      <visible-modes .mode=${this.mode} edit add><input
        type="text"
        value=${(this.data||{}).name}
        @change=${(ev)=>{this.rawValue = {...this.rawValue, name: ev.target.value}}}
      /></visible-modes>
    </dd>

    <dt style=${this.rawValue.mode==="h_val" ? "" : "display: none"}>value</dt>
    <dd style=${this.rawValue.mode==="h_val" ? "" : "display: none"}>
      <visible-modes .mode=${this.mode} list><span>${(this.data||{}).value}</span></visible-modes>
      <visible-modes .mode=${this.mode} edit add><input
        type="text"
        value=${(this.data||{}).value}
        @change=${(ev)=>{this.rawValue = {...this.rawValue, value: ev.target.value}}}
      /></visible-modes>
    </dd>

  </dl>
</fieldset>`;
  }
}
customElements.define('dw-header-field-selector', HeaderFieldSelector);

class ScopeValues extends LitElement {
  static get properties() {
    return {
      mode: {type: String}, // 'list' or 'edit'
      data: {type: Array},
    };
  }

  set value(newVal) {
    this._value = newVal;
    this.dispatchEvent(new Event("change"));
  }

  get value() {
    return JSON.parse(JSON.stringify(this._value || this.data));
  }

  reset() {
    this._value = null;
    this.shadowRoot.querySelectorAll('input').forEach((el)=>{el.value = el.defaultValue;});
  }

  render() {
    return html`<div>

<visible-modes .mode=${this.mode} list>
  <ul>
    ${(this.data||[]).map((v) => html`<li>${v}</li>`)}
  </ul>
</visible-modes>

<visible-modes .mode=${this.mode} edit add>
  <label>enter space-separated scope values: <input
    type="text"
    value=${(this.data||[]).join(" ")}
    @change=${(ev)=>{this.value = ev.target.value.split(/\s+/)}}
  /></label>
</visible-modes>


</div>`;
  }
}
customElements.define('dw-scope-values', ScopeValues);
