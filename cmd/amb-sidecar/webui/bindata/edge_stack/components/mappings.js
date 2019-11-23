import {html} from 'https://cdn.pika.dev/-/lit-element/2.2.1/dist-es2019/lit-element.min.js'
import {Resource, Resources} from '/edge_stack/components/resources.js';

class Mapping extends Resource {

  kind() {
    return "Mapping"
  }

  /*
   * In addition to the attributes supplied by my parent class (Resource)
   * (those attributes are metadata.name and metadata.namespace) the attributes
   * of a Mapping are: prefix and target.
   */
  reset() {
    super.reset();
    this.prefix().value = this.prefix().defaultValue;
    this.target().value = this.target().defaultValue;
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

  /*
   * The rendering functions in this file use the CSS styles defined in the resources.js
   * file. It's a little confusing and it's a left-over/consequence of this class
   * being a subclass of the more general Resource class. In the future we might find
   * a better way to encapsulate the styles; or not.
   */

  /*
   * A Mapping renders itself differently for each of its four states:
   *   off: ??
   *   list: shows the read-only version of the object
   *   edit: shows the editable version of the object (all the input fields)
   *   add: shows the add version of the object, similar to the edit version
   */
  render() {
    if( this.state.mode === "off") {
      return this.render_off_mode();
    } else {
      if( this.state.mode === "list") {
        return this.render_list_mode();
      } else if( this.state.mode === "edit") {
        return this.render_edit_mode();
      } else if( this.state.mode === "add") {
        return this.render_add_mode();
      } else {
        /* there should always be a mode, so we should never get here, but if
         * we do, we render in the default way, which is list mode. */
        return this.render_list_mode();
      }
    }
  }
  render_off_mode() {
    /*
     * TODO this mode is not completed
     */
    return html`
<slot @click=${this.onAdd.bind(this)}></slot>
`
  }
  render_list_mode() {
    /*
     * The list (read-only) version of the object is an expand/collapse
     * object. It shows a summary when collapsed and then a complete
     * version when expanded. In HTML, we generate both of those versions
     * at the same time and then use the display: none to turn one of the
     * versions off.
     */
    let resourceState = status.state
    let reason = resourceState == "Error" ? `(${status.reason})` : ''
    return html`
<div class="frame-no-grid">
    <div class="collapsed" id="collapsed-div">
      <div class="up-down-triangle" @click=${() => this.onExpand()}></div>
      <div class="grid">
        <div class="left">
          <span>${this.resource.metadata.name}</span>
          (<span>${this.resource.metadata.namespace}</span>)
        </div>
        <div class="right">
          <span class="code">${this.resource.spec.prefix}</span>
        </div>
      </div>
    </div>
    <div class="expanded off" id="expanded-div">
      <div class="up-down-triangle" @click=${() => this.onCollapse()}></div>
      <div class="grid">
        <div class="left">
          <span>${this.resource.metadata.name}</span>
          (<span>${this.resource.metadata.namespace}</span>)
        </div>
        <div class="right">
          <span class="code">${this.resource.spec.prefix}</span>
        </div>
        <div class="left" style="text-align:right;">
          &rArr;
        </div>
        <div class="right">
          <span>${this.resource.spec.service}</span>
        </div>
        <div class="both">
           <span>${resourceState} ${reason}</span>
        </div>
      </div>
    </div>
</div>
`
  }
  render_edit_mode() {
    /*
     * TODO this mode is not completed
     */
    return html`MOREMOREedit ` // TODO
  }
  render_add_mode() {
    /*
     * TODO this mode is not completed
     */
    return html`MOREMOREedit ` // TODO
  }
  /*
   * The onExpand and onCollapse functions are triggered by clicking on the
   * expand/collapse triangles and they act by hiding one of the two divs and
   * showing the other one. The two divs (expanded and collapsed) are produced
   * by render_list_mode().
   */
  onExpand() {
    this.shadowRoot.getElementById("collapsed-div").classList.add("off");
    this.shadowRoot.getElementById( "expanded-div").classList.remove("off");
  }
  onCollapse() {
    this.shadowRoot.getElementById("collapsed-div").classList.remove("off");
    this.shadowRoot.getElementById( "expanded-div").classList.add("off");
  }

  /*
   * This is old code that needs to be removed and it is fully converted to the new code.
    old_render() {
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

  old_renderResource() {
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
  */

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
