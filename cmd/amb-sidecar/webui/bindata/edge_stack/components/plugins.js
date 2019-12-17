import { LitElement, html, css } from '../vendor/lit-element.min.js'
import {SingleResource} from './resources.js'
import {Snapshot} from './snapshot.js'

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
<div class="row line">
  <div class="row-col margin-right justify-right">service url:</div>
  <div class="row-col">${this.irData().name}</div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right">weight:</div>
  <div class="row-col">${this.irData()._service_weight}</div>
</div>
`;
    let spec = (this.spec()||{});
    for (let key in spec) {
      if (spec.hasOwnProperty(key)) {
        str += `<div class="row line">
  <div class="row-col margin-right justify-right">${key}:</div>
  <div class="row-col">${(typeof spec[key] === 'string') ? spec[key] : JSON.stringify(spec[key])}</div>
</div>`;
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

  static get styles() {
    return css`
    * {
      margin: 0;
      padding: 0;
      border: 0;
      position: relative;
      box-sizing: border-box
    }
    
    *, textarea {
      vertical-align: top
    }
    
    
    .header_con, .header_con .col {
      display: -webkit-flex;
      display: -ms-flexbox;
      display: flex;
      -webkit-justify-content: center;
      -ms-flex-pack: center;
      justify-content: center
    }
    
    .header_con {
      margin: 30px 0 0;
      -webkit-flex-direction: row;
      -ms-flex-direction: row;
      flex-direction: row
    }
    
    .header_con .col {
      -webkit-flex: 0 0 80px;
      -ms-flex: 0 0 80px;
      flex: 0 0 80px;
      -webkit-align-content: center;
      -ms-flex-line-pack: center;
      align-content: center;
      -webkit-align-self: center;
      -ms-flex-item-align: center;
      align-self: center;
      -webkit-flex-direction: column;
      -ms-flex-direction: column;
      flex-direction: column
    }
    
    .header_con .col svg {
      width: 100%;
      height: 60px
    }
    
    .header_con .col svg path {
      fill: #5f3eff
    }
    
    .header_con .col:nth-child(2) {
      -webkit-flex: 2 0 auto;
      -ms-flex: 2 0 auto;
      flex: 2 0 auto;
      padding-left: 20px
    }
    
    .header_con .col h1 {
      padding: 0;
      margin: 0;
      font-weight: 400
    }
    
    .header_con .col p {
      margin: 0;
      padding: 0
    }
    
    .header_con .col2, .col2 a.cta .label {
      -webkit-align-self: center;
      -ms-flex-item-align: center;
      -ms-grid-row-align: center;
      align-self: center
    }
    
    .col2 a.cta  {
      text-decoration: none;
      border: 2px #efefef solid;
      border-radius: 10px;
      width: 90px;
      padding: 6px 8px;
      -webkit-flex: auto;
      -ms-flex: auto;
      flex: auto;
      margin: 10px auto;
      color: #000;
      transition: all .2s ease;
      cursor: pointer;
    }
    
    .header_con .col2 a.cta  {
      border-color: #c8c8c8;
    }
    
    .col2 a.cta .label {
      text-transform: uppercase;
      font-size: .8rem;
      font-weight: 600;
      line-height: 1rem;
      padding: 0 0 0 10px;
      -webkit-flex: 1 0 auto;
      -ms-flex: 1 0 auto;
      flex: 1 0 auto
    }
    
    .col2 a.cta svg {
      width: 15px;
      height: auto
    }
    
    .col2 a.cta svg path, .col2 a.cta svg polygon {
      transition: fill .7s ease;
      fill: #000
    }
    
    .col2 a.cta:hover {
      color: #5f3eff;
      transition: all .2s ease;
      border: 2px #5f3eff solid
    }
    
    .col2 a.cta:hover svg path, .col2 a.cta:hover svg polygon {
      transition: fill .2s ease;
      fill: #5f3eff
    }
    
    .col2 a.cta {
      display: -webkit-flex;
      display: -ms-flexbox;
      display: flex;
      -webkit-flex-direction: row;
      -ms-flex-direction: row;
      flex-direction: row
    }
    
    .col2 a.off {
      display: none;
    }    
  `
}

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
    return html`
<div class="header_con">
  <div class="col">
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24.99 21.04"><defs><style>.cls-1{fill:#fff;}</style></defs><title>services</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><path class="cls-1" d="M0,0V21H25V0ZM5.94,14.4l.94-.94,1.85,1.32L9,14.63a6.39,6.39,0,0,1,2.23-.92l.27-.06.37-2.24h1.34l.36,2.24.27.06a6.39,6.39,0,0,1,2.23.92l.23.15,1.85-1.33.94,1-1.32,1.85.15.23a6.39,6.39,0,0,1,.92,2.23l.06.27,2.24.36v.89H3.89v-.89L6.13,19l.06-.27a6.39,6.39,0,0,1,.92-2.23l.15-.23Zm18.24,5.83H21.91v-1a.7.7,0,0,0-.58-.69l-1.8-.29a7.12,7.12,0,0,0-.83-2l1.06-1.47a.7.7,0,0,0-.07-.9l-1.08-1.08a.7.7,0,0,0-.9-.07l-1.47,1.06a7.12,7.12,0,0,0-2-.83L14,11.18a.7.7,0,0,0-.68-.58H11.73a.69.69,0,0,0-.68.58l-.3,1.8a7.12,7.12,0,0,0-2,.83L7.27,12.74a.69.69,0,0,0-.89.07L5.29,13.9a.71.71,0,0,0-.07.89l1.07,1.48a7.12,7.12,0,0,0-.83,2l-1.79.29a.71.71,0,0,0-.59.69v1H.81V5.2H24.18ZM.81,4.4V.81H24.18V4.4Z"/><rect class="cls-1" x="2.15" y="2.2" width="1.33" height="0.81"/><rect class="cls-1" x="4.01" y="2.2" width="1.33" height="0.81"/><rect class="cls-1" x="5.88" y="2.2" width="1.33" height="0.81"/><path class="cls-1" d="M12.5,17a3.23,3.23,0,0,1,3.22,3.23h.81a4,4,0,0,0-8.07,0h.81A3.23,3.23,0,0,1,12.5,17Z"/></g></g></svg>
  </div>
  <div class="col">
    <h1>Plugins</h1>
    <p>Special plugin services that enhance the functionality of Ambassador Edge Stack.<br/>
       These plugin services are called when Ambassador handles requests.</p>
  </div>
</div>
<div>
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
