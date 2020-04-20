import { Model } from '../framework/model.js'
import { View, html, css } from '../framework/view.js'
import { controls } from './icons.js'
import { copy } from './helpers.js'

/**
 * Display a log of errors in accordion style.
 */
class Errors extends View {

  static get properties() {
    return {
      summarize: {type: Boolean},
      errors: {type: Array},
      expanded: {type: Set},
      columns: {type: Number}
    }
  }

  static get styles() {
    return css`
      .summary {
        display: inline-block;
      }
      .error-count {
        color: red;
      }
      .errors {
        font-family: monospace;
        max-height: 60ch;
        overflow-y: auto;
      }
      .error {
        display: flex;
        white-space: nowrap;
      }
      .error, .plus, .minus {
        margin: 0.2em;
      }
      .expanded {
        white-space: normal;
      }
      .block {
        display: flex;
        padding: 0.2em;
        background: #eee;
        border-radius: 5px;
      }
      .message {
        color: red;
        flex-grow: 1;
        width: calc(var(--columns));
      }
      .expanded .message {
        white-space: pre-wrap;
        width: calc(var(--columns) - 44px);
      }
      .error .controls {
        display: none;
      }
      .expanded .controls {
        display: flex;
      }
      .off {
        display: none;
      }

      ${controls()}

      .link {
        color: blue;
        text-decoration: none;
      }

      .link:hover {
        cursor: pointer;
      }
`
  }

  constructor() {
    super()
    this.errors = []
    this.expanded = new Set()
    this.summarize = true
    this.columns = 70
  }

  bugUrl() {
    let start = "Steps to reproduce:\n  1.\n\n```\n"
    let end = "\n```"
    let urlFor = (e)=>`https://github.com/datawire/ambassador/issues/new?body=${encodeURIComponent(start+e+end)}`

    let max = 4096
    let encoded = JSON.stringify(this.errors)
    let overhead = urlFor("...").length
    let shortened = encoded.slice(0, max-overhead)
    if (shortened !== encoded) {
      shortened += "..."
    }
    return urlFor(shortened)
  }

  render() {
    if (this.errors.length) {
      let bugUrl = this.bugUrl()
      return html`
<div class="summary">
  <span class="error-count">${this.errors.length} error${this.errors.length === 1 ? "" : "s"}</span>
    &mdash;
  <span class="link" @click=${()=>this.toggleSummarize()}>${this.summarize ? "show" : "hide" }</span>
    |
  <a class="link" target="_blank" href="${bugUrl}">report a bug</a>
</div>
<div class=${this.summarize ? "off" : "errors"} style="--columns: ${this.columns}ch">${this.errors.map((e, idx)=>this.renderError(e, idx))}</div>
`
    } else {
      return html``
    }
  }

  toggleSummarize() {
    this.summarize = !this.summarize
  }

  renderError(e, idx) {
    let expanded = this.expanded.has(idx)
    let message = expanded ? e.message : e.message.slice(0, this.columns - 3)
    if (message !== e.message) {
      message += "..."
    }
    return html`
<div class="${expanded ? "expanded" : ""} error">
  <div class="${expanded ? "minus" : "plus"}" @click=${()=>this.toggle(idx)}></div>
  <div class="block">
    <div class="message" @click=${()=>this.toggle(idx)}>${message}</div>
    <div class="controls">
      <div class="copy" @click=${()=>copy(e.message)}></div>
      <div class="close" @click=${()=>this.toggle(idx)}></div>
    </div>
  </div>
</div>`
  }

  toggle(idx) {
    if (this.expanded.has(idx)) {
      this.expanded.delete(idx)
    } else {
      this.expanded.add(idx)
    }
    this.requestUpdate("expanded")
  }

}

customElements.define("dw-errors", Errors)
