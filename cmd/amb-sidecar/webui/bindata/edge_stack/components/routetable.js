import { LitElement, html, css } from '/edge_stack/vendor/lit-element.min.js'
import { Snapshot } from '/edge_stack/components/snapshot.js'

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
table {
    margin: auto;
    width: 100%;
}

tr:nth-child(even) {
    background: #EEE;
}

thead {
    font-weight: bold;
}
    `;
  }

  render() {
    return html`
    <table>
      <thead>
        <td>URL</td>
        <td>Service</td>
        <td>Weight</td>
      </thead>
      <tbody>
  ${this.diagd.route_info.map(r => {
      return html`
        <tr style="display:${r.diag_class == "private" ? "font-style: oblique; opacity: 60%;" : ""}">
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
</table>`;
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
