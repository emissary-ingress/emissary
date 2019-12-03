import  {LitElement, html} from '../vendor/lit-element.min.js';
import {registerContextChangeHandler, useContext} from './context.js';
import {getCookie} from './cookies.js';
import {ApiFetch} from "./api-fetch.js";

export const aes_res_editable   = "aes_res_editable";
export const aes_res_changed    = "aes_res_changed";
export const aes_res_source     = "aes_res_source";
export const aes_res_downloaded = "aes_res_downloaded";

export function updateCredentials(value) {
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
    return ((this.data.Watt || {}).Kubernetes || {})[kind] || []
  };

  /* Return all Kubernetes resources regardless of kind */
  getAllResources() {
    var allKinds  = (this.data.Watt || {}).Kubernetes || {}
    var resources = []

      for (const [key, value] of Object.entries(allKinds)) {
        if (value === null) { continue }
        resources = resources.concat(value)
    }

    return resources
  }

    /*
    * Return all the kubernetes resources (that the backend AES
    * instance is paying attention to) that have been changed by the user
    * with the Web UI.
    */
  getChangedResources() {
    /* Get every resource */
    var resources = this.getAllResources()

    /* filter on annotation: "aes-res-changed".
    *  if the key exists in the annotations, it's changed,
    *  and the value is the timestamp of the change.
    */
    var changed = resources.filter((res) => {
      let md = res.metadata;
      if ("annotations" in md) {
        var changed = md.annotations.aes_res_changed;
        /* changed is undefined, true, or false. */
        return changed === "true"
      }
      else {
        return false
      }
    });

    /* list of changed resources. */
    return changed;
  };


  /**
   * Return the JSON representation of the OSS diagnostics page.
   */
  getDiagnostics() {
    return this.data.Diag || {};
  };

  getLicense() {
    return this.data.License || {};
  };

  getRedisInUse() {
    return this.data.RedisInUse || false;
  };
}

export class Snapshot extends LitElement {

  /**
   * Subscribe to snapshots from the AES backend server. The
   * onSnapshotChange parameter is a function that will be passed an
   * instance of the SnapshotWrapper class.
   */
  static subscribe(onSnapshotChange) {
    const arr = useContext('aes-api-snapshot', new SnapshotWrapper({}));
    onSnapshotChange(arr[0]);
    registerContextChangeHandler('aes-api-snapshot', onSnapshotChange);
  }

  static get properties() {
    return {
      data: Object,
      loading: Boolean,
      loadingError: Boolean,
      fragment: String,
    };
  }

  constructor() {
    super();

    this.setSnapshot = useContext('aes-api-snapshot', new SnapshotWrapper({}))[1];
    this.setAuthenticated = useContext('auth-state', null)[1];
    this.loading = true;
    this.loadingError = null;

    if (getCookie("edge_stack_auth")) {
      this.fragment = "should-try";
    } else {
      updateCredentials(window.location.hash.slice(1));
      this.fragment = "trying";
    }
  }

  fetchData() {
    ApiFetch('/edge_stack/api/snapshot', {
      headers: {
        'Authorization': 'Bearer ' + getCookie("edge_stack_auth")
      }
    })
      .then((response) => {
        if (response.status === 400 || response.status === 401 || response.status === 403) {
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
          response.text()
            .then((text) => {
              var json;
              try {
                  json = JSON.parse(text);
              } catch(err) {
                this.loadingError = err;
                this.requestUpdate();
                console.error('error parsing snapshot', err);
                setTimeout(this.fetchData.bind(this), 1000);
                return
              }
              if (this.fragment === "trying") {
                window.location.hash = "";
              }

              this.fragment = "";
              this.setAuthenticated(true);
              this.setSnapshot(new SnapshotWrapper(json || {}));
              if (this.loading) {
                this.loading = false;
                this.loadingError = null;
                this.requestUpdate();
                document.onclick = () => {
                  ApiFetch('/edge_stack/api/activity', {
                    method: 'POST',
                    headers: new Headers({
                      'Authorization': 'Bearer ' + getCookie("edge_stack_auth")
                    }),
                  });
                }
              } else {
                if( this.loadingError ) {
                  this.loadingError = null;
                  this.requestUpdate();
                }
              }

              setTimeout(this.fetchData.bind(this), 1000);
            })
            .catch((err) => {
              this.loadingError = err;
              this.requestUpdate();
              console.error('error reading snapshot', err);
            })
        }
      })
      .catch((err) => {
        this.loadingError = err;
        this.requestUpdate();
        console.error('error fetching snapshot', err);
      })
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
    } else if (this.loadingError) {
      return html`
        <slot></slot>
        <dw-wholepage-error/>
      `;
    } else {
      return html`<slot></slot>`;
    }
  }
}

customElements.define('aes-snapshot-provider', Snapshot);
