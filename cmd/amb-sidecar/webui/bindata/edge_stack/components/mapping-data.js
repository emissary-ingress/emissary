import  {LitElement, html} from 'https://cdn.pika.dev/-/lit-element/2.2.1/dist-es2019/lit-element.min.js';
import {useContext} from '/edge_stack/components/context.js';

export default class MappingData extends LitElement {
  static get properties() {
    return {
      data: Object,
      loading: Boolean
    };
  }

  constructor() {
    super();

    const arr = useContext('aes-api-snapshot', null);

    this.setContext = arr[1];
    this.data = {};
    this.loading = false;
  }

  fetchData() {
    fetch('/edge_stack/api/snapshot')
      .then((data) => data.json())
      .then((json) => {
        this.setContext(json);
        this.fetchData();
      })
      .catch((err) => { console.log('error fetching snapshot', err); })
  }

  onMount() {
    this.loading = true;

    this.fetchData();
  }
  onUnmount() {}

  render() {
    if (this.loading) {
      return html`
      Loading...
      `;
    } else {
      return html`<slot></slot>`;
    }
  }
}

customElements.define('aes-snapshot-provider', MappingData);