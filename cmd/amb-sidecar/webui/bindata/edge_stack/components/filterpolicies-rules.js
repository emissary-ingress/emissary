import {LitElement, html, css} from '/edge_stack/vendor/lit-element.min.js'
import './filterpolicies-rules-filterrefs.js';

class FilterPolicyRule extends LitElement {
  static get properties() {
    return {
      mode: {type: String}, // 'list' or 'edit'
      data: {type: Object}, // from YAML
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
    this.shadowRoot.querySelectorAll('input').forEach((el)=>{el.value = el.defaultValue;});
    this.shadowRoot.querySelectorAll('dw-filterref-list').forEach((el)=>{el.reset();});
    this._value = null;
  }

  constructor() {
    super();
    this.mode = "list";
    this.data = {
      host: "",
      path: "",
      filters: [],
    };
  }

  render() {
    return html`
<div class="row line">

  <div class="row-col margin-right justify-right">host:</div>
  <div class="row-col">
    <visible-modes .mode=${this.mode} list><span>${this.data.host}</span></visible-modes>
    <visible-modes .mode=${this.mode} edit add><input
      type="text"
      value="${this.data.host}"
      @change=${(ev)=>{this.value = {...this.value, host: ev.target.value};}}
    /></visible-modes>
  </div>

  <div class="row-col margin-right justify-right">path:</div>
  <div class="row-col">
    <visible-modes .mode=${this.mode} list><span>${this.data.path}</span></visible-modes>
    <visible-modes .mode=${this.mode} edit add><input
      type="text"
      value="${this.data.path}"
      @change=${(ev)=>{this.value = {...this.value, path: ev.target.value};}}
    /></visible-modes>
  </div>

  <div class="row-col margin-right justify-right">filters:</div>
  <div class="row-col">
    <dw-filterref-list
      .mode=${this.mode}
      .data=${this.data.filters||[]}
      .namespace=${this.namespace}
      @change=${(ev)=>{this.value = {...this.value, filters: ev.target.value};}}
    ></dw-filterref-list>
  </div>

</div>
`;
  }

}
customElements.define('dw-filterpolicy-rule', FilterPolicyRule);

class FilterPolicyRuleList extends LitElement {
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
    this.shadowRoot.querySelectorAll('dw-filterpolicy-rule').forEach((el)=>{el.reset();});
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
    let newRuleData = {
      host: "",
      path: "",
      filters: [],
    };
    return html`
<visible-modes list>
  ${this.data.length == 0 ? html`(none)` : html``}
</visible-modes>

<ul>
${this.data.map((ruleData, i) => {
  return html`<li>
    <dw-filterpolicy-rule
      .mode=${this.mode}
      .data=${ruleData}
      .namespace=${this.namespace}
      @change=${(ev)=>{
        let dat = this.value;
        dat[i] = ev.target.value;
        this.value = dat;
      }}
    ></dw-filterpolicy-rule>
  </li>`;
})}
</ul>

<visible-modes add edit>
  <button
    @click=${(ev)=>{this.value = [...this.value, newRuleData];}}
  >Add rule</button>
</visible-modes>
`;
  }
}
customElements.define('dw-filterpolicy-rule-list', FilterPolicyRuleList);
