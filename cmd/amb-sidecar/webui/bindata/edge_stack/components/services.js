import { LitElement, html, css } from '/edge_stack/vendor/lit-element.min.js'
import {SingleResource} from '/edge_stack/components/resources.js'
import {Snapshot} from '/edge_stack/components/snapshot.js'

export class Service extends SingleResource {
  // implement
  kind() {
    return this.resource.kind;
  }

  // implement
  spec() {
    return this.resource.spec;
  }

  // internal
  irData() {
    const qname = this.resource.metadata.name + "." + this.resource.metadata.namespace;
    return this.diag.ambassador_services.find(s => s._source in this.diag.source_map[qname]);
  }

  // implement
  renderResource() {
    let str = `
     <div class="attribute-name">service url:</div>
     <div class="attribute-value">${this.irData().name}</div>

     <div class="attribute-name">weight:</div>
     <div class="attribute-value">${this.irData()._service_weight}</div>
`;
    let spec = (this.spec()||{});
    for (let key in spec) {
      if (spec.hasOwnProperty(key)) {
        str += `<div class="attribute-name">${key}:</div>
        <div class="attribute-value">${spec[key]}</div>`;
      }
    }
    return html([str]);
  }

  // override
  static get properties() {
    return {
      resource: {type: Map},
      diag: {type: Object},
    }
  }

  // override; this tab is read-only
  readOnly() {
    return true;
  }
}

customElements.define('dw-service', Service);

export class Services extends LitElement {

  // external ////////////////////////////////////////////////////////

  static get properties() {
    return {
      services: {type: Array},
      diag: {type: Object},
    };
  }

  constructor() {
    super();
    Snapshot.subscribe(this.onSnapshotChange.bind(this))
  }

  render() {
    return html`<div>
      ${this.services.map(s => {
        return html`<dw-service .resource=${s} .diag=${this.diag}></dw-service>`;
      })}
    </div>`;
  }

  // internal ////////////////////////////////////////////////////////

  onSnapshotChange(snapshot) {
    let kinds = ['AuthService', 'RateLimitService', 'TracingService', 'LogService']
    this.services = []
    kinds.forEach((k)=>{
      this.services.push(...snapshot.getResources(k))
    })
    this.diag = snapshot.getDiagnostics();
  }

}

customElements.define('dw-services', Services);
