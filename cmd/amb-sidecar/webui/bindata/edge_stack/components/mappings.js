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

  render() {
    return html`
<slot class="${this.state.mode == "off" ? "" : "off"}" @click=${this.onAdd.bind(this)}></slot>
<div class="${this.state.mode == "off" ? "off" : "frame"}">
    <div class="left">
      <span class="${this.visible("list", "edit")}">${this.resource.metadata.name}</span
          ><input class="${this.visible("add")}" name="name" type="text" value="${this.resource.metadata.name}"/>
      (<span class="${this.visible("list", "edit")}">${this.resource.metadata.namespace}</span
          ><input class="${this.visible("add")}" name="namespace" type="text" value="${this.resource.metadata.namespace}"/>)
    </div>
    <div class="right">
      <span class="${this.visible("list")}"><span class="code">${this.resource.spec.prefix}</span></span
          ><input class="${this.visible("edit", "add")}" type="text" name="prefix" value="${this.resource.spec.prefix}" />
    </div>
</div>
</div>`
    //TODO MOREMORE expandable
  }

    old_render() {  //TODO
      return html`
<slot class="${this.state.mode == "off" ? "" : "off"}" @click=${this.onAdd.bind(this)}></slot>
<div class="${this.state.mode == "off" ? "off" : "frame"}">
  <div class="title">
    ${this.kind()}: <span class="${this.visible("list", "edit")}">${this.resource.metadata.name}</span>
          <input class="${this.visible("add")}" name="name" type="text" value="${this.resource.metadata.name}"/>


      (<span class="${this.visible("list", "edit")}">${this.resource.metadata.namespace}</span><input class="${this.visible("add")}" name="namespace" type="text" value="${this.resource.metadata.namespace}"/>)</div>

  ${this.renderResource()}

  <div class="both">
    <label>
      <button class="${this.visible("list")}" @click=${() => this.onEdit()}>Edit</button>
      <button class="${this.visible("list")}" @click=${() => this.onDelete()}>Delete</button>
      <button class="${this.visible("edit", "add")}" @click=${() => this.onCancel()}>Cancel</button>
      <button class="${this.visible("edit", "add")}" @click=${() => this.onSave()}>Save</button>
    </label>
  </div>

  ${this.state.renderErrors()}
</div>`
    }

  old_renderResource() {  //TODO
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
