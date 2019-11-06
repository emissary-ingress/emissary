import  {LitElement, html} from 'https://cdn.pika.dev/-/lit-element/2.2.1/dist-es2019/lit-element.min.js';

export class LoginGate extends LitElement {
  static get properties() {
    return {
      authToken: String,
      authenticated: Boolean,
      loading: Boolean,
      hasError: Boolean
    };
  }

  constructor() {
    super();

    this.authToken = window.location.hash.slice(1);
    this.authenticated = false;
    this.hasError = false;
    this.loading = true;

    this.loadData();
  }

  loadData() {
    fetch('api/empty', {
      headers: {
        'Authorization': 'Bearer ' + this.authToken
      }
    }).then((data) => {
      this.authenticated = (data.status == 200);
      this.loading = false;
      this.hasError = false;
    }).catch((err) => {
      console.log(err);

      this.authenticated = false;
      this.loading = false;
      this.hasError = true;
    });
  }

  renderError() {
    return html`
<p>Error check the console.</p>
    `;
  }

  renderLoading() {
    return html`
<p>Loading...</p>
    `;
  }

  renderUnauthenticated() {
    return html`
	<p>TODO: They get a page explaining that in order to
	authenticate they need to download edgectl and run
	<code>edgectl login</code>. The page includes a link they can
	click on to download edgectl.</p>
    `;
  }

  render() {
    if (this.hasError) {
      return this.renderError();
    } else if (this.loading) {
      return this.renderLoading();
    } else if (!this.authenticated) {
      return this.renderUnauthenticated();
    } else {
      return html`
<slot></slot>
      `;
    }
  }
}

customElements.define('login-gate', LoginGate);
