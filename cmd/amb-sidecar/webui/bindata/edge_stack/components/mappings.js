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

  validate() {
    super.validate();
    /*
     * Validate that the labels have no empty strings in them. The labels data structure
     * is complex, hence the multiple nested iterations here:
     *
     * Array of
     *   Object with many
     *     String X: Array of
     *          String X
     *       or Object with
     *             String X: Object with
     *                "header": String X
     * We need to check all the String Xs to be sure they are not empty strings.
     */
    if (this.state.labels && this.state.labels.length > 0) {
      for(var j = 0; j < this.state.labels.length; j++) { // Array of..
        var c = this.state.labels[j];  // Object..
        for(const n in c) {  // with many..
          if (n && n.length > 0) {  // String X: ..
            var a = c[n];
            for(var i = 0; i < a.length; i++) {  // ..: Array of..
              var s = a[i];
              if (typeof s === "string" && s.length === 0) {  // String X
                this.state.messages.push("Labels must not be empty names.");
                return;
              }
              if (typeof s === "object") {  // or Object..
                for(const k in s) {  // with..
                  if (k.length === 0) {  // String X: ..
                    this.state.messages.push("Labels must not be empty names in either box.");
                    return;
                  } else {
                    var b = s[k];
                    for(const m in b) {  // ..: Object with..
                      var d = b[m];
                      if (d.length === 0) { // String X
                        this.state.messages.push("Labels must not be empty names in either box");
                        return;
                      }
                    }
                  }
                }
              }
            }
          } else {
            this.state.messages.push("Labels must not have empty names.")
            return;
          }
        }
      }
    }
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
    <img alt="mappings logo" class="logo" src="../images/svgs/mappings.svg">
      <defs><style>.cls-1{fill:#fff;}</style></defs>
        <g id="Layer_2" data-name="Layer 2">
          <g id="Layer_1-2" data-name="Layer 1"></g>
        </g>
    </img>
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
