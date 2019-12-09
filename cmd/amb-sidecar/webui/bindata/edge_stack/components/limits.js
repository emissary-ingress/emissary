import {html} from '../vendor/lit-element.min.js'
import {SingleResource, SortableResourceSet} from './resources.js';
import './limit-set.js';
//MOREMORE do the new look for the limits page

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
  <div class="attribute-value" style="margin: 0.5em">
    <visible-modes add edit>Use "*" to match against any value.</visible-modes>
  </div>
  <div class="attribute-name">limits:</div>
  <div class="attribute-value">
    <dw-limit-set .mode=${this.state.mode} .limits=${limits} @change=${(e)=>this.limitsChanged(e.target)}></dw-limit-set>
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

  renderSet() {
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
<dw-limit .resource=${addLimit} .state=${this.addState}><add-button></add-button></dw-limit>
<div>
  ${this.resources.map(l => html`<dw-limit .resource=${l} .state=${this.state(l)}></dw-limit>`)}
</div>`
  }

}

customElements.define('dw-limits', Limits);
