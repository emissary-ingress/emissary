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
    let result = {
      prefix: this.prefixInput().value,
      service: this.targetInput().value
    };

    if (this.state.labels && this.state.labels.length > 0) {
      result["labels"] = {
        ambassador: this.state.labels
      };
    }

    return result;
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
    let source = this.sourceURI();

    return html`
<div class="row line">
  <label class="row-col margin-right justify-right">prefix url:</label>
  <div class="row-col">
    <span class="${this.visible("list")}">${this.resource.spec.prefix}</span>
    <input class="${this.visible("edit", "add")}" type="text" name="prefix"  value="${this.resource.spec.prefix}"/>
  </div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right">service:</div>
  <div class="row-col">
    <span class="${this.visible("list")}">${this.resource.spec.service}</span>
    <input class="${this.visible("edit", "add")}" type="text" name="target"  value="${this.resource.spec.service}"/>
  </div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right">labels:</div>
  <div class="row-col">
    <dw-request-labels .mode=${this.state.mode} .labels=${labels}
                         @change=${(e)=>{this.state.labels = e.target.labels}}>
    </dw-request-labels>
  </div>
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

  renderInner() {
    let shtml = super.renderInner();
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
<div class="header_con">
  <div class="col">
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 25.06 23.27"><defs><style>.cls-1{fill:#fff;}</style></defs><title>mappings</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><path class="cls-1" d="M25,19.86l-.81-1.62V14.32a.45.45,0,0,0-.45-.45H21.48V12.08a.45.45,0,0,0-.45-.45H14.25A1.8,1.8,0,0,0,13,10.35V8c1.84-.08,3.57-.69,3.57-1.78V4.47a.39.39,0,0,0,0-.18.75.75,0,0,0,0-.26V2.24a.41.41,0,0,0,0-.19.75.75,0,0,0,0-.26c0-1.18-2-1.79-4-1.79s-4,.61-4,1.79a1.09,1.09,0,0,0,0,.26.58.58,0,0,0,0,.19V4a1.09,1.09,0,0,0,0,.26.58.58,0,0,0,0,.18V6.26C8.5,7.35,10.24,8,12.08,8v2.31a1.8,1.8,0,0,0-1.28,1.28H4a.45.45,0,0,0-.45.45v1.79H1.34a.45.45,0,0,0-.45.45v3.92L.05,19.93a.46.46,0,0,0,0,.44.44.44,0,0,0,.38.21H7.61a.45.45,0,0,0,.45-.45A.44.44,0,0,0,8,19.86l-.81-1.62V14.32a.45.45,0,0,0-.45-.45H4.47V12.53H10.8a1.8,1.8,0,0,0,1.28,1.28v2.3H9.4a.45.45,0,0,0-.45.44v4.37l-.85,1.7a.45.45,0,0,0,.4.65h8.06a.45.45,0,0,0,.45-.45.42.42,0,0,0-.09-.27l-.81-1.63V16.55a.45.45,0,0,0-.45-.44H13v-2.3a1.8,1.8,0,0,0,1.27-1.28h6.33v1.34H18.34a.45.45,0,0,0-.44.45v3.92l-.85,1.69a.46.46,0,0,0,0,.44.44.44,0,0,0,.38.21h7.16a.45.45,0,0,0,.45-.45A.44.44,0,0,0,25,19.86Zm-23.8-.17.45-.9H6.43l.45.9ZM6.26,17.9H1.79V14.76H6.26Zm3,4.47.45-.89h5.71l.45.89Zm6-1.79H9.84V17h5.37ZM9.4,3a7,7,0,0,0,3.13.63A7,7,0,0,0,15.66,3V4c0,.26-1.1.89-3.13.89S9.4,4.29,9.4,4ZM12.53.89c2,0,3.13.64,3.13.9s-1.1.89-3.13.89S9.4,2.05,9.4,1.79,10.49.89,12.53.89ZM9.4,6.26V5.19a7.09,7.09,0,0,0,3.13.63,7.15,7.15,0,0,0,3.13-.63V6.26c0,.26-1.1.9-3.13.9S9.4,6.52,9.4,6.26ZM12.53,13a.9.9,0,1,1,0-1.79.9.9,0,0,1,0,1.79Zm6.26,1.78h4.48V17.9H18.79Zm-.62,4.93.45-.9h4.82l.44.9Z"/></g></g></svg>
  </div>
  <div class="col">
    <h1>Mappings</h1>
    <p>Associations between prefix URLs and target services.</p>
  </div>
  <div class="col2">
    <a class="cta add ${this.readOnly() ? "off" : ""}" @click=${()=>this.shadowRoot.getElementById("add-mapping").onAdd()}>
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
<dw-mapping id="add-mapping" .resource=${newMapping} .state=${this.addState}>
  <add-button></add-button>
</dw-mapping>
${shtml}
`;
  }
  renderSet() {
    return html`
<div>
  ${this.resources.map(r => {
    return html`<dw-mapping .resource=${r} .state=${this.state(r)}></dw-mapping>`
  })}
</div>`
  }

}

customElements.define('dw-mappings', Mappings);
