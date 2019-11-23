import { LitElement, html, css } from 'https://cdn.pika.dev/-/lit-element/2.2.1/dist-es2019/lit-element.min.js'

export class Documentation extends LitElement {
  static get properties() {
    return {};
  }

  constructor() {
    super();
  }

  static get styles() {
    return css``;
  }

  render() {
    return html`
      <ul>
        <li><a href="https://getambassador.io/docs/" target="_blank">Ambassador Documentation</a></li>
        <li><a href="https://getambassador.io/concepts/overview" target="_blank">Concepts</a></li>
        <li><a href="https://getambassador.io/docs/guides/" target="_blank">Guides</a></li>
        <li><a href="https://getambassador.io/reference/configuration" target="_blank">Configuration Reference</a></li>
      </ul>
    `;
  }
}
customElements.define('dw-documentation', Documentation);
