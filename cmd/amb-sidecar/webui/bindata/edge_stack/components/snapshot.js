import  {LitElement, html} from 'https://cdn.pika.dev/-/lit-element/2.2.1/dist-es2019/lit-element.min.js';
import {useContext} from '/edge_stack/components/context.js';

export default class Snapshot extends LitElement {
  static get properties() {
    return {
      data: Object,
      loading: Boolean
    };
  }

  constructor() {
    super();

    this.setSnapshot = useContext('aes-api-snapshot', null)[1];
    this.setAuthenticated = useContext('auth-state', null)[1];
    this.loading = false;
  }

  fetchData() {
    fetch('/edge_stack/api/snapshot', {
      headers: {
        'Authorization': 'Bearer ' + window.location.hash.slice(1)
      }
    })
      .then((response) => {
        if (response.status == 401 || response.status == 403) {
          this.setAuthenticated(false)
          this.setSnapshot({})
        }  else {
          response.json().then((json) => {
            this.setSnapshot(json)
            this.setAuthenticated(true)
            this.loading = false;
            setTimeout(this.fetchData.bind(this), 1000);
          })
        }
      })
      .catch((err) => { console.log('error fetching snapshot', err); })
  }

  firstUpdated() {
    this.loading = true;
    this.fetchData();
  }

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

customElements.define('aes-snapshot-provider', Snapshot);
