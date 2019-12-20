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
    this._value = null;
    this.shadowRoot.querySelectorAll('input').forEach((el)=>{el.value = el.defaultValue;});
    this.shadowRoot.querySelectorAll('dw-filterref-list').forEach((el)=>{el.reset();});
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

  static get styles() {
    return css`
* {
  box-sizing: border-box;
}

:host {
  display: block
}

dl {
  display: grid;
  grid-template-columns: max-content;
  grid-gap: 0;
  margin: 0;
}
dl > dt {
  grid-column: 1 / 2;
  text-align: right;
	font-weight: 600;
}
dl > dt::after {
  content: ":";
}
dl > dd {
  grid-column: 2 / 3;
}
dl > * {
  margin: 0;
	padding: 10px 5px;
	border-bottom: 1px solid rgba(0, 0, 0, .1);
}
dl > :nth-last-child(2), dl > :last-child {
	border-bottom: none;
}
`;
  }

  render() {
    return html`<dl>

  <dt>host</dt>
  <dd>
    <visible-modes .mode=${this.mode} list><span>${this.data.host}</span></visible-modes>
    <visible-modes .mode=${this.mode} edit add><input
      type="text"
      value="${this.data.host}"
      @change=${(ev)=>{this.value = {...this.value, host: ev.target.value};}}
    /></visible-modes>
  </dd>

  <dt>path</dt>
  <dd>
    <visible-modes .mode=${this.mode} list><span>${this.data.path}</span></visible-modes>
    <visible-modes .mode=${this.mode} edit add><input
      type="text"
      value="${this.data.path}"
      @change=${(ev)=>{this.value = {...this.value, path: ev.target.value};}}
    /></visible-modes>
  </dd>

  <dt>filters</dt>
  <dd style="padding-top: 0">
    <dw-filterref-list
      .mode=${this.mode}
      .data=${this.data.filters||[]}
      .namespace=${this.namespace}
      @change=${(ev)=>{this.value = {...this.value, filters: ev.target.value};}}
    ></dw-filterref-list>
  </dd>

</dl>`;
  }

}
customElements.define('dw-filterpolicy-rule', FilterPolicyRule);

class FilterPolicyRuleList extends LitElement {
  static get properties() {
    return {
      mode: {type: String}, // 'list' or 'edit'
      data: {type: Array}, // from YAML
      value: {type: Array}, // same as data
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
    this._value = null;
    this.shadowRoot.querySelectorAll('dw-filterpolicy-rule').forEach((el)=>{el.reset();});
  }

  // implement
  constructor() {
    super();
    this.mode = "list";
    this.data = [];
  }

  static get styles() {
    return css`
* {
  box-sizing: border-box;
}
ul {
  margin: 0;
  padding: 0;
  list-style: none;
}
li::before {
  /* content */
  content: "—";
  font-weight: 600;

  /* same as a row */
	padding: 10px 5px;

  /* positioning */
  float: left;
  margin: 0;
  width: 1.1em;
  vertical-align: middle;
  text-align: center;
}
li {
	border-bottom: 1px solid rgba(0, 0, 0, .1);
}
li:last-child {
	border-bottom: none;
}
`;
  }

  // implement
  render() {
    let newRuleData = {
      host: "",
      path: "",
      filters: [],
    };
    return html`<div>

  ${((this.data||[]).length == 0 && this.mode != "add" && this.mode != "edit") ? html`(none)` : ``}

  <ul>

    ${(this.value||[]).map((ruleData, i) => {
      return html`<li><dw-filterpolicy-rule
        .mode=${this.mode}
        .data=${ruleData}
        .namespace=${this.namespace}
        @change=${(ev)=>{
          let dat = this.value;
          dat[i] = ev.target.value;
          this.value = dat;
        }}
      ></dw-filterpolicy-rule></li>`;
    })}

    ${ (this.mode === "add" || this.mode === "edit") ? html`
      <li><button
        @click=${(ev)=>{this.value = [...this.value, newRuleData];}}
      >Add rule</button></li>
    ` : html`` }

  </ul>

</div>`;
  }
}
customElements.define('dw-filterpolicy-rule-list', FilterPolicyRuleList);
