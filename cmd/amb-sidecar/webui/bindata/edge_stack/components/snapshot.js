import  {LitElement, html} from '/edge_stack/vendor/lit-element.min.js';
import {registerContextChangeHandler, useContext} from '/edge_stack/components/context.js';
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

/**
 * This class wraps the snapshot that the server returns and provides
 * a set of more consistent, convenient, and document APIs for
 * accessing the data.
 */
class SnapshotWrapper {
  constructor(data) {
    this.data = data
  }

  /**
   * Return all the kubernetes resources (that the backend AES
   * instance is paying attention to) of the specified Kind.
   */
  getResources(kind) {
    // XXX we should really update this to return a uniform data
    // structure on the server side, but for now I'm patching over
    // that stuff here
    if (kind === "RateLimit") {
      return this.data.Limits || []
    } else {
      return ((this.data.Watt || {}).Kubernetes || {})[kind] || []
    }
  }

  /**
   * Return the JSON representation of the OSS diagnostics page.
   */
  getDiagnostics() {
    return this.data.Diag || {}
  }

}

export class Snapshot extends LitElement {

  /**
   * Subscribe to snapshots from the AES backend server. The
   * onSnapshotChange parameter is a function that will be passed an
   * instance of the SnapshotWrapper class.
   */
  static subscribe(onSnapshotChange) {
    const arr = useContext('aes-api-snapshot', null);
    onSnapshotChange(arr[0] || new SnapshotWrapper({}));
    registerContextChangeHandler('aes-api-snapshot', onSnapshotChange);
  }

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
    this.setAuthenticated = useContext('auth-state', null)[1];
    this.loading = true;

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
            this.setSnapshot(new SnapshotWrapper({}));
          }
        } else {
          response.json()
            .then((json) => {
              if (this.fragment == "trying") {
                window.location.hash = "";
              }
              this.fragment = ""
              this.setSnapshot(new SnapshotWrapper(json || {}))
              this.setAuthenticated(true)
              if (this.loading) {
                this.loading = false;
                document.onclick = () => {
                  fetch('/edge_stack/api/activity', {
                    method: 'POST',
                    headers: new Headers({
                      'Authorization': 'Bearer ' + getCookie("edge_stack_auth")
                    }),
                  });
                }
              }
              setTimeout(this.fetchData.bind(this), 1000);
            })
            .catch((err) => {
              console.log('error parsing snapshot', err);
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
