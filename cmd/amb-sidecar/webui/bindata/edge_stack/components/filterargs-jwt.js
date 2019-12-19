import {LitElement, html, css} from '/edge_stack/vendor/lit-element.min.js'
import './filterpolicies-common.js';

class FilterArgsJWT extends LitElement {
  static get properties() {
    return {
      mode: {type: String}, // 'list' or 'edit'
      data: {type: Object}, // from YAML
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
    this.shadowRoot.querySelectorAll('dw-scope-values').forEach((el)=>{el.reset();});
  }

  constructor() {
    super();
    this.mode = "list";
    this.data = {
      scope: [],
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

  <dt>scope</dt>
  <dd style="padding-top: 0"><dw-scope-values
    .mode=${this.mode}
    .data=${this.data.scope}
    @change=${(ev)=>{this.value = {scope: ev.target.value}}}
  ></dw-scope-values></dd>

</dl>`;
  }
}
customElements.define('dw-filterargs-jwt', FilterArgsJWT);
