import { LitElement, html, css } from 'https://cdn.pika.dev/-/lit-element/2.2.1/dist-es2019/lit-element.min.js'
import {SingleResource} from '/edge_stack/components/resources.js'
import {registerContextChangeHandler, useContext} from '/edge_stack/components/context.js'

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

  // override; don't show any of the "edit/delete/whatever" buttons;
  // this tab is read-only.
  visible() {
    return [...arguments].includes("list") ? "" : "off";
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

    const [currentSnapshot, setSnapshot] = useContext('aes-api-snapshot', null);
    this.onSnapshotChange(currentSnapshot);
    this.onDiagChange({});
    registerContextChangeHandler('aes-api-snapshot', this.onSnapshotChange.bind(this));
    registerContextChangeHandler('aes-api-diag', this.onDiagChange.bind(this))
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
    this.services = [
      (((snapshot || {}).Kubernetes || {}).AuthService || []),
      (((snapshot || {}).Kubernetes || {}).RateLimitService || []),
      (((snapshot || {}).Kubernetes || {}).TracingService || []),
      (((snapshot || {}).Kubernetes || {}).LogService || []),
    ].reduce((acc, item) => acc.concat(item));
  }

  onDiagChange(snapshot) {
    this.diag = snapshot;
  }
}
customElements.define('dw-services', Services);
