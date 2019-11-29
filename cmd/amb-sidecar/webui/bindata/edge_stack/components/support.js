import { LitElement, html, css } from '../vendor/lit-element.min.js'
import { Snapshot } from './snapshot.js'

export class Support extends LitElement {

  static get properties() {
    return {
      licenseClaims: { type: Object },
      enabledFeatures: { type: Object }
    };
  }

  constructor() {
    super();

    Snapshot.subscribe(this.onSnapshotChange.bind(this));
  }

  onSnapshotChange(snapshot) {
    this.licenseClaims = snapshot.getLicense().Claims || {};
    this.enabledFeatures = this.licenseClaims.enabled_features || [];
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
      <div class="center">
       <ul>
        <li><a href="http://d6e.co/slack" target="_blank">
          <img src="../images/logos/slack-mark.svg" alt=""/>
          <span>Ask for help on Slack</span>
        </a></li>

        <li><a href="https://github.com/datawire/ambassador/issues/new/choose" target="_blank">
          <img src="../images/logos/github-mark.png" alt=""/>
          <span>Found a bug or have a feature request?<br/>File an issue.</span>
        </a></li>

        ${this.enabledFeatures.includes("support-business-tier")
            || this.enabledFeatures.includes("support-24x7-tier")
          ? html`<li><a href="https://support.datawire.io" target="_blank">
              <img src="../images/logos/datawire-mark.png" alt=""/>
              <span>Enterprise Support</span>
            </a></li>`
          : html `<li><a href="https://www.getambassador.io/contact" target="_blank">
              <img src="../images/logos/datawire-mark.png" alt=""/>
              <span>Contact Ambassador</span>
            </a></li>`
        }

        ${this.enabledFeatures.includes("support-business-tier")
            || this.enabledFeatures.includes("support-24x7-tier")
          ? html`<li><a href="mailto:support@datawire.io" target="_blank">
              <img src="../images/logos/email-mark.png" alt=""/>
              <span>support@datawire.io</span>
            </a></li>`
          : html ``
        }
        
       </ul>
       
       ${this.licenseClaims.customer_id != "unregistered" 
         ? html`<div>Ambassador is licensed to ${this.licenseClaims.customer_email || this.licenseClaims.customer_id}</div>`
         : html`<div>Ambassador is running unlicensed</div>`
       }
      </div>
    `;
  }
}
customElements.define('dw-support', Support);
