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
    this.shadowRoot.querySelectorAll('dw-scope-values').forEach((el)=>{el.reset();});
    this._value = null;
  }

  constructor() {
    super();
    this.mode = "list";
    this.data = {
      scope: [],
    };
  }

  render() {
    return html`
<div class="row line">

  <div class="row-col margin-right justify-right">scope:</div>
  <div class="row-col">
    <dw-scope-values
      .mode=${this.mode}
      .data=${this.data.scope}
      @change=${(ev)=>{this.value = {scope: ev.target.value}}}
    ></dw-scope-values>
  </div>

</div>
      `;
  }
}
customElements.define('dw-filterargs-jwt', FilterArgsJWT);
