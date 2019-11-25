import {html} from '/edge_stack/vendor/lit-element.min.js'
import {SingleResource, ResourceSet} from '/edge_stack/components/resources.js';

class Mapping extends SingleResource {
  //TODO When Bjorn did a “Save” of a new mapping, he got errors in console about parse errors in the JSON returned from snapshot.
  /**
   * Implement.
   */
  kind() {
    return "Mapping"
  }

  /**
   * Override: in addition to the metadata.name and metadata.namespace attributes supplied
   * by my parent class (parent = Resource) the attributes of a Mapping are: prefix and target.
   */
  reset() {
    super.reset();
    this.prefixInput().value = this.prefixInput().defaultValue;
    this.targetInput().value = this.targetInput().defaultValue;
  }

  prefixInput() {
    return this.shadowRoot.querySelector('input[name="prefix"]')
  }

  targetInput() {
    return this.shadowRoot.querySelector('input[name="target"]')
  }

  /**
   * Implement.
   */
  spec() {
    return {
      prefix: this.prefixInput().value,
      service: this.targetInput().value
    }
  }

  /**
   * Implement.
   */
  renderResource() {
    return html`
    <div class="attribute-name">prefix url:</div>
    <div class="attribute-value"><visible-modes list><code>${this.resource.spec.prefix}</code></visible-modes>
      <visible-modes add edit><input type="text" name="prefix" value="${this.resource.spec.prefix}" /></visible-modes>
      </div>
    <div class="attribute-name">service:</div>
    <div class="attribute-value"><visible-modes list>${this.resource.spec.service}</visible-modes>
      <visible-modes add edit><input type="text" name="target" value="${this.resource.spec.service}" /></visible-modes>
      </div>

`
  }
  /**
   * Override.
   */
  minimumNumberOfEditRows() {
    return 2;
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
