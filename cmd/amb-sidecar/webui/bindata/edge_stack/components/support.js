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
          : html `
           <li>
             <a href="https://www.getambassador.io/contact" target="_blank">
              <img src="../images/logos/datawire-mark.png" alt=""/>
              <span>Contact Ambassador</span>
             </a>
            </li>`
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
         ? html`<div style="text-align: center;">
                  Licensed to ${this.licenseClaims.customer_email || this.licenseClaims.customer_id}
                  through ${new Date(this.licenseClaims.exp * 1000).toLocaleDateString()}
                  ${this.hasOldLicense() ? html`<p style="font-size:90%; color:gray;">You have a older-generation license. Please contact Support for<br/>an upgraded Ambassador Edge Stack license key.</p>` : html``}
                </div>`
         : html`<div style="text-align: center;">Running in evaluation mode. Use the dashboard to get a <a href="#dashboard">free Community license</a>.</div>`
       }
      </div>
    `;
  }
}

customElements.define('dw-support', Support);
