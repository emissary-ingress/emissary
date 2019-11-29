import {LitElement, css, html} from '../vendor/lit-element.min.js';

export class Error extends LitElement {
  static get properties() {
    return {
      description: { type: String }
    }
  }

  constructor() {
    super();
    this.description = html`Having trouble connecting to the Kubernetes cluster...<br>please check your network connectivity.`;
  }

  static get styles() {
    return css`
div.outer {
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
#spinner-img {
  width: 40px;
  height: 40px;
  overflow: hidden;
  float: left;
}
img {
  width: 218px;
  height: 149px;
  margin: -54px 0 0 -90px;
}
#inner {
}
`
  }

  render() {
    return html`
<div class="outer">
    <div id="spinner-img"><img src="/edge_stack/images/busy_spinner.gif"/></div>
    <div id="inner">${this.description}</div>
</div>`;
  }
}

customElements.define('dw-wholepage-error', Error);
