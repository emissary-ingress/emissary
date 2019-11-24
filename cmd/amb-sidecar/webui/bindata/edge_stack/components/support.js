import { LitElement, html, css } from 'https://cdn.pika.dev/-/lit-element/2.2.1/dist-es2019/lit-element.min.js'

export class Support extends LitElement {
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

        <li><a href="http://d6e.co/slack" target="_blank">
          <img src="/edge_stack/images/logos/slack-mark.svg" alt=""/>
          <span>Slack: go here to ask for help</span>
        </a></li>

        <li><a href="https://github.com/datawire/ambassador/issues/new/choose" target="_blank">
          <img src="/edge_stack/images/logos/github-mark.png" alt=""/>
          <span>Found a bug or have a feature request?</span>
        </a></li>

        <li><a href="https://www.getambassador.io/contact" target="_blank">
          <img src="/edge_stack/images/logos/datawire-mark.png" alt=""/>
          <span>Enterprise Support</span>
        </a></li>

      </ul>
    `;
  }
}
customElements.define('dw-support', Support);
