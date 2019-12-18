import {html} from '../vendor/lit-element.min.js'
import {SingleResource, SortableResourceSet} from './resources.js';
import './filterpolicies-rules.js';

class FilterPolicy extends SingleResource {

  // override
  constructor() {
    super();
    this.formRules = [];
  }

  // implement
  kind() {
    return "FilterPolicy";
  }

  // implement
  spec() {
    return {
      rules: this.formRules
    };
  }

  // override
  reset() {
    super.reset();
    this.shadowRoot.querySelectorAll('dw-filterpolicy-rule-list').forEach((el)=>{el.reset();});
  }

  // implement
  renderResource() {
    return html`
<div class="row line">
  <div class="row-col margin-right justify-right">rules:</div>
  <div class="row-col">
    <dw-filterpolicy-rule-list
      .mode=${this.state.mode}
      .data=${this.resource.spec.rules}
      .namespace=${this.resource.metadata.namespace}
      @change=${(ev)=>{this.formRules = ev.target.rules;}}
    ></dw-filterpolicy-rule-list>
  </div>
</div>
`;
  }

  // override
  minimumNumberOfAddRows() {
    return 1;
  }

  // override
  minimumNumberOfEditRows() {
    return 1;
  }
}
customElements.define('dw-filterpolicy', FilterPolicy);

class FilterPolicies extends SortableResourceSet {
  // implement
  constructor() {
    super([
      {value: "name", label: "Name"},
      {value: "namespace", label: "Namespace"},
    ]);
  }

  // implement
  sortFn(sortByAttribute) {
    return function(a, b) {
      switch (sortByAttribute) {
      case "name":
      case "namespace":
        return a.metadata[sortByAttribute].localeCompare(b.metadata[sortByAttribute]);
      default:
        throw new Error("how did sortByAttribute get set wrong!?");
      }
    }
  }

  // implement
  getResources(snapshot) {
    return snapshot.getResources("FilterPolicy");
  }

  // implement
  renderInner() {
    let shtml = super.renderInner();
    let newFilterPolicy = {
      metadata: {
        namespace: "default",
        name: ""
      },
      spec: {
        rules: []
      }
    };
    return html`
<div class="header_con">
  <div class="col">
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32" width="32" height="32"><title>filterpolicy</title><g class="nc-icon-wrapper" stroke-linecap="square" stroke-linejoin="miter" stroke-width="2" fill="#608cee" stroke="#608cee"><polygon points="30 5 19 16 19 26 13 30 13 16 2 5 2 1 30 1 30 5" fill="none" stroke="#111111" stroke-miterlimit="10"/></g></svg>
  </div>
  <div class="col">
    <h1>FilterPolicies</h1>
    <p>Configure which middlewares apply to which requests.</p>
  </div>
  <div class="col2">
    <a class="cta add ${this.readOnly() ? "off" : ""}" @click=${()=>this.shadowRoot.getElementById("add-filterpolicy").onAdd()}>
      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 30 30"><defs><style>.cls-a{fill:none;stroke:#000;stroke-linecap:square;stroke-miterlimit:10;stroke-width:2px;}</style></defs><title>add_1</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><line class="cls-a" x1="15" y1="9" x2="15" y2="21"/><line class="cls-a" x1="9" y1="15" x2="21" y2="15"/><circle class="cls-a" cx="15" cy="15" r="14"/></g></g></svg>
      <div class="label">add</div>
    </a>
    <div class="sortby">
      <select id="sortByAttribute" @change=${this.onChangeSortByAttribute.bind(this)}>
    ${this.sortFields.map(f => {
      return html`<option value="${f.value}">${f.label}</option>`
    })}
      </select>
    </div>
  </div>
</div>
<dw-filterpolicy id="add-filterpolicy" .resource=${newFilterPolicy} .state=${this.addState}></dw-filterpolicy>
${shtml}
`;
  }

  // implement
  renderSet() {
    return html`
<div>
  ${this.resources.map(r => {
    return html`<dw-filterpolicy .resource=${r} .state=${this.state(r)}></dw-filterpolicy>`;
  })}
</div>
`;
  }
}
customElements.define('dw-filterpolicies', FilterPolicies);
