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

    .header_con .col img {
      width: 100%;
      height: 60px
    }
    
    .header_con .col img path {
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
    
    .logo {
      filter: invert(19%) sepia(64%) saturate(4904%) hue-rotate(248deg) brightness(107%) contrast(101%);
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
    <img alt="plugins logo" class="logo" src="../images/svgs/plugins.svg">
      <defs><style>.cls-1{fill:#fff;}</style></defs>
        <g id="Layer_2" data-name="Layer 2">
          <g id="Layer_1-2" data-name="Layer 1"></g>
        </g>
    </img>
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
