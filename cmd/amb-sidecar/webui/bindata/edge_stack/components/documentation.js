import { LitElement, html, css } from '../vendor/lit-element.min.js'

export class Documentation extends LitElement {
  static get properties() {
    return {};
  }

  constructor() {
    super();
  }

  static get styles() {
    return css`
div.center {
   margin: auto;
}

ul {
  display: flex;
  flex-direction: row;
  flex-wrap: wrap;
  padding: 0;
  justify-content: center;
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
  border: 2px solid var(--dw-item-border);
  border-radius: 10px;
  background-color: #fff;
  padding: 10px 30px 10px 30px;
  box-shadow: 0 10px 5px -11px rgba(0, 0, 0, .6);

  text-decoration: none;
  color: black;
}

ul > li > a:hover {
  color: #5f3eff;
  transition: all .2s ease;
  border: 2px #5f3eff solid
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
      <div class="center"><ul>

        <li><a href="https://www.getambassador.io/docs/" target="_blank">
          <img src="../images/card-docs.png" alt=""/>
          <span>Ambassador Edge Stack Documentation</span>
        </a></li>

        <li><a href="https://www.getambassador.io/resources/" target="_blank">
          <img src="../images/card-resources.png" alt=""/>
          <span>Resources and<br/>Case Studies</span>
        </a></li>

        <li><a href="https://blog.getambassador.io/" target="_blank">
          <img src="../images/logos/medium-mark.png" alt=""/>
          <span>Ambassador Edge Stack Blog</span>
        </a></li>

      </ul></div>
    `;
  }
}
customElements.define('dw-documentation', Documentation);
