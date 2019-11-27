import {LitElement, css, html} from '/edge_stack/vendor/lit-element.min.js';

export class Error extends LitElement {
  static get properties() {
    return {
      description: { type: String }
    }
  }

  constructor() {
    super();
    this.description = "Having trouble connecting to the Kubernetes cluster...<br>please check your network connectivity.";
  }

  static get styles() {
    return css`
div {
  position: absolute;
  left: 20%;
  top: 2em;
  width: 60%;
  text-align: center;
  background-color: #fef4f4;
  padding: 1em;
  font-weight: bold;
  border: 3px #fecccc solid;
  box-shadow: rgba(0,0,0,0.3) 0px 2px 4px;
}
`
  }

  render() {
    return html( ["<div>" + this.description + "</div>"]);
  }
}

customElements.define('dw-wholepage-error', Error);
