import {LitElement, css, html} from '../vendor/lit-element.min.js'
import {registerContextChangeHandler, useContext} from './context.js'
import {ApiFetch, hasDebugBackend} from './api-fetch.js'
import {updateCredentials} from './snapshot.js'

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
button {
  margin-left: 1.5em;
}
button:hover {
  background-color: #ede7f3;
}
button:focus {
  background-color: #ede7f3;
}
div.login-repeatUser {
  display: flex;
  justify-content: space-between;
  border: 1px solid #ede7f3;
  box-shadow: 0 2px 4px rgba(0, 0, 0, .1);
  padding: 0.5em;
  margin-bottom: 0.6em;
  line-height: 1.3;
  position: relative;
}
div.login-newUser {
  display: flex;
  justify-content: space-between;
  padding-bottom: 0;
  border: 1px solid #ede7f3;
  border-top: 1px solid #ede7f3;
  border-right: 1px solid #ede7f3;
  box-shadow: 0 2px 4px rgba(0,0,0,.1);
  padding: 0.5em;
  line-height: 1.3;
  position: relative;
}
summary:focus {
    outline: none;
    color: #5F3EFF;
}
.darwinLink, .linuxLink{
    color: #5F3EFF;
}
div.login-section {
    border: 1px solid var(--dw-item-border);
    box-shadow: 0 2px 4px rgba(0,0,0,.1);
    padding: 0.5em;
    margin-bottom: 0.6em;
    line-height: 1.3;
}
div.login-container {
     display: flex;
     justify-content: space-between;
}
div.login-darwin {
    width: 50%;
    border: 1px solid #ede7f3;
    box-shadow: 0 2px 4px rgba(0, 0, 0, .1);
    padding: 0.5em;
    margin-bottom: 0.6em;
    line-height: 1.3;
    position: relative;
    overflow: hidden;
}
div.login-linux {
    width: 50%;
    border: 1px solid #ede7f3;
    box-shadow: 0 2px 4px rgba(0, 0, 0, .1);
    padding: 0.5em;
    margin-bottom: 0.6em;
    line-height: 1.3;
    position: relative;
    overflow: hidden;
}
div.info-title {
    font-weight: bold;
    font-size: 120%;
}
h1.info-title, p.login-downloadText {
    text-align: center;
}
p.login-instr {
    font-size: 90%;
    text-align: center;
    margin-bottom: 1em;
}
p.login-edgectl {
    font-size: 120%;
    text-align: center;
    margin-bottom: 2em;
}
p.download {
    margin: 16px;
}
span.repeatUserText {
    font-size: 140%;
}
span.command {
    background-color: #f5f2f0;
    color: #5F3EFF;
    padding: 3px;
    letter-spacing: .2px;
    font-family: Consolas,Monaco,Andale Mono,Ubuntu Mono,monospace;
    font-size: 150%;
    word-spacing: normal;
    word-break: normal;
    word-wrap: normal;
    hypens: none;
}
span.newUser {
    margin-left: 1em;
}
span.newUserIcon {
    font-size: 22px;
}
span.repeatUser{
    margin-left: 1em;
}
span.repeatUserIcon{
    font-size: 30px;
}
div.overage-alert {
    border: 3px solid red;
    border-radius: 0.7em;
    padding: 0.5em;
    background-color: #FFe8e8;
}
pre {
    margin: 0 1em 1em 1em;
    padding: .5em;
    background-color: #f5f2f0;
    letter-spacing: .2px;
    font-family: Consolas,Monaco,Andale Mono,Ubuntu Mono,monospace;
    font-size: 80%;
    white-space: normal;
    overflow-wrap: break-word;
    width: 92%;
}
details {
    margin: 1em;
}
#debug-dev-loop-box {
  width: 90%;
  text-align: center;
  border: thin dashed black;
  padding: 0.5em; 
}
#debug-dev-loop-box div {
  margin: auto;
}
#debug-dev-loop-box button {
}  
img#darwinLogo {
    width: 35px;
    margin: 0 0 -5px 0;
}
img#linuxLogo {
  width: 35px;
  margin: 0 0 -8px 0;
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
    if( this.authenticated !== auth ) {
      this.authenticated = auth;
      this.loading = false;
    }
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
        //this.loading = false;
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

  updated(changedProperties) {
    this.renderFocus();
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

  renderDebugDetails() {
    if( hasDebugBackend() ) {
    return html`
      <div id="debug-dev-loop-box"><div>
        <h3>Debug</h3>
        1. <button style="margin-right: 1em" @click=${this.copyLoginToKeyboard.bind(this)}>Copy edgectl command to clipboard</button>
        2. <button @click="${this.enterDebugDetails.bind(this)}">Enter the URL+JWT</button>
      </div></div>
` } else {
      return html``
    }
  }
  enterDebugDetails() {
    let the_whole_url = prompt("Enter the edgectl url w/ JST", "");
    let segments = the_whole_url.split('#');
    if( segments.length > 1 ) {
      updateCredentials(segments[1]);
    } else {
      updateCredentials(segments[0]);
    }
    window.location.reload();
  }

  renderDarwinDetails() {
    return html`
  <details id="darwin" ?open=${this.os === 'darwin'}>
  <summary id="darwinFocus"><h2 style="display:inline">MacOS
    <img id="darwinLogo" src="/edge_stack/images/logos/apple.svg" alt="linux logo" display=inline>
          </h2>
  </summary>
  <h3>1. Download with this CLI:</h3>
  
  <pre id="install-darwin">sudo curl -fL https://metriton.datawire.io/downloads/darwin/edgectl -o /usr/local/bin/edgectl && \\sudo chmod a+x /usr/local/bin/edgectl</pre>
  
  <button @click=${this.copyDarwinInstallToKeyboard.bind(this)}>Copy to Clipboard</button>
  <h3>2. Or download the executable:</h3>
  <p class="download">Download <a href="https://metriton.datawire.io/downloads/darwin/edgectl" class="darwinLink">edgectl for MacOS</a></p>
</details>
    `;
  }

  renderLinuxDetails() {
    return html`
    
<details id="linux" ?open=${this.os === 'linux'}>
  <summary id="linuxFocus"><h2 style="display:inline">Linux
    <img id="linuxLogo" src="/edge_stack/images/logos/linuxTux.svg" alt="linux logo" display=inline>
          </h2>
  </summary>
  <h3>1. Download with this CLI:</h3>
  
  <pre id="install-linux">sudo curl -fL https://metriton.datawire.io/downloads/linux/edgectl -o /usr/local/bin/edgectl && \\sudo chmod a+x /usr/local/bin/edgectl</pre>

  <button @click=${this.copyLinuxInstallToKeyboard.bind(this)}>Copy to Clipboard</button>
  <h3>2. Or download the executable:</h3>
  <p class="download">Download <a href="https://metriton.datawire.io/downloads/linux/edgectl" class="linuxLink">edgectl for Linux </a></p>
  
</details>
    `;
  }

  renderUnauthenticated() {
    return html`
  <div class="login-section">
    <h1 class="info-title">Welcome to the Ambassador Edge Stack</h1>
    <p class="login-downloadText">
    </p>
   
    <p class="login-instr">
      <div class="login-repeatUser">
        <span class="repeatUser">
          <span class="repeatUserIcon" style="display: inline">&#129413;&nbsp;</span>
            <span class="repeatUserText">Repeat users can log in to the Edge Policy Console directly with this command:<br></span>
    </p>

    <p class="login-edgectl">
      <span class="command" id="login" style="block">edgectl login --namespace=${this.namespace} ${window.location.host}</span> 
      <button style="margin-left: 1em" @click=${this.copyLoginToKeyboard.bind(this)}>Copy to Clipboard</button>
        </span>
    </p>  
  </div>

    <p class="login-instr">
    <div class="login-newUser">
    <span class="newUser"><span class="newUserIcon" style="display: inline">&#128037;&nbsp;</span>First time users will need to download and install the edgectl executable. Once complete, log in to Ambassador with the edgectl command above.</p>
    </span></div>
    
    <div class="login-container">
      <div class="login-darwin">
        ${this.renderDarwinDetails()}
      </div>
      <div class="login-linux">
        ${this.renderLinuxDetails()}
      </div>
    </div>
        ${this.renderDebugDetails()}
  </div>
    `;
  }

  renderFocus() {
    if (this.os === "darwin"){
      let element = this.shadowRoot.getElementById('darwinFocus');
      if( element ) { element.focus(); }
    } else if (this.os === "linux") {
      let element = this.shadowRoot.getElementById('linuxFocus');
      if( element ) { element.focus(); }
    }
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
