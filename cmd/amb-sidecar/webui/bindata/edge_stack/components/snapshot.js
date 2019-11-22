import  {LitElement, html} from 'https://cdn.pika.dev/-/lit-element/2.2.1/dist-es2019/lit-element.min.js';
import {useContext} from '/edge_stack/components/context.js';
import {getCookie} from '/edge_stack/components/cookies.js';

function updateCredentials(value) {
  // Keep this in-sync with webui.go:registerActivity()
  //
  // - Don't set expires=/max-age=; leave it as a "session cookie", so
  //   that it will expire at the end of the "session" (when they
  //   close their browser).  We'll let time-based expiration be
  //   enforced by the `exp` JWT claim.
  //
  // - Don't set domain=; explicitly it to window.location.hostname
  //   would instead also match "*.${window.location.hostname".
  //
  // - Restrict it to the `/edge_stack/*` path.
  document.cookie = `edge_stack_auth=${value}; path=/edge_stack/`;
}

export default class Snapshot extends LitElement {
  static get properties() {
    return {
      data: Object,
      loading: Boolean,
      fragment: String,
    };
  }

  constructor() {
    super();

    this.setSnapshot = useContext('aes-api-snapshot', null)[1];
    this.setDiag = useContext('aes-api-diag', null)[1];
    this.setAuthenticated = useContext('auth-state', null)[1];
    this.loading = false;

    if (getCookie("edge_stack_auth")) {
      this.fragment = "should-try";
    } else {
      updateCredentials(window.location.hash.slice(1));
      this.fragment = "trying";
    }
  }

  fetchData() {
    fetch('/edge_stack/api/snapshot', {
      headers: {
        'Authorization': 'Bearer ' + getCookie("edge_stack_auth")
      }
    })
      .then((response) => {
        if (response.status == 400 || response.status == 401 || response.status == 403) {
          if (this.fragment === "should-try") {
            updateCredentials(window.location.hash.slice(1));
            this.fragment = "trying";
            setTimeout(this.fetchData.bind(this), 1);
          } else {
            this.fragment = "";
            this.setAuthenticated(false);
            this.setSnapshot({});
            this.setDiag({});
          }
        } else {
          response.json().then((json) => {
            if (this.fragment == "trying") {
              window.location.hash = "";
            }
            this.fragment = ""
            this.setSnapshot(json.Watt)
            this.setDiag(json.Diag || {})
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
