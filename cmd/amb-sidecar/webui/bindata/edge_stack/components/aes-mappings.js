import  {LitElement, html} from 'https://cdn.pika.dev/-/lit-element/2.2.1/dist-es2019/lit-element.min.js';
import { repeat } from '/edge_stack/components/repeat.js';
import {registerContextChangeHandler, useContext} from '/edge_stack/components/context.js';

export default class AesMappings extends LitElement {
  static get properties() {
    return {
      isShowingAdd: Boolean,
      mappings: Array
    };
  }

  constructor() {
    super();

    const arr = useContext('aes-api-snapshot', null);
    this.isShowingAdd = false;
    if (arr[0] != null) {
      this.mappings = arr[0]['Mapping'] || [];
    } else {
      this.mappings = [];
    }
    registerContextChangeHandler('aes-api-snapshot', this.onSnapshotChange.bind(this));
  }

  onSnapshotChange(snapshot) {
    this.mappings = snapshot['Mapping'] || [];
  }

  onAddClick() {
    this.isShowingAdd = true;
  }

  addMapping(evt) {
  }

  getAddSnippet() {
    return html`
<form>
  <label>Name: <input id="name" type="text" name="prefix" /></label>
  <label>Prefix: <input id="prefix" type="text" name="prefix" /></label>
  <label>Rewrite: <input id="rewrite" type="text" name="rewrite" value="" /></label>
  <label>Target: <input id="target" type="text" name="target" /></label>
  <button @click=${this.addMapping.bind(this)}>Add</button>
</form>
    `;
  }

  render() {
    return html`
<h2>Mappings</h2>
<button class="link-button" @click=${this.onAddClick.bind(this)}>Add</button>
${this.isShowingAdd ? this.getAddSnippet() : html``}
<div id="mappings">
  ${repeat(this.mappings, (mapping) => mapping.namespace + '_' + mapping.name, (mapping, idx) => html`
    <div class="mapping ${mapping.kind.toLowerCase()}" >
      <p>${mapping.name}</p>
    </div>
  `)}
</div>
    `;
  }
}

customElements.define('aes-mappings', AesMappings);