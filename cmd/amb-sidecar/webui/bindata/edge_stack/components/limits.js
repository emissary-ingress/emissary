import {html} from 'https://cdn.pika.dev/-/lit-element/2.2.1/dist-es2019/lit-element.min.js'
import {SingleResource, ResourceSet} from '/edge_stack/components/resources.js';
import {getCookie} from '/edge_stack/components/cookies.js';

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
    let spec = this.resource.spec
    return html`
  <div class="left">Domain:</div>
  <div class="right">
    <visible-modes list detail>${spec.domain}</visible-modes>
    <visible-modes edit add><input type=text name="domain" value="${spec.domain}"/></visible-modes>
  </div>
`
  }

}

customElements.define('dw-limit', Limit)

export default class Limits extends ResourceSet {

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
      status: {}}
    return html`
<dw-limit .resource=${addLimit} .state=${this.addState}><add-button></add-button></dw-limit>
<div>
  ${this.resources.map(l => html`<dw-limit .resource=${l} .state=${this.state(l)}></dw-limit>`)}
</div>`
  }

}

customElements.define('dw-limits', Limits)
