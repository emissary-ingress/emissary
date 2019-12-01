import { LitElement, html, css } from '../vendor/lit-element.min.js'
import { Snapshot } from './snapshot.js'
import { License } from './license.js'

export class Support extends LitElement {

  static get properties() {
    return {
      licenseClaims: { type: Object },
      enabledFeatures: { type: Object },
      licenseMetadata: { type: Object }
    };
  }

  constructor() {
    super();

    Snapshot.subscribe(this.onSnapshotChange.bind(this));
  }

  onSnapshotChange(snapshot) {
    this.licenseClaims = snapshot.getLicense().Claims || {};
    this.enabledFeatures = this.licenseClaims.enabled_features || [];
    this.licenseMetadata = this.licenseClaims.metadata || {};
  }

  hasTicketSupport() {
    return this.enabledFeatures.includes(License._BUSINESS_SUPPORT)
      || this.enabledFeatures.includes(License._24X7_SUPPORT);
  }

  hasEmailSupport() {
    return this.enabledFeatures.includes(License._BUSINESS_SUPPORT)
      || this.enabledFeatures.includes(License._24X7_SUPPORT);
  }

  hasPhoneSupport() {
    return this.enabledFeatures.includes(License._24X7_SUPPORT);
  }

  hasOldLicense() {
    return this.licenseClaims.license_key_version !== License._LATEST_LICENSE_KEY_VERSION;
  }

  slackLink() {
    if (this.licenseMetadata.support_slack_link) {
      return this.licenseMetadata.support_slack_link;
    }
    return "http://d6e.co/slack";
  }

  phoneSupportNumber() {
    if (this.licenseMetadata.support_phone_number) {
      return this.licenseMetadata.support_phone_number;
    }
    return "";
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
  width: 1.7in;
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
  margin-top: 1em;
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
        <li><a href="${this.slackLink()}" target="_blank">
          <img src="../images/logos/slack-mark.svg" alt=""/>
          <span>Ask for help on Slack</span>
        </a></li>

        <li><a href="https://github.com/datawire/ambassador/issues/new/choose" target="_blank">
          <img src="../images/logos/github-mark.png" alt=""/>
          <span>Found a bug or have a feature request?<br/>File an issue.</span>
        </a></li>

        ${this.hasTicketSupport()
          ? html`<li><a href="https://support.datawire.io" target="_blank">
              <img src="../images/logos/datawire-mark.png" alt=""/>
              <span>Enterprise Support</span>
            </a></li>`
          : html `<li><a href="https://www.getambassador.io/contact" target="_blank">
              <img src="../images/logos/datawire-mark.png" alt=""/>
              <span>Contact Ambassador</span>
            </a></li>`
        }

        ${this.hasEmailSupport()
          ? html`<li><a href="mailto:support@datawire.io" target="_blank">
              <img src="../images/logos/email-mark.png" alt=""/>
              <span>support@datawire.io</span>
            </a></li>`
          : html ``
        }

        ${this.hasPhoneSupport()
          ? html`<li><a href="#">
              <img src="/edge_stack/images/logos/phone-mark.png" alt=""/>
              <span>Severity 1<br/>24x7 Support<br/><br/>${this.phoneSupportNumber()}</span>
            </a></li>`
          : html ``
        }
        
       </ul>
       
       ${((this.licenseClaims.customer_id != null) && (this.licenseClaims.customer_id !== License._UNREGISTERED_CUSTOMER_ID)) 
         ? html`<div>
                  Ambassador is licensed to <code>${this.licenseClaims.customer_email || this.licenseClaims.customer_id}</code>
                  and is valid until <code>${new Date(this.licenseClaims.exp * 1000).toISOString()}</code><br/>
                  ${this.hasOldLicense() ? html`You are running a older-generation license. Please contact Support for an upgraded Ambassador Edge Stack license key.` : html``}
                </div>`
         : html`<div>Ambassador is running unlicensed</div>`
       }
      </div>
    `;
  }
}

customElements.define('dw-support', Support);
