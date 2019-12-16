import {html} from '../vendor/lit-element.min.js'
import {SingleResource, SortableResourceSet} from './resources.js';
import './limit-set.js';

export class Limit extends SingleResource {

  static get properties() {
    let copy = JSON.parse(JSON.stringify(super.properties));
    copy["limits"] = {type: Array};
    return copy
  }

  constructor() {
    super();
    this.limits = [];
  }

  spec() {
    return {
      domain: "ambassador",
      limits: this.limits
    }
  }

  kind() {
    return "RateLimit"
  }

  onEdit() {
    super.onEdit();
    this.limits = ( this.resource.spec.limits || [] )
  }

  limitsChanged(limitSet) {
    this.limits = limitSet.limits;
  }

  renderResource() {
    let spec = this.resource.spec;
    let limits = this.state.mode === "edit" || this.state.mode === "add" ? this.limits : spec.limits || [];
    return html`
<div class="row line">
  <div class="row-col margin-right justify-right">limits:</div>
  <div class="row-col">
    <dw-limit-set .mode=${this.state.mode} .limits=${limits} @change=${(e)=>this.limitsChanged(e.target)}></dw-limit-set>
  </div>
</div>
`
  }
  // Override because we only have one row in the renderResource
  minimumNumberOfAddRows() {
    return 1;
  }
  minimumNumberOfEditRows() {
    return 1;
  }

}

customElements.define('dw-limit', Limit);

export class Limits extends SortableResourceSet {

  constructor() {
    super([
      {value: "name", label: "Name"},
      {value: "namespace", label: "Namespace"}
    ]);
  }

  getResources(snapshot) {
    return snapshot.getResources("RateLimit");
  }

  sortFn(sortByAttribute) {
    return function(r1, r2) {
      if (sortByAttribute === "name" || sortByAttribute === "namespace") {
        return r1.metadata[sortByAttribute].localeCompare(r2.metadata[sortByAttribute]);
      } else {
        return r1.spec[sortByAttribute].localeCompare(r2.spec[sortByAttribute]);
      }
    }
  }

  renderInner() {
    let shtml = super.renderInner();
    let addLimit = {
      metadata: {
        namespace: "default",
        name: ""
      },
      spec: {
        domain: "ambassador"
      },
      status: {}};
    return html`
<div class="header_con">
  <div class="col">
    <img alt="ratelimits logo" class="img" src="../images/svgs/ratelimits.svg">
      <defs><style>.cls-1{fill:#fff;}</style></defs>
        <g id="Layer_2" data-name="Layer 2">
          <g id="Layer_1-2" data-name="Layer 1"></g>
        </g>
    </img>
  </div>
  <div class="col">
    <h1>Rate Limits</h1>
    <p>Rate limits for different request classes.</p>
  </div>
  <div class="col2">
    <a class="cta add ${this.readOnly() ? "off" : ""}" @click=${()=>this.shadowRoot.getElementById("add-limit").onAdd()}>
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
<dw-limit id="add-limit" .resource=${addLimit} .state=${this.addState}></dw-limit>
${shtml}
`;

  }

  renderSet() {
    return html`
<div>
  ${this.resources.map(l => html`<dw-limit .resource=${l} .state=${this.state(l)}></dw-limit>`)}
</div>`
  }

}

customElements.define('dw-limits', Limits);
