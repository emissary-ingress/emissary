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
    
    .header_con .col img {
      width: 100%;
      height: 60px;
    }
    
    .header_con .col img path {
      fill: #5f3eff
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
    
    .logo {
      filter: invert(19%) sepia(64%) saturate(4904%) hue-rotate(248deg) brightness(107%) contrast(101%);
    }
    
    .col2 a.cta  {
      text-decoration: none;
      border: 2px #efefef solid;
      border-radius: 10px;
      width: 90px;
      padding: 6px 8px;
      max-height: 35px;
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
    `;
  }

  render() {
    return html`
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
