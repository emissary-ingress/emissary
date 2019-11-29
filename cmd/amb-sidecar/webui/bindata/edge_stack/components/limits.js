import {html} from '../vendor/lit-element.min.js'
import {SingleResource, ResourceSet} from './resources.js';

export class Limit extends SingleResource {

  constructor() {
    super()
  }

  domain() {
    return this.shadowRoot.querySelector('input[name="domain"]')
  }

  spec() {
    return {
      domain: this.domain().value
    }
  }

  kind() {
    return "RateLimit"
  }

  renderResource() {
    let spec = this.resource.spec;
    return html`
  <div class="attribute-name">domain:</div>
  <div class="attribute-value">
    <visible-modes list detail>${spec.domain}</visible-modes>
    <visible-modes edit add><input type=text name="domain" value="${spec.domain}"/></visible-modes>
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

export class Limits extends ResourceSet {

  getResources(snapshot) {
    return snapshot.getResources("RateLimit")
  }

  render() {
    let addLimit = {
      metadata: {
        namespace: "default",
        name: ""
      },
      spec: {
        domain: ""
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
