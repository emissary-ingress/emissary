import { LitElement, html, css } from 'https://cdn.pika.dev/-/lit-element/2.2.1/dist-es2019/lit-element.min.js'
import {Resource} from '/edge_stack/components/resources.js'
import {registerContextChangeHandler, useContext} from '/edge_stack/components/context.js'

export class Service extends Resource {
  // implement
  kind() {
    return this.resource.kind;
  }

  // implement
  spec() {
    return this.resource.spec;
  }

  // internal
  irURL() {
    if (this.kind() === 'AuthService') {
      return this.spec().auth_service;
    } else {
      return this.spec().service;
    }
  }

  // implement
  renderResource() {
    return html`
     <div class="left">Service URL:</div>
     <div class="right">${this.irURL()}</div>
     <div class="left">Spec:</div>
     <div class="right"><pre>${JSON.stringify(this.spec(), null, 4)}</pre></div>`;
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
    };
  }

  constructor() {
    super();

    const [currentSnapshot, setSnapshot] = useContext('aes-api-snapshot', null);
    this.onSnapshotChange(currentSnapshot)
    registerContextChangeHandler('aes-api-snapshot', this.onSnapshotChange.bind(this))
 }

  static get styles() {
    return css``;
  }

  render() {
    return html`<div>
      ${this.services.map(s => {
        return html`<dw-service .resource=${s}></dw-service>`;
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
}
customElements.define('dw-services', Services);
