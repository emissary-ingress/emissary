import { LitElement, html, css } from 'https://cdn.pika.dev/-/lit-element/2.2.1/dist-es2019/lit-element.min.js'

export class Documentation extends LitElement {
  static get properties() {
    return {};
  }

  constructor() {
    super();
  }

  static get styles() {
    return css`
ul {
  display: flex;
  flex-direction: row;
  flex-wrap: wrap;
  padding: 0;
}

ul > li {
  display: block;
}

ul > li > a {
  display: block;
  width: 2in;
  height: 3in;
  text-align: center;

  margin: 0.4em;
  border: 2px solid #ede7f3;
  border-radius: 0.4em;
  background-color: #fdfaff;

  text-decoration: none;
}

ul > li > a:hover {
  background-color: #ede7f3;
}

ul > li > a > * {
  display: block;
  margin: 1em;
}

img {
  height: 1.7in;
  margin-left: auto;
  margin-right: auto;
}
`;
  }

  render() {
    return html`
      <ul>

        <li><a href="https://www.getambassador.io/docs/" target="_blank">
          <img src="/edge_stack/images/card-docs.png" alt="Ambassador Edge Stack Documentation"/>
          <span>Ambassador Edge Stack Documentation</span>
        </a></li>

        <li><a href="https://www.getambassador.io/resources/" target="_blank">
          <img src="/edge_stack/images/card-resources.png" alt="Resources and Case Studies"/>
          <span>Resources and Case Studies</span>
        </a></li>

        <li><a href="https://blog.getambassador.io/" target="_blank">
          <img src="/edge_stack/images/logos/medium-mark.png" alt="Blog"/>
          <span>Blog</span>
        </a></li>

      </ul>
    `;
  }
}
customElements.define('dw-documentation', Documentation);
