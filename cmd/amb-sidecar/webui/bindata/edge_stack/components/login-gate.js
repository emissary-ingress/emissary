import  {LitElement, css, html} from 'https://cdn.pika.dev/-/lit-element/2.2.1/dist-es2019/lit-element.min.js';

export class LoginGate extends LitElement {
  static get properties() {
    return {
      authToken: String,
      authenticated: Boolean,
      loading: Boolean,
      hasError: Boolean
    };
  }

  static get styles() {
    return css`
* {
  font-family: Source Sans Pro,sans-serif;
  margin-left: 2em;
  margin-right: 2em;
}

#all-wrapper {
  width: 80%;
  margin-left: 10%;
}
#ambassador-logo {
  background-color: black;
  padding: 5px;
  width: 456px;
  height: 42px;
  margin-bottom: 1em
}
div.info-blob {
  border: 1px solid #ede7f3;
  box-shadow: 0 2px 4px rgba(0,0,0,.1);
  padding: 0.5em;
  margin-bottom: 0.6em;
  line-height: 1.3;
}
div.info-blob2 {
  border: thin solid grey;
  border-radius: 0.4em;
  padding: 0.5em;
  margin-bottom: 0.6em;
  line-height: 1.3;
}
div.info-title {
  font-weight: bold;
  font-size: 120%;
}
span.command {
  background-color: #f5f2f0;;
  padding: 3px;
  letter-spacing: .2px;
  font-family: Consolas,Monaco,Andale Mono,Ubuntu Mono,monospace;
  font-size: 90%;
  word-spacing: normal;
  word-break: normal;
  word-wrap: normal;
  hypens: none;
}
div.overage-alert {
  border: 3px solid red;
  border-radius: 0.7em;
  padding: 0.5em;
  background-color: #FFe8e8;
}
    `;
  }

  constructor() {
    super();

    this.authToken = window.location.hash.slice(1);
    this.authenticated = false;
    this.hasError = false;
    this.loading = true;

    this.isSlotOpen = false;
    this.slotChildren = [];

    this.loadData();
  }

  onSlotChange({ target }) {
    this.isSlotOpen = !this.isSlotOpen;

    if (this.isSlotOpen) {
      this.slotChildren = target.assignedNodes().filter(node => node.nodeName != '#text');
      this.slotChildren.forEach(node => {
        if (node.hasOwnProperty('onMount') && typeof node['onMount'] == 'function') {
          node.onMount();
        }
      });
    } else {
      let nodes = target.assignedNodes().filter(node => node.nodeName != '#text');
      let removed_nodes = [];
      this.slotChildren = this.slotChildren.filter(node => {
        if (nodes.indexOf(node) == -1) {
          removed_nodes.push(node);
          return false;
        }
        return true;
      });
      removed_nodes.forEach(removedNode => {
        if (removedNode.hasOwnProperty('onUnmount') && typeof removedNode['onUnmount'] == 'function') {
          node.onUnmount();
        }
      });
    }
  }

  loadData() {
    fetch('/edge_stack/tls/api/empty', {
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
<div id="all-wrapper">
  <div class="info-blob">
    <div class="info-title">Welcome to Ambassador Edge Stack!</div>
    <p>To login to the admin portal, use <span class="command">edgectl login ${window.location.hostname}</span>.</p>
    <p>
      If you do not yet have the edgectl executable, download it <a href="https://deploy-preview-91--datawire-ambassador.netlify.com/downloads/edgectl">from the getambassador.io website</a>.
    </p>
  </div>
</div>
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
<slot @slotchange=${this.onSlotChange}></slot>
      `;
    }
  }
}

customElements.define('login-gate', LoginGate);
