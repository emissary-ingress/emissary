import { LitElement, html, css } from '../vendor/lit-element.min.js'
import { Snapshot } from './snapshot.js'

export class RouteTable extends LitElement {
  // external ////////////////////////////////////////////////////////

  static get properties() {
    return {
      diagd: {type: Object}
    };
  }

  constructor() {
    super();

    Snapshot.subscribe(this.onSnapshotChange.bind(this));
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
      
      .card {
        background: #fff;
        border-radius: 10px;
        padding: 10px 30px 10px 30px;
        box-shadow: 0 10px 5px -11px rgba(0, 0, 0, .6);
        width: 100%;
        -webkit-flex-direction: row;
        -ms-flex-direction: row;
        flex-direction: row;
        -webkit-flex: 1 1 1;
        -ms-flex: 1 1 1;
        flex: 1 1 1;
        margin: 30px 0 0;
        font-size: .9rem;
      }
      
      .card, .card .col .con {
        display: -webkit-flex;
        display: -ms-flexbox;
        display: flex
      }

      .card .col .con {
        margin: 10px 0;
        -webkit-flex: 1;
        -ms-flex: 1;
        flex: 1;
        -webkit-justify-content: flex-end;
        -ms-flex-pack: end;
        justify-content: flex-end;
        height: 30px
      }
      
      .card .col, .card .col .con label, .card .col2, .col2 a.cta .label {
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

      table {
          margin: auto;
          width: 100%;
      }

      tr {
        height: 1.8em;
      }
      td {
        vertical-align: middle;
      }
      tr:nth-child(even) {
          background: #eeeeee;
      }

      thead {
          font-weight: bold;
      }

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
    `;
  }

  render() {
    return html`
<div class="header_con">
  <div class="col">
    <img alt="routeTable logo" class="logo" src="../images/svgs/routetable.svg"></img>
    <defs><style>.cls-1{fill:#fff;}</style></defs>
      <title>mappings</title>
        <g id="Layer_2" data-name="Layer 2">
          <g id="Layer_1-2" data-name="Layer 1"></g>
        </g>
  </div>
  <div class="col">
    <h1>Route Table</h1>
    <p>The active Envoy route table.</p>
  </div>
</div>

<div class="card">
    <table>
      <thead>
        <td>URL</td>
        <td>Service</td>
        <td>Weight</td>
      </thead>
      <tbody>
  ${this.diagd.route_info.map(r => {
      return html`
        <tr style="${r.diag_class == "private" ? "font-style: oblique; opacity: 60%;" : ""}">
          <td>
            <code>
              ${r.diag_class == "private" ? html`[internal route]<br/>` : html``}
              ${r.key}
              ${r.headers.map(h => html`<br/>${h.name}: ${h.value}`)}
              ${r.precedence != 0 ? html`<br/>precedence ${r.precedence}` : html``}
            </code>
          </td>
          <td>
            <code>
              ${r.clusters.map(c => html`
                <span style="color: ${c._hcolor}">${c.type_label ? html`${c.type_label}:` : html``}${c.service}</span><br/>
              `)}
            </code>
          </td>
          <td>
            ${r.clusters.map(c => html`
              ${c.weight.toLocaleString('en-US', {minimumFractionDigits: 2, maximumFractionDigits: 2, useGrouping:false})}%<br/>
            `)}
          </td>
        </tr>
      </tbody>
      `
    })}
</table>
</div>`;
  }

  // internal ////////////////////////////////////////////////////////

  onSnapshotChange(snapshot) {
    let diagnostics = snapshot.getDiagnostics();
    this.diagd = (('system' in (diagnostics||{})) ? diagnostics :
     {
       route_info: []
     });
  }
}

customElements.define('dw-routetable', RouteTable);
