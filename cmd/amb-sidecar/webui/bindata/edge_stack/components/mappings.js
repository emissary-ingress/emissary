import {html} from '/edge_stack/vendor/lit-element.min.js'
import {SingleResource, ResourceSet} from '/edge_stack/components/resources.js';

class Mapping extends SingleResource { //TODO need to abstract the changes I made to the outer Resource class for use in other sub-classes

  kind() {
    return "Mapping"
  }

  /*
   * In addition to the metadata.name and metadata.namespace attributes supplied
   * by my parent class (parent = Resource) the attributes of a Mapping are: prefix and target.
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
     * TODO the 'off' mode is not completed
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
    let resourceState = status.state;
    let reason = resourceState == "Error" ? `(${status.reason})` : '';
    return html`
<div class="frame-no-grid">
    <div class="collapsed" id="collapsed-div">
      <div class="up-down-triangle" @click=${() => this.onExpand()}></div>
      <div class="grid" @click=${() => this.onStartEdit()}>
        <div class="attribute-name">
          <span>${this.name()}</span>
          <span class="crd-namespace">(${this.namespace()})</span>
        </div>
        <div class="attribute-value">
          <span class="code">${this.resource.spec.prefix}</span>
        </div>
      </div>
    </div>
    <div class="expanded off" id="expanded-div">
      <div class="up-down-triangle" @click=${() => this.onCollapse()}></div>
      <div class="grid" @click=${() => this.onStartEdit()}>
        <div class="attribute-name">
          <span>${this.name()}</span>
          <span class="crd-namespace">(${this.namespace()})</span>
        </div>
        <div class="attribute-value">
          <span class="code">${this.resource.spec.prefix}</span>
        </div>
        <div class="attribute-name" style="text-align:right;">
          &rArr;
        </div>
        <div class="attribute-name">
          <span>${this.resource.spec.service}</span>
        </div>
        <div class="attribute-value">
           <span>${resourceState} ${reason}</span>
        </div>
      </div>
    </div>
</div>
`
  }
  render_edit_mode() {
    /*
     * TODO this comment about the 'edit' mode needs to be written
     */
    return html`
<div class="frame-no-grid">
  <div style="float: right">
    <div class="one-grid">
      <div class="one-grid-one" @click=${() => this.onCancelButton()}><img class="edit-action-icon" src="/edge_stack/images/cancel.png"/></div>
      <div class="one-grid-one" @click=${() => this.onSaveButton()}><img class="edit-action-icon" src="/edge_stack/images/save.png"/></div>
      <div class="one-grid-one" @click=${() => this.onDeleteButton()}><img class="edit-action-icon" src="/edge_stack/images/delete.png"/></div>
    </div>
  </div>
  <div class="three-grid">
    <div class="three-grid-all">
      <span>${this.resource.metadata.name}</span>
      <span class="crd-namespace">(${this.resource.metadata.namespace})</span>
    </div>
    <div class="three-grid-one edit-field-label">prefix url:</div>
    <div class="three-grid-two"><input type="text" name="prefix" value="${this.resource.spec.prefix}" /></div>
    <div class="three-grid-three"></div>
    <div class="three-grid-one edit-field-label">service:</div>
    <div class="three-grid-two"><input type="text" name="prefix" value="${this.resource.spec.service}" /></div>
    <div class="three-grid-three"></div>
  </div>
</div>
`
  }
  render_add_mode() {
    /*
     * TODO the 'add' mode is not completed
     */
    return html`NOT YET IMPLEMENTED ` // TODO
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
   * The onStartEdit function is triggered by clicking on a read-only displayed
   * version. It switches the mode of the component and then triggers a re-render.
   */
  onStartEdit() {
    this.requestUpdate();
    if (this.state.mode !== "edit") {
      this.state.mode = "edit"
    }
  }
  /*
   * TODO this comment about onCancelButton needs to be written
   */
  onCancelButton() {
    this.requestUpdate();
    if( this.state.mode === "edit") {
      this.state.mode = "list";
    } else {
      this.state.mode = "off";
    }
  }
  /*
   * TODO this comment about onSaveButton needs to be written
   */
  onSaveButton() {
    alert('Save!'); // TODO save is not yet re-implemented
  }
  /*
   * TODO this comment about onDeleteButton needs to be written
   */
  onDeleteButton() {
    alert('Delete!'); // TODO delete is not yet re-implemented
  }



}


customElements.define('dw-mapping', Mapping);

export class Mappings extends ResourceSet {

  getResources(snapshot) {
    return snapshot.getResources('Mapping')
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
    };
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

customElements.define('dw-mappings', Mappings);
