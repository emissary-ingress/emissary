import {LitElement, css, html} from '../vendor/lit-element.min.js'

export class BrowserWarning extends LitElement {
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
body {
    width:100%;
    height: 100%;
}
details {
    margin: 1em;
    align-content: center;
    text-align: center;
}
#browser-warning-outer-wrapper {
    width: 100%;
    background-color: #fff;
    align-content: center;
    text-align: center;
}
div.login-container {
    display: flex;
    justify-content: space-between;
}
div.login-section {
    border: 1px solid var(--dw-item-border);
    box-shadow: 0 2px 4px rgba(0,0,0,.1);
    padding: 0.5em;
    margin-bottom: 0.6em;
    line-height: 1.3;
}
div.warning-chrome .warning-safari .warning-firefox .warning-other{
    width: 50%;
    border: 1px solid #ede7f3;
    box-shadow: 0 2px 4px rgba(0, 0, 0, .1);
    padding: 0.5em;
    margin-bottom: 0.6em;
    line-height: 1.3;
    position: relative;
    overflow: hidden;
    align-content: center;
    text-align: center;
}
h3 {
    margin: 0.1em;
}
img#chromeLogo {
    width: 25px;
    margin: 0 0 -5px 0;
}
img#safariLogo {
    width: 25px;
    margin: 0 0 -8px 0;
}
img#firefoxLogo {
    width: 25px;
    margin: 0 0 -8px 0;
}
img#securityWarning {
    height: 100%;
    width: 70%;
    display: block;
    margin-top: 8px;
    align-content: center;
}
p#browser-warning-text {
    font-size: 70%;
    width: 200px;
    margin-left: 10px;
    margin-right: 5px;
    margin-top: 15px;
    text-align: left;
}
p#browser-warning-text-other{
  font-size: 90%;
  width: 100%;
  margin-left: 10px;
  margin-right: 5px;
  margin-top: 15px;
  text-align: left;
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
summary:hover {
    outline: none;
    color: #5F3EFF;
}
summary:focus {
    outline: none;
    color: #5F3EFF;
}
summary {
  list-style: none;
  margin-bottom: 10px;
  prevent-default: true;
}
details > summary::-webkit-details-marker {
  display: none;
}
.dropdown {
  border: 1px solid #ede7f3;
  box-shadow: 0 2px 4px rgba(0, 0, 0, .1);
  padding: 0.5em;
  margin-bottom: 0.6em;
  line-height: 1.3;
  position: relative;
  display: flex;
}
    `;
  }

  constructor() {
    super();
    this.browser = this.getBrowser();
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

renderChromeDetails() {
  return html`
    <div class="warning-chrome">
      <details id="chrome" ?open=${this.browser === 'chrome'}>
        <summary id="chromeFocus"><h3 style="display:inline" type="button">Chrome
          <img id="chromeLogo" src="/edge_stack/images/logos/chrome.svg" alt="chrome logo" display=inline></h3>
        </summary>
        <div class="dropdown">
          <p id="browser-warning-text">This warning appears because a secure connection has not yet been established. Click 'Advanced' to view details and then 'Proceed' to continue to the Edge Policy Console, where you can set your own certificate. Image provided is a generic example for reference.</p>  
          <img id="securityWarning" src="/edge_stack/images/svgs/chromeSecWarning.png" alt="chrome security warning logo">
        </details>
      </div> 
    </div>
  `;
}

renderSafariDetails() {
  return html`
    <div class="warning-safari">
      <details id="safari" ?open=${this.browser === 'safari'}>
        <summary id="safariFocus"><h3 style="display:inline">Safari</h3>
          <img id="safariLogo" src="/edge_stack/images/logos/safari.svg" alt="safari logo">
        </summary>
        <div class="dropdown">
          <p id="browser-warning-text">This warning appears because a secure connection has not yet been established. Click 'Advanced' to view details and then 'Proceed' to continue to the Edge Policy Console, where you can set your own certificate. Image provided is a generic example for reference.</p>  
          <img id="securityWarning" src="/edge_stack/images/svgs/safariSecWarning.png" alt="safari security warning logo"> 
        </details>
      </div>
    </div>
  `;
}

renderFirefoxDetails() {
  return html`
    <div class="warning-firefox">
      <details id="firefox" ?open=${this.browser === 'firefox'}>
        <summary id="firefoxFocus"><h3 style="display:inline">Firefox</h3>
          <img id="firefoxLogo" src="/edge_stack/images/logos/firefox.png" alt="firefox logo">
        </summary>
        <div class="dropdown">
          <p id="browser-warning-text">This warning appears because a secure connection has not yet been established. Click 'Advanced' to view details and then 'Accept the Risk and Continue' to the Edge Policy Console, where you can set your own certificate. Image provided is a generic example for reference.</p>  
          <img id="securityWarning" src="/edge_stack/images/svgs/firefoxSecWarning.png" alt="firefox security warning logo"> 
        </details>
      </div>
    </div>
  `;
}

renderOtherDetails() {
  return html`
    <div class="warning-other">
      <details id="other" ?open=${this.browser === 'other'}>
        <summary id="otherFocus"><h3 style="display:inline">Browser Security Warning</h3>
        </summary>
        <div class="dropdown">
          <p id="browser-warning-text-other">This warning appears because a secure connection has not yet been established. Follow browser instructions to view details and Accept Risk/Proceed to continue to the Edge Policy Console, where you can set your own certificate.</p>  
        </details>
      </div> 
    </div>
  `;
}

  renderBrowserWarning() {
    return html`
      <div id="browser-warning-outer-wrapper">
        ${this.renderChromeDetails()}

        ${this.renderSafariDetails()}

        ${this.renderFirefoxDetails()}

        ${this.renderOtherDetails()}
      </div>
    `;
  }

renderFocus() {
  let chrome = this.shadowRoot.getElementById('chromeFocus');
  let safari = this.shadowRoot.getElementById('safariFocus');
  let firefox = this.shadowRoot.getElementById('firefoxFocus');
  let other = this.shadowRoot.getElementById('otherFocus');
  if (this.browser === "chrome") {
    let element = this.shadowRoot.getElementById('chromeFocus');
      if (safari) { safari.setAttribute("hidden", "true") };
      if (firefox) { firefox.setAttribute("hidden", "true") };
      if (other) { other.setAttribute("hidden", "true") };
      if (element) { element.focus() }; 
  } else if (this.browser === "safari") {
    let element = this.shadowRoot.getElementById('safariFocus');
      if (chrome) { chrome.setAttribute("hidden", "true") };
      if (firefox) { firefox.setAttribute("hidden", "true") };
      if (other) { other.setAttribute("hidden", "true") };
      if (element) { element.focus() };
  } else if (this.browser === "firefox") {
    let element = this.shadowRoot.getElementById('firefoxFocus');
      if (chrome) { chrome.setAttribute("hidden", "true") };
      if (safari) { safari.setAttribute("hidden", "true") };
      if (other) { other.setAttribute("hidden", "true") };
      if (element) { element.focus() };
  } else if (this.browser === "other") {
    let element = this.shadowRoot.getElementById('otherFocus');
      if (chrome) { chrome.setAttribute("hidden", "true") };
      if (safari) { safari.setAttribute("hidden", "true") };
      if (firefox) { firefox.setAttribute("hidden", "true") };
      if (element) { element.focus() } ;
    }
  }

render() {
  if (this.hasError) {
    return this.renderError();
  } else if (this.loading) {
    return this.renderLoading();
  } else {
    return this.renderBrowserWarning();
  } 
  }
}
customElements.define('browser-warning', BrowserWarning);