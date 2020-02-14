import {LitElement, html} from '../vendor/lit-element.min.js';
import {registerContextChangeHandler, useContext} from './context.js';
import {getCookie} from './cookies.js';
import {ApiFetch} from "./api-fetch.js";

export const aes_res_editable   = "getambassador.io/resource-editable";
export const aes_res_changed    = "getambassador.io/resource-changed";
export const aes_res_source     = "getambassador.io/resource-source";
export const aes_res_downloaded = "getambassador.io/resource-downloaded";

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
  constructor(previousData, newData) {
    // Save the raw data
    this.data = {
      Watt: newData.Watt,
      License: newData.License,
      RedisInUse: newData.RedisInUse,
    };

    if (Array.isArray(newData.Diag)) {
      // `Diag` is an array, assume it's a json-patch and not a full representation
      if (previousData && previousData.Diag) {
        this.data.Diag = previousData.Diag;
      }
      try {
        this.data.Diag = jsonpatch.applyPatch(this.data.Diag || {}, newData.Diag).newDocument;
      } catch (err) {
        console.error('Snapshot Diag update failed!', err);
      }
    } else {
      // `Diag` is coming in as full JSON representation
      this.data.Diag = newData.Diag;
    }
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
        var changed = md.annotations[aes_res_changed];
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
    const arr = useContext('aes-api-snapshot', new SnapshotWrapper({}, {}));
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

    Snapshot.subscribe((snapshot)=>{
      this.currentSnapshot = snapshot;
    });

    // This is basically just a feature-flag to short-circuit the Snapshot patches
    this.snapshotPatches = true;
    // Use a unique token for this page's lifetime; enabling Snapshot patches
    this.clientSession = Math.random();

    this.setSnapshot = useContext('aes-api-snapshot', new SnapshotWrapper(this.currentSnapshot.data, {}))[1];
    this.setAuthenticated = useContext('auth-state', null)[1];
    this.loading = true;
    this.loadingError = null;

/*    this.cookieChanged = false;

    this.checkCookie = function() {
      var lastCookie = document.cookie; // 'static' memory between function calls
      console.log("lastCookie is " + lastCookie);
      return function() {
        var currentCookie = 100;;
        console.log("currentCookie is " + currentCookie);
        if (currentCookie != lastCookie) {
          console.log("cookie changed");
          this.cookieChanged = true;
          console.log(this.cookieChanged);
          console.log(cookieChanged);
          this.lastCookie = this.currentCookie; // store latest cookie
          return this.cookieChanged;
        }
      };  
    }();

    window.setInterval(this.checkCookie, 10000);
//    console.log('cookie checked');
    console.log('cookie changed is ' + this.cookieChanged);
    console.log("document.cookie is" + document.cookie);
    console.log("lastCookie is" + this.lastCookie);
    console.log("cookieChanged is " + this.cookieChanged);
    console.log("currentCookie is" + this.currentCookie);
*/

    if (getCookie("edge_stack_auth")) {
      this.fragment = "should-try";
    } else {
      updateCredentials(window.location.hash.slice(1));
      this.fragment = "trying";
    }
  }

  queueNextSnapshotPoll() {
    if( Snapshot.theTimeoutId === 0 ) { // if we aren't already waiting to fetch a new snapshot...
    Snapshot.theTimeoutId = setTimeout(this.fetchData.bind(this), 1000); // fetch a new snapshot every second
    }
  }

  clearTimeout() { // it's ok to clear a timeout that has already expired
    Snapshot.theTimeoutId = 0;
  }      

  fetchData() {
      if( Snapshot.theTimeoutId !== 0 ) {
        this.clearTimeout(Snapshot.theTimeoutId).bind(this);
      }
    ApiFetch(`/edge_stack/api/snapshot?client_session=${this.snapshotPatches ? this.clientSession : ''}`, {
      headers: {
        'Authorization': 'Bearer ' + getCookie("edge_stack_auth")
      }
    })
      .then(this.handleFetchResponse.bind(this))
      .catch(this.handleFetchResponseError.bind(this))
      this.queueNextSnapshotPoll();
  }

  handleFetchResponse(response) {
    if (response.status === 400 || response.status === 401 || response.status === 403) {
      if (this.fragment === "should-try") {
        updateCredentials(window.location.hash.slice(1));
        this.fragment = "trying";
        setTimeout(this.fetchData.bind(this), 0); // try again immediately
      } else {
        this.fragment = "";
        this.setAuthenticated(false);
        this.setSnapshot(new SnapshotWrapper(this.currentSnapshot.data, {}));
        this.queueNextSnapshotPoll();
      }
    } else {
      response.text()
        .then(this.handleValidText.bind(this))
        .catch(this.handleValidTextError.bind(this))
        this.queueNextSnapshotPoll();
    }
  }

  handleFetchResponseError(err) {
    this.loadingError = err;
    this.requestUpdate();
    console.error('error fetching snapshot', err);
  }

  handleValidText(text) {
    var json;
    queueNextSnapshotPoll();
    try {
        json = JSON.parse(text);
    } catch(err) {
      this.loadingError = err;
      this.requestUpdate();
      console.error('error parsing snapshot', err);
      return
    }
    if (this.fragment === "trying") {
      window.location.hash = "";
    }
    this.fragment = "";
    this.setAuthenticated(true);
    this.setSnapshot(new SnapshotWrapper(this.currentSnapshot.data, json || {}));
    if (this.loading) {
      this.loading = false;
      this.loadingError = null;
      this.requestUpdate();
      this.recordUserActivity();
    } else {
      if( this.loadingError ) {
        this.loadingError = null;
        this.requestUpdate();
      }
    }
  }

  recordUserActivity() {
    document.onclick = () => {
      ApiFetch('/edge_stack/api/activity', {
        method: 'POST',
        headers: new Headers({
          'Authorization': 'Bearer ' + getCookie("edge_stack_auth")
        }),
      });
    }
  }  

  handleValidTextError(err) {
    this.loadingError = err;
    this.requestUpdate();
    console.error('error reading snapshot', err);
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

Snapshot.theTimeoutId = 0; // we use this to make sure that we only ever have one active timeout

customElements.define('aes-snapshot-provider', Snapshot);
