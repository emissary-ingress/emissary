import {LitElement, css, html} from '../vendor/lit-element.min.js'
import {registerContextChangeHandler, useContext} from './context.js'
import {ApiFetch, hasDebugBackend} from './api-fetch.js'
import {updateCredentials} from './snapshot.js'

export class BrowserDetector extends LitElement {
  static get properties() {
    return {
      authenticated: Boolean,
      loading: Boolean,
      hasError: Boolean,
      browser: String,
      namespace: String
    };
  }

  static get styles() {
    return css`
#unauthenticated-outer-wrapper {
  width: 80%;
  margin-left: 10%;
  background-color: #fff;
}
#ambassador-logo {
    background-color: white;
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
.darwinLink, .linuxLink, .windowsLink {
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
div.login-windows {
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
h2 {
  margin: 0.1em;
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
img#chromeLogo {
    width: 35px;
    margin: 0 0 -5px 0;
}
img#safariLogo {
  width: 35px;
  margin: 0 0 -8px 0;
}
img#firefoxLogo {
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
    this.browser = this.getBrowser();

    this.loadData();
  }

// Detects user's browser to guide response to browser security warnings
  getBrowser() {
    if (!!window.chrome && (!!window.chrome.webstore || !!window.chrome.runtime)) {
      return "chrome";
    } else if (/constructor/i.test(window.HTMLElement) || (function (p) { return p.toString() === "[object SafariRemoteNotification]"; })(!window['safari'] || (typeof safari !== 'undefined' && safari.pushNotification))) {
      return "safari";
    } else if (typeof InstallTrigger !== 'undefined') {
      return "firefox";
    } else {
      return "other";
    }
  }

renderChromeDetails() {
  return html`
<details id="chrome" ?open=${this.browser === 'chrome'}>
<summary id="chromeFocus"><h2 style="display:inline">Chrome
  <img id="chromeLogo" src="/edge_stack/images/logos/apple.svg" alt="apple logo" display=inline>
        </h2>
</summary>
<h3>1. Download with this CLI:</h3>

</details>
  `;
}

renderSafariDetails() {
  return html`

<details id="safari" ?open=${this.browser === 'safari'}>
<summary id="safariFocus"><h2 style="display:inline">Safari
  <img id="safariLogo" src="/edge_stack/images/logos/linuxTux.svg" alt="linux logo" display=inline>
        </h2>
</summary>
<h3>1. Download with this CLI:</h3>


</details>
  `;
}

renderFirefoxDetails() {
  return html`

<details id="firefox" ?open=${this.browser === 'firefox'}>
<summary id="firefoxFocus"><h2 style="display:inline">Firefox
  <img id="windowsLogo" src="/edge_stack/images/logos/windows.svg" alt="windows logo" display=inline>
        </h2>
</summary>
<h3>1. Download the executable:</h3>

</details>
  `;
}

renderUnauthenticated() {
    return html`
<div id="unauthenticated-outer-wrapper">
  <div class="login-container">
    <div class="login-darwin">
      ${this.renderChromeDetails()}
    </div>
    <div class="login-linux">
      ${this.renderSafariDetails()}
    </div>
    <div class="login-windows">
      ${this.renderFirefoxDetails()}
    </div>
  </div>
</div>
`;
}

customElements.define('browser-detector', BrowserDetector);