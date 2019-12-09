import { LitElement, html, css } from '../vendor/lit-element.min.js'
//MOREMORE do the new look for the documentation page

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
   max-width: 6.6in;
}

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
  border: 2px solid var(--dw-item-border);
  border-radius: 0.4em;
  background-color: var(--dw-item-background-fill);

  text-decoration: none;
  color: black;
}

ul > li > a:hover {
  background-color: var(--dw-item-background-hover);
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
