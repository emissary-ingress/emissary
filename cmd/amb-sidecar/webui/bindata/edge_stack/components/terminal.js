import {LitElement, html, css} from '../vendor/lit-element.min.js'
import {getCookie} from './cookies.js';
import {ApiFetch} from "./api-fetch.js";

// todo: vendor these
import "https://cdn.jsdelivr.net/npm/xterm@4.4.0/lib/xterm.js";
import "https://cdn.jsdelivr.net/npm/xterm-addon-fit@0.3.0/lib/xterm-addon-fit.js";

class Term extends LitElement {

  static get properties() {
    return {
      source: {type: String}
    };
  }

  constructor() {
    super();
    this.source = "";
    this.activeSource = this.source;
    this.term = null;
    this.es = null;
  }

  render() {
    let div = this.shadowRoot.getElementById("terminal")
    if (div !== null && div.isConnected && this.activeSource !== this.source) {
      if (this.term !== null) {
        this.term.dispose();
        this.term = null;
      }

      if (this.es !== null) {
        this.es.close();
        this.es = null;
      }

      if (this.source !== "") {
        this.term = new Terminal({rows: 24, cols: 40, convertEol: true});
        let fitter = new FitAddon.FitAddon();
        this.term.loadAddon(fitter);
        this.term.open(div);
        fitter.fit();
        // todo: auth?
        this.es = new EventSource(this.source);
        this.es.onerror = (e) => { console.log("terminal event error", e); };
        this.es.addEventListener("close", (e) => {
          this.es.close();
        });
        this.es.onmessage = (e) => {
          this.term.write(e.data + "\n");
        };
      }

      this.activeSource = this.source;
    }

    return html`
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/xterm@4.4.0/css/xterm.css" integrity="sha256-I3n7q4Kl55oWvltoLRCCpA5HW8W3O34RUeC/ob43fWY=" crossorigin="anonymous">
<div id="terminal"></div>
`;
  }

}

customElements.define('dw-terminal', Term);
