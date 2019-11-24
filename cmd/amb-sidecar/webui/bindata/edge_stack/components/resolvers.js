import { LitElement, html, css } from 'https://cdn.pika.dev/-/lit-element/2.2.1/dist-es2019/lit-element.min.js'
import {SingleResource} from '/edge_stack/components/resources.js'
import {registerContextChangeHandler, useContext} from '/edge_stack/components/context.js'

export class Resolver extends SingleResource {
  // implement
  kind() {
    return this.resource_diag.kind;
  }

  // implement
  spec() {
    return (this.resource_watt || {}).spec;
  }

  // implement
  renderResource() {
    return html`
     <div class="left">Spec:</div>
     <div class="right"><pre>${JSON.stringify((this.resource_watt||{}).spec, null, 4)}</pre></div>

     <!--
     <div class="left">diag:</div>
     <div class="right"><pre>${JSON.stringify(this.resource_diag, null, 4)}</pre></div>
     -->
     `;
  }

  // override
  name() {
    if (this.resource_watt &&
        this.resource_watt.metadata &&
        this.resource_watt.metadata.name &&
        this.resource_watt.metadata.namespace) {
      return this.resource_watt.metadata.name
    } else if (this.qname.includes(".")) {
      return this.qname.replace(/\.[^.]*$/, '');
    } else {
      return this.qname;
    }
  }

  // override
  namespace() {
    if (this.resource_watt &&
        this.resource_watt.metadata &&
        this.resource_watt.metadata.name &&
        this.resource_watt.metadata.namespace) {
      return this.resource_watt.metadata.name
    } else if (this.qname.includes(".")) {
      return this.qname.replace(/.*\./, '');
    } else {
      return '';
    }
  }

  // override
  static get properties() {
    return {
      qname: {type: String},
      resource_watt: {type: Object},
      resource_diag: {type: Object},
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
customElements.define('dw-resolver', Resolver);

export class Resolvers extends LitElement {
  // external ////////////////////////////////////////////////////////

  static get properties() {
    return {
      watt: {type: Object},
      diag: {type: Object},
    };
  }

  constructor() {
    super();

    const [currentWatt, setWatt] = useContext('aes-api-snapshot', null);
    const [currentDiag, setDiag] = useContext('aes-api-diag', null);
    this.onWattChange(currentWatt);
    this.onDiagChange(currentDiag);
    registerContextChangeHandler('aes-api-snapshot', this.onWattChange.bind(this));
    registerContextChangeHandler('aes-api-diag', this.onDiagChange.bind(this));
  }

  static get styles() {
    return css``;
  }

  render() {
    return html`<div>
      ${(this.diag.ambassador_resolvers||[]).map(diag_resolver => {
        return html`<dw-resolver
          .qname=${this.originalSource(diag_resolver._source)}
          .resource_watt=${this.findResource(diag_resolver.kind, this.originalSource(diag_resolver._source))}
          .resource_diag=${diag_resolver}
        ></dw-resolver>`;
      })}
    </div>`;
  }

  // internal ////////////////////////////////////////////////////////

  findResource(kind, qname) {
    return ((this.watt.Kubernetes || {} )[kind] || []).find(object => {
      return object &&
        object.metadata &&
        object.metadata.name &&
        object.metadata.namespace &&
        (object.metadata.name+'.'+object.metadata.namespace) === qname;
    });
  }

  originalSource(source_name) {
    while (true) {
      let next = Object.entries(this.diag.source_map || {})
          .filter(([source, products]) => source !== source_name && source_name in products)
          .map(([source, products]) => source);
      switch (next.length) {
      case 0:
        return source_name;
      case 1:
        source_name = next[0];
        break;
      default:
        console.log("Go yell at Luke that he misunderstood how source_map works");
        source_name = next[0];
      }
    }
  }

  onWattChange(snapshot) {
    this.watt = snapshot || {};
  }

  onDiagChange(snapshot) {
    this.diag = snapshot || {};
  }
}
customElements.define('dw-resolvers', Resolvers);
