import {LitElement, html, css} from '../../vendor/lit-element.min.js'
import {getCookie} from '../../components/cookies.js'
import {ApiFetch} from "../../components/api-fetch.js"
import {top, bottom, close, tooltip} from './icons.js'

// todo: vendor these
import "https://cdn.jsdelivr.net/npm/xterm@4.4.0/lib/xterm.js"
import "https://cdn.jsdelivr.net/npm/xterm-addon-fit@0.3.0/lib/xterm-addon-fit.js"

class Term extends LitElement {

  static get properties() {
    return {
      source: {type: String},
      transform: {type: Function}
    };
  }

  static get styles() {
    return css`
      .controls {
        display: flex;
        flex-direction: row-reverse;
        padding: 0 0 7px 0;
      }

      .controls div {
        padding: 0 2px;
      }

      ${top()}
      ${bottom()}
      ${close()}
      ${tooltip()}
`
  }

  constructor() {
    super();
    this.source = "";
    this.transform = (x)=>x
    this.activeSource = this.source;
    this.term = null;
    this.es = null;
  }

  updated() {
    super.updated();
    this.updateSource();
  }

  updateSource() {
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

      let style = window.getComputedStyle(div)
      if (this.source && style.height === "auto") {
        // Try again later because we can't show the terminal if we
        // don't have a height yet. This appears to be due to a bug in
        // the fitter add on. It tries to compute the size of the
        // terminal, but then throws an exception because the computed
        // height is "auto" because the browser is still rendering
        // things.
        console.log("delaying terminal init because height is auto")
        window.requestAnimationFrame(this.updateSource.bind(this))
        return
      }

      if (this.source !== "") {
        this.term = new Terminal({rows: 24, cols: 40, convertEol: true});
        let fitter = new FitAddon.FitAddon();
        this.term.loadAddon(fitter);
        this.term.open(div);
        fitter.fit();
        // todo: auth?
        this.es = new EventSource(this.source, {withCredentials: true});
        this.es.onerror = (e) => { console.log("terminal event error", e); };
        this.es.addEventListener("close", (e) => {
          this.es.close();
        });
        this.es.onmessage = (e) => {
          let xformed = this.transform(e.data)
          if (xformed !== undefined) {
            this.term.write(xformed + "\n");
          }
        };
      }

      this.activeSource = this.source;
    }
  }

  onClose() {
    this.dispatchEvent(new CustomEvent("close"))
  }

  render() {
    this.updateSource();
    return html`
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/xterm@4.4.0/css/xterm.css" integrity="sha256-I3n7q4Kl55oWvltoLRCCpA5HW8W3O34RUeC/ob43fWY=" crossorigin="anonymous">
<div style="display:${this.source ? "block" : "none"}">
  <div class="controls">
    <div class="close" @click=${()=>this.onClose()}>
      <div class="tooltip"><p>Close Terminal</p></div>
    </div>
    <div class="bottom" @click=${()=>this.term.scrollToBottom()}>
      <div class="tooltip"><p>Scroll to Bottom</p></div>
    </div>
    <div class="top" @click=${()=>this.term.scrollToTop()}>
      <div class="tooltip"><p>Scroll to Top</p></div>
    </div>
  </div>
  <div id="terminal"></div>
</div>
`;
  }

}

customElements.define('dw-terminal', Term);
