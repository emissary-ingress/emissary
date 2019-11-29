import { LitElement, html, css } from '../vendor/lit-element.min.js'
import {SingleResource} from '../components/resources.js'
import {Snapshot} from '../components/snapshot.js'

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
    let str = '';
    let spec = (this.resource_watt||{}).spec;
    for (let key in spec) {
      if (spec.hasOwnProperty(key)) {
        str += `<div class="attribute-name">${key}:</div>
        <div class="attribute-value">${spec[key]}</div>`;
      }
    }
    return html([str]);
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

  // override; this tab is read-only
  readOnly() {
    return true;
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
    Snapshot.subscribe(this.onSnapshotChange.bind(this))
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

  onSnapshotChange(snapshot) {
    this.watt = snapshot.data.Watt
    this.diag = snapshot.getDiagnostics()
  }
}
customElements.define('dw-resolvers', Resolvers);
