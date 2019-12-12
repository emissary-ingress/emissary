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
    `;
  }

  render() {
    return html`
<link rel="stylesheet" href="../styles/resources.css">
<div class="header_con">
  <div class="col">
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 25.06 23.27"><defs><style>.cls-1{fill:#fff;}</style></defs><title>mappings</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><path class="cls-1" d="M25,19.86l-.81-1.62V14.32a.45.45,0,0,0-.45-.45H21.48V12.08a.45.45,0,0,0-.45-.45H14.25A1.8,1.8,0,0,0,13,10.35V8c1.84-.08,3.57-.69,3.57-1.78V4.47a.39.39,0,0,0,0-.18.75.75,0,0,0,0-.26V2.24a.41.41,0,0,0,0-.19.75.75,0,0,0,0-.26c0-1.18-2-1.79-4-1.79s-4,.61-4,1.79a1.09,1.09,0,0,0,0,.26.58.58,0,0,0,0,.19V4a1.09,1.09,0,0,0,0,.26.58.58,0,0,0,0,.18V6.26C8.5,7.35,10.24,8,12.08,8v2.31a1.8,1.8,0,0,0-1.28,1.28H4a.45.45,0,0,0-.45.45v1.79H1.34a.45.45,0,0,0-.45.45v3.92L.05,19.93a.46.46,0,0,0,0,.44.44.44,0,0,0,.38.21H7.61a.45.45,0,0,0,.45-.45A.44.44,0,0,0,8,19.86l-.81-1.62V14.32a.45.45,0,0,0-.45-.45H4.47V12.53H10.8a1.8,1.8,0,0,0,1.28,1.28v2.3H9.4a.45.45,0,0,0-.45.44v4.37l-.85,1.7a.45.45,0,0,0,.4.65h8.06a.45.45,0,0,0,.45-.45.42.42,0,0,0-.09-.27l-.81-1.63V16.55a.45.45,0,0,0-.45-.44H13v-2.3a1.8,1.8,0,0,0,1.27-1.28h6.33v1.34H18.34a.45.45,0,0,0-.44.45v3.92l-.85,1.69a.46.46,0,0,0,0,.44.44.44,0,0,0,.38.21h7.16a.45.45,0,0,0,.45-.45A.44.44,0,0,0,25,19.86Zm-23.8-.17.45-.9H6.43l.45.9ZM6.26,17.9H1.79V14.76H6.26Zm3,4.47.45-.89h5.71l.45.89Zm6-1.79H9.84V17h5.37ZM9.4,3a7,7,0,0,0,3.13.63A7,7,0,0,0,15.66,3V4c0,.26-1.1.89-3.13.89S9.4,4.29,9.4,4ZM12.53.89c2,0,3.13.64,3.13.9s-1.1.89-3.13.89S9.4,2.05,9.4,1.79,10.49.89,12.53.89ZM9.4,6.26V5.19a7.09,7.09,0,0,0,3.13.63,7.15,7.15,0,0,0,3.13-.63V6.26c0,.26-1.1.9-3.13.9S9.4,6.52,9.4,6.26ZM12.53,13a.9.9,0,1,1,0-1.79.9.9,0,0,1,0,1.79Zm6.26,1.78h4.48V17.9H18.79Zm-.62,4.93.45-.9h4.82l.44.9Z"/></g></g></svg>
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
                <span style="color: ${c._hcolor}">${c.type_label ? html`${type_label}:` : html``}${c.service}</span><br/>
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
