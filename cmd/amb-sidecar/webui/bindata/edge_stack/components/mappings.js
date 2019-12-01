import {html} from '../vendor/lit-element.min.js'
import {SingleResource, SortableResourceSet} from './resources.js';
import './request-labels.js';

class Mapping extends SingleResource {

  /**
   * Implement.
   */
  init() {
    this.state.labels = null;
  }

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
    this.state.labels = null;
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
      service: this.targetInput().value,
      labels: {
        ambassador: this.state.labels
      }
    }
  }

  /**
   * Override.
   */
  onEdit() {
    super.onEdit()
    this.state.labels = this.labels()
  }

  // internal
  labels() {
    return (this.resource.spec.labels || {}).ambassador || [];
  }

  /**
   * Implement.
   */
  renderResource() {
    let labels = this.state.mode === "edit" ? this.state.labels : this.labels();
    return html`
    <div class="attribute-name">prefix url:</div>
    <div class="attribute-value"><visible-modes list><code>${this.resource.spec.prefix}</code></visible-modes>
      <visible-modes add edit><input type="text" name="prefix" value="${this.resource.spec.prefix}" /></visible-modes>
      </div>
    <div class="attribute-name">service:</div>
    <div class="attribute-value"><visible-modes list>${this.resource.spec.service}</visible-modes>
      <visible-modes add edit><input type="text" name="target" value="${this.resource.spec.service}" /></visible-modes>
    </div>
    <div class="attribute-name">labels:</div>
    <div class="attribute-value">
      <dw-request-labels .mode=${this.state.mode} .labels=${labels}
                         @change=${(e)=>{this.state.labels = e.target.labels}}>
      </dw-request-labels>
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

export class Mappings extends SortableResourceSet {

  constructor() {
    super([
      {value: "name", label: "Name"},
      {value: "namespace", label: "Namespace"},
      {value: "prefix", label: "Prefix"}
    ]);
  }

  getResources(snapshot) {
    return snapshot.getResources('Mapping')
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
