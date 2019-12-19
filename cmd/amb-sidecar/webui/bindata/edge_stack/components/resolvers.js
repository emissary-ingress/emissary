import { LitElement, html, css } from '../vendor/lit-element.min.js'
import {SingleResource} from './resources.js'
import {Snapshot} from './snapshot.js'

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
        str += `<div class="row line">
  <div class="row-col margin-right justify-right">${key}:</div>
  <div class="row-col">${spec[key]}</div>
</div>`;
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

  // override; this tab is read-only
  readOnly() {
    return true;
  }

  constructor() {
    super();
    Snapshot.subscribe(this.onSnapshotChange.bind(this))
  }

  static get styles() {
    return css``;
  }

  render() {
    return html`
<link rel="stylesheet" href="../styles/resources.css">
<div class="header_con">
  <div class="col">
    <img alt="resolvers logo" class="logo" src="../images/svgs/resolvers.svg">
      <defs><style>.cls-1{fill:#fff;}</style></defs>
        <g id="Layer_2" data-name="Layer 2">
          <g id="Layer_1-2" data-name="Layer 1"></g>
        </g>
    </img>
  </div>
  <div class="col">
    <h1>Resolvers</h1>
    <p>Resolvers in use.</p>
  </div>
</div>
<div>

<div>
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
