import {LitElement, html, css} from '/edge_stack/vendor/lit-element.min.js'
import {Snapshot} from './snapshot.js'

class FilterRef extends LitElement {
  static get properties() {
    return {
      mode: {type: String}, // 'list' or 'edit'
      data: {type: Object}, // from YAML
      filters: {type: Object},
    };
  }

  constructor() {
    super();
    this.mode = "list";
    this.data = {
      name: "",
      namespace: "",
      onDeny: "break",
      onAllow: "continue",
      ifRequestHeader: null,
      'arguments': null,
    };
    this.filters = {};
    Snapshot.subscribe((snapshot)=>{
      this.filters = Object.fromEntries(snapshot.getResources("Filter").map((r)=>[`${r.metadata.name}.${r.metadata.namespace}`, r]));
    });
  }

  set value(newVal) {
    this._value = newVal;
    this.dispatchEvent(new Event("change"));
  }

  get value() {
    return JSON.parse(JSON.stringify(this._value || this.data));
  }

  reset() {
    this.shadowRoot.querySelectorAll('select').forEach((el)=>{el.value = el.querySelector('option[selected]').value;});
    // TODO: ifRequestHeader
    // TODO: arguments
    this._value = null;
  }

  render() {
    // name
    // namespace
    // onDeny
    // onAllow
    // ifRequestHeader
    // arguments
    // - JWT
    // - OAuth2
    return html`
<div class="row line">

  <div class="row-col margin-right justify-right">filter:</div>
  <div class="row-col">
    <visible-modes .mode=${this.mode} list><span>${this.data.name}.${this.data.namespace}</span></visible-modes>
    <visible-modes .mode=${this.mode} edit add><select
      @change=${(ev)=>{
        let qname = ev.target.value;
        let sep = qname.lastIndexOf('.');
        let name = qname.slice(0, sep);
        let namespace = qname.slice(sep+1);
        this.value = {
          ...this.value,
          name: name,
          namespace: namespace,
        };
      }}
    >
      ${Object.entries(this.filters).sort((a, b) => b[0].localeCompare(a[0])).map(([k, v])=>{
        return html`<option ?selected=${k===`${this.data.name}.${this.data.namespace}`} value=${k}>${k}</option>`;
      })}
    </select></visible-modes> 
  </div>

  <div class="row-col margin-right justify-right">onDeny:</div>
  <div class="row-col">
    <visible-modes .mode=${this.mode} list><span><tt>${this.data.onDeny}</tt></span></visible-modes>
    <visible-modes .mode=${this.mode} edit add><select
      @change=${(ev)=>{this.value = {...this.value, onDeny: ev.target.value};}}
    >
      <option ?selected=${this.data.onDeny==="break"} value="break"><tt>break</tt> (default)</option>
      <option ?selected=${this.data.onDeny==="continue"} value="continue"><tt>continue</tt></option>
    </select></visible-modes> 
  </div>

  <div class="row-col margin-right justify-right">onAllow:</div>
  <div class="row-col">
    <visible-modes .mode=${this.mode} list><span><tt>${this.data.onAllow}</tt></span></visible-modes>
    <visible-modes .mode=${this.mode} edit add><select
      @change=${(ev)=>{this.value = {...this.value, onAllow: ev.target.value};}}
    >
      <option ?selected=${this.data.onAllow==="break"} value="break"><tt>break</tt></option>
      <option ?selected=${this.data.onAllow==="continue"} value="continue"><tt>continue</tt> (default)</option>
    </select></visible-modes> 
  </div>

</div>
`;
  }
}
customElements.define('dw-filterref', FilterRef);

class FilterRefList extends LitElement {
  static get properties() {
    return {
      mode: {type: String}, // 'list' or 'edit'
      data: {type: Array}, // from YAML
      namespace: {type: String},
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
    this.shadowRoot.querySelectorAll('dw-filterref').forEach((el)=>{el.reset();});
    this._value = null;
  }

  // implement
  constructor() {
    super();
    this.mode = "list";
    this.data = [];
  }

  // implement
  render() {
    let newRefData = {
      name: "",
      namespace: this.namespace,
      onDeny: "break",
      onAllow: "continue",
      ifRequestHeader: null,
      'arguments': null,
    };
    return html`
<visible-modes list>
  ${this.data.length == 0 ? html`(none)` : html``}
</visible-modes>

<ul>
${this.data.map((refData, i) => {
  return html`<li>
    <dw-filterref
      .mode=${this.mode}
      .data=${refData}
      @change=${(ev)=>{
        let dat = this.value;
        dat[i] = ev.target.value;
        this.value = dat;
      }}
    ></dw-filterref>
  </li>`;
})}
</ul>

<visible-modes add edit>
  <button
    @click=${(ev)=>{this.value = [...this.value, newRefData];}}
  >Add filter reference</button>
</visible-modes>
`;
  }

}
customElements.define('dw-filterref-list', FilterRefList);

