import { Model } from '../framework/model.js'
import { View, html, css } from '../framework/view.js'

/**
 * Display a log of errors in accordion style.
 */
class Errors extends View {

  static get properties() {
    return {
      errors: {type: Array},
      expanded: {type: Set}
    }
  }

  static get styles() {
    return css`
      .error {
        white-space: nowrap;
        overflow: hidden;
        width: 80ch;
      }
      .expanded {
        white-space: normal;
      }
      .triangle {
        display: inline-block;
        font-size: large;
      }
      .expanded .triangle {
        transform: rotate(90deg);
      }
      .message {
        display: inline-block;
        color: red;
      }
      .expanded .message {
        margin-left: 0.8em;
	background: #eee;
	border-radius: 5px;
	padding: 5px 10px 5px 10px;
      }
`
  }

  constructor() {
    super()
    this.errors = []
    this.expanded = new Set()
  }

  render() {
    if (this.errors.length) {
      return html`<div>${this.errors.map((e, idx)=>this.renderError(e, idx))}</div>`
    } else {
      return html``
    }
  }

  renderError(e, idx) {
    let expanded = this.expanded.has(idx)
    let message = expanded ? e.message : e.message.slice(0, 77)
    if (message !== e.message) {
      message += "..."
    }
    return html`<div class="${expanded ? "expanded" : ""} error" @click=${()=>this.toggle(idx)}><div class="triangle"><div>&#8227;</div></div><div class="message">${message}</div></div>`
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
