import {LitElement, css, html} from '/edge_stack/vendor/lit-element.min.js';

export class Error extends LitElement {
  static get properties() {
    return {
      description: { type: String }
    }
  }

  constructor() {
    super();
    this.description = "Unknown error";
  }

  static get styles() {
    return css`
div {
  position: absolute;
  left: 0;
  top: 35px;
  width: 100%;
  text-align: center;
  background-color: #fe0000;
  padding: 4px 0 4px 0;
  font-weight: bold;
}
`
  }

  render() {
    return html`<div>${this.description}</div>`
  }
}

customElements.define('dw-error', Error);
