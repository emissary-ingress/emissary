import {html} from '/edge_stack/vendor/lit-element.min.js'
import {SingleResource, ResourceSet} from '/edge_stack/components/resources.js';
import {getCookie} from '/edge_stack/components/cookies.js';

/**
 * A YAMLItem is the UI-side object for a "getambassador.io/v2" resource of any kind.
 */
export class YAMLItem extends SingleResource {
  constructor() {
    super();
  }

  render() {
  let resource = this.resource;

  return html`Resource YAML goes here...`;

  return html`
    <div class="attribute-name">Resource:</div>
    <div class="attribute-value">
      <span class="${this.visible("list")}">${resource.name}</span>
    </div>
    
    <div class="attribute-name">YAML:</div>
    <div class="attribute-value">
    <span class="${this.visible("list")}">${resource._yaml}</span>
    </div>`
  }
}

customElements.define('dw-yaml-item', YAMLItem);

export class YAMLDownloads extends ResourceSet {
  getResources(snapshot) {
    return snapshot.GetChangedResources()
  }

  render() {

    return html`
      <div>
        ${this.resources.map(h => html`<dw-yaml-item .resource=${h}}></dw-yaml-item>`)}
      </div>`
  }

}

customElements.define('dw-yaml-dl', YAMLDownloads);
