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
  left: 20%;
  top: 2em;
  width: 60%;
  text-align: center;
  background-color: #fecccc;
  padding: 4px 0 4px 0;
  font-weight: bold;
  border: 3px #fe0000 solid;
  box-shadow: rgba(0,0,0,0.3) 0px 2px 4px;
}
`
  }

  render() {
    return html`<div>${this.description}</div>`
  }
}

customElements.define('dw-wholepage-error', Error);
