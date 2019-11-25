import { LitElement, html, css } from 'https://cdn.pika.dev/-/lit-element/2.2.1/dist-es2019/lit-element.min.js'
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
    return html`
     <div class="left">Service URL:</div>
     <div class="right">${this.irData().name}</div>

     <div class="left">Weight:</div>
     <div class="right">${this.irData()._service_weight}</div>

     <div class="left">Spec:</div>
     <div class="right"><pre>${JSON.stringify(this.spec(), null, 4)}</pre></div>`;
  }

  // override
  static get properties() {
    return {
      resource: {type: Map},
      diag: {type: Object},
    }
  }

  // override; don't show any of the "edit/delete/whatever" buttons;
  // this tab is read-only.
  static get styles() {
    return css`${super.styles} button { display: none; }`;
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

  static get styles() {
    return css``;
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
