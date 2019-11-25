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
    return css``;
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
      return html`<tr>
          <td>${r.key}</td>
          <td>${r.clusters.map(c => html`<span style="color: ${c._hcolor}">${c.service}</span><br/>`)}</td>
          <td>${r.clusters.map(c => html`${c.weight}%<br/>`)}</td>
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
