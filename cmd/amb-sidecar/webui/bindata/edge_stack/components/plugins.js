import { LitElement, html, css } from '../vendor/lit-element.min.js'
import {SingleResource} from './resources.js'
import {Snapshot} from './snapshot.js'
////MOREMORE do the new look for the plugins page

export class Plugin extends SingleResource {
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
    if (this.diag['ambassador_services'] == null) {
      return [];
    } else {
      return this.diag.ambassador_services.find(s => s._source in this.diag.source_map[qname]);
    }
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
        <div class="attribute-value">${(typeof spec[key] === 'string') ? spec[key] : JSON.stringify(spec[key])}</div>`;
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

customElements.define('dw-plugin', Plugin);

export class Plugins extends LitElement {

  // external ////////////////////////////////////////////////////////

  static get properties() {
    return {
      plugins: {type: Array},
      diag: {type: Object},
    };
  }

  // override; this tab is read-only
  readOnly() {
    return true;
  }

  constructor() {
    super();

    this.diag = {};
    this.plugins = [];

    Snapshot.subscribe(this.onSnapshotChange.bind(this))
  }

  render() {
    return html`<div>
      ${this.plugins.map(s => {
        return html`<dw-plugin .resource=${s} .diag=${this.diag}></dw-plugin>`;
      })}
    </div>`;
  }

  // internal ////////////////////////////////////////////////////////

  onSnapshotChange(snapshot) {
    let kinds = ['AuthService', 'RateLimitService', 'TracingService', 'LogService'];
    this.plugins = [];
    kinds.forEach((k)=>{
      this.plugins.push(...snapshot.getResources(k))
    });
    this.diag = snapshot.getDiagnostics();
  }

}

customElements.define('dw-plugins', Plugins);
