import {html} from 'https://cdn.pika.dev/-/lit-element/2.2.1/dist-es2019/lit-element.min.js'
import {Resource, Resources} from '/edge_stack/components/resources.js';

class Mapping extends Resource {

  kind() {
    return "Mapping"
  }

  reset() {
    super.reset()
    this.prefix().value = this.prefix().defaultValue
    this.target().value = this.target().defaultValue
  }

  prefix() {
    return this.shadowRoot.querySelector('input[name="prefix"]')
  }

  target() {
    return this.shadowRoot.querySelector('input[name="target"]')
  }

  spec() {
    return {
      prefix: this.prefix().value,
      service: this.target().value
    }
  }

  renderResource() {
    let resource = this.resource
    let spec = resource.spec
    let status = resource.status || {"state": "<none>"}
    let resourceState = status.state
    let reason = resourceState == "Error" ? `(${status.reason})` : ''

    return html`
  <div class="left">Prefix:</div>
  <div class="right">
    <span class="${this.visible("list")}">${spec.prefix}</span>
    <input class="${this.visible("edit", "add")}" type="text" name="prefix"  value="${spec.prefix}" />
  </div>

  <div class="left">Target:</div>
  <div class="right">
    <span class="${this.visible("list")}">${spec.service}</span>
    <input class="${this.visible("edit", "add")}" type="text" name="target"  value="${spec.service}" />
  </div>

  <div class="left ${this.visible("list", "edit")}">Status:</div>
  <div class="right ${this.visible("list", "edit")}">
    <span>${resourceState} ${reason}</span>
  </div>
`
  }

}

customElements.define('dw-mapping', Mapping)

export class Mappings extends Resources {

  key() {
    return 'Mapping'
  }

  render() {
    let newMapping = {
      metadata: {
        namespace: "default",
        name: ""
      },
      spec: {
        prefix: "",
        service: ""
      }
    }
    return html`
<dw-mapping
  .resource=${newMapping}
  .state=${this.addState}>
  <add-button></add-button>
</dw-mapping>
<div>
  ${this.resources.map(r => {
    return html`<dw-mapping .resource=${r} .state=${this.state(r)}></dw-mapping>`
  })}
</div>`
  }

}

customElements.define('dw-mappings', Mappings)
