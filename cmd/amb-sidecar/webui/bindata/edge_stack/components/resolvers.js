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
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 25.82 25.82"><defs><style>.cls-1{fill:#fff;}</style></defs><title>resolve</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><path class="cls-1" d="M12.91,4.19a8.72,8.72,0,1,0,8.72,8.72A8.73,8.73,0,0,0,12.91,4.19ZM20,9.53H16.58a9.53,9.53,0,0,0-1.72-4.25A7.9,7.9,0,0,1,20,9.53Zm-4,3.38a19.91,19.91,0,0,1-.17,2.54H9.94a19.06,19.06,0,0,1,0-5.08h5.94A19.91,19.91,0,0,1,16.05,12.91ZM12.91,5c1.11,0,2.28,1.72,2.82,4.5H10.09C10.63,6.75,11.8,5,12.91,5ZM11,5.28A9.53,9.53,0,0,0,9.24,9.53H5.8A7.9,7.9,0,0,1,11,5.28ZM5,12.91a7.65,7.65,0,0,1,.43-2.54H9.1a19.06,19.06,0,0,0,0,5.08H5.46A7.65,7.65,0,0,1,5,12.91Zm.77,3.38H9.24A9.53,9.53,0,0,0,11,20.54,7.9,7.9,0,0,1,5.8,16.29Zm7.11,4.5c-1.11,0-2.28-1.72-2.82-4.5h5.64C15.19,19.07,14,20.79,12.91,20.79Zm1.95-.25a9.53,9.53,0,0,0,1.72-4.25H20A7.9,7.9,0,0,1,14.86,20.54Zm1.86-5.09a18.67,18.67,0,0,0,.17-2.54,18.67,18.67,0,0,0-.17-2.54h3.64a7.72,7.72,0,0,1,0,5.08Z"/><path class="cls-1" d="M16,24.47a12,12,0,0,1-11.57-20l.67-.66v.33a.47.47,0,1,0,.94,0V2.64a.44.44,0,0,0,0-.15l0-.06a.5.5,0,0,0-.08-.12h0a.38.38,0,0,0-.13-.09.23.23,0,0,0-.11,0l-.07,0H4.11a.47.47,0,0,0-.47.47.48.48,0,0,0,.47.48h.33l-.66.66A12.92,12.92,0,0,0,3.78,22a13,13,0,0,0,9.15,3.78,12.83,12.83,0,0,0,3.32-.44.46.46,0,0,0,.35-.45.45.45,0,0,0,0-.12A.49.49,0,0,0,16,24.47Z"/><path class="cls-1" d="M21.71,22.7h-.33L22,22A12.92,12.92,0,0,0,22,3.78,12.93,12.93,0,0,0,9.57.44a.46.46,0,0,0-.35.45.45.45,0,0,0,0,.12.48.48,0,0,0,.58.34,12,12,0,0,1,11.57,20l-.67.66v-.33a.47.47,0,0,0-.94,0v1.47a.49.49,0,0,0,0,.15l0,.06a.5.5,0,0,0,.08.12.36.36,0,0,0,.12.08l.06,0,.15,0h1.47a.47.47,0,0,0,.47-.47A.48.48,0,0,0,21.71,22.7Z"/></g></g></svg>
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
