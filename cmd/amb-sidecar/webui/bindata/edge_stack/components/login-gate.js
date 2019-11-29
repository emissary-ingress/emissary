import {LitElement, css, html} from '../vendor/lit-element.min.js';
import {registerContextChangeHandler, useContext} from './context.js';
import {ApiFetch} from "./api-fetch.js";

export class LoginGate extends LitElement {
  static get properties() {
    return {
      authenticated: Boolean,
      loading: Boolean,
      hasError: Boolean,
      os: String,
      namespace: String
    };
  }

  static get styles() {
    return css`
:host {
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
div.login-section {
    border: 1px solid var(--dw-item-border);
    box-shadow: 0 2px 4px rgba(0,0,0,.1);
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
    font-size: 80%;
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
pre {
    margin: 1em;
    padding: .5em;
    background-color: #f5f2f0;;
    letter-spacing: .2px;
    font-family: Consolas,Monaco,Andale Mono,Ubuntu Mono,monospace;
    font-size: 80%;
    word-spacing: normal;
    word-break: normal;
    word-wrap: normal;
    hypens: none;
}
details {
    margin: 1em;
}
    `;
  }

  constructor() {
    super();

    this.hasError = false;
    this.loading = true;

    this.namespace = '';
    this.os = this.getOS();

    this.loadData();

    this.authenticated = useContext('auth-state', null)[0];
    registerContextChangeHandler('auth-state', this.onAuthChange.bind(this));
  }

  onAuthChange(auth) {
    this.authenticated = auth
  }

  getOS() {
    if (window != null && window["navigator"] != null && window["navigator"]["platform"] != null) {
      const os = window.navigator.platform; // Mac, Win, Linux

      if (os.toLocaleLowerCase().indexOf("win") != -1) {
        return "windows";
      } else if (os.toLowerCase().indexOf("mac") != -1) {
        return "darwin";
      } else if (os.toLowerCase().indexOf("linux") != -1) {
        return "linux";
      } else {
        return "other";
      }
    } else {
      return "other";
    }
  }

  loadData() {
    ApiFetch('/edge_stack/api/config/pod-namespace')
    //fetch('http://localhost:9000/edge_stack/api/config/pod-namespace', { mode:'no-cors'})
      .then(data => data.text()).then(body => {
        this.namespace = body;
        this.loading = false;
        this.hasError = false;
      })
      .catch((err) => {
        console.error(err);
        this.loading = false;
        this.hasError = true;
      });
  }

  renderError() {
    return html`
<dw-wholepage-error/>
    `;
  }

  renderLoading() {
    return html`
<p>Loading...</p>
    `;
  }

  copyToKeyboard(theId) {
    const copyText = this.shadowRoot.getElementById(theId).innerText;
    const el = document.createElement('textarea');  // Create a <textarea> element
    el.value = copyText;                            // Set its value to the string that you want copied
    el.setAttribute('readonly', '');                // Make it readonly to be tamper-proof
    el.style.position = 'absolute';
    el.style.left = '-9999px';                      // Move outside the screen to make it invisible
    document.body.appendChild(el);                  // Append the <textarea> element to the HTML document
    const selected =
      document.getSelection().rangeCount > 0        // Check if there is any content selected previously
        ? document.getSelection().getRangeAt(0)     // Store selection if found
        : false;                                    // Mark as false to know no selection existed before
    el.select();                                    // Select the <textarea> content
    document.execCommand('copy');                   // Copy - only works as a result of a user action (e.g. click events)
    document.body.removeChild(el);                  // Remove the <textarea> element
    if (selected) {                                 // If a selection existed before copying
      document.getSelection().removeAllRanges();    // Unselect everything on the HTML document
      document.getSelection().addRange(selected);   // Restore the original selection
    }
  }

  copyLoginToKeyboard() {
    this.copyToKeyboard('login');
  }

  copyDarwinInstallToKeyboard() {
    this.copyToKeyboard('install-darwin');
  }

  copyLinuxInstallToKeyboard() {
    this.copyToKeyboard('install-linux');
  }

  renderDarwinDetails() {
    return html`
<details id="darwin" ?open=${this.os === 'darwin'}>
  <summary><h2 style="display:inline">Darwin</h2></summary>
  <pre id="install-darwin">
sudo curl -fL https://metriton.datawire.io/downloads/darwin/edgectl -o /usr/local/bin/edgectl && \\
sudo chmod a+x /usr/local/bin/edgectl
  </pre>

  <button @click=${this.copyDarwinInstallToKeyboard.bind(this)}>Copy to Clipboard</button>
</details>
    `;
  }

  renderLinuxDetails() {
    return html`
<details id="linux" ?open=${this.os === 'linux'}>
  <summary><h2 style="display:inline">Linux</h2></summary>
  <pre id="install-linux">
sudo curl -fL https://metriton.datawire.io/downloads/linux/edgectl -o /usr/local/bin/edgectl && \\
sudo chmod a+x /usr/local/bin/edgectl
  </pre>

  <button @click=${this.copyLinuxInstallToKeyboard.bind(this)}>Copy to Clipboard</button>
</details>
    `;
  }

  renderUnauthenticated() {
    return html`
  <div class="login-section">
    <h1 class="info-title">Welcome to Ambassador Edge Stack!</h1>
    <p>
      To start using the Edge Policy Consule, download the edgectl executable
      from the getambassador.io
      website: (<a href="https://metriton.datawire.io/downloads/darwin/edgectl">darwin</a>, <a href="https://metriton.datawire.io/downloads/linux/edgectl">linux</a>).
    </p>
    <p>
    Once downloaded, you can login to the Edge Policy Console with: <span class="command" id="login">edgectl login --namespace=${this.namespace} ${window.location.host}</span> <button style="margin-left: 1em" @click=${this.copyLoginToKeyboard.bind(this)}>Copy to Clipboard</button>
    </p>

    ${this.renderDarwinDetails()}
    ${this.renderLinuxDetails()}
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
      return html`<slot></slot>`;
    }
  }
}

customElements.define('login-gate', LoginGate);
