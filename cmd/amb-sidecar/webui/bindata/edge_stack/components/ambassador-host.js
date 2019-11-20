import  {LitElement, html, css} from 'https://cdn.pika.dev/-/lit-element/2.2.1/dist-es2019/lit-element.min.js';
import {getCookie} from '/edge_stack/components/cookies.js';

export class AmbassadorHost extends LitElement {
  static get properties() {
    return {
      hostname: String,
      hasChangedHostname: Boolean,
      provider: String,
      hasAgreedToTOS: Boolean,
      isFetchingTos: Boolean,
      failedTOS: Boolean,
      tosUrl: String,
      email: String,
      yaml: String,
      output: String
    };
  }

  static get styles() {
    return css`
    `;
  }

  constructor() {
    super();

    this.hostname = window.location.hostname;
    this.hasChangedHostname = false;
    this.provider = 'https://acme-v02.api.letsencrypt.org/directory';
    this.tosUrl = '';

    this.hasAgreedToTOS = false;
    this.isFetchingTos = false;
    this.failedTOS = false;

    this.email = '';
    this.yaml = '';
    this.output = '';
  }

  onMount() {
    this.fetchNewTosUrl();
  }

  onUnmount() {}

  fetchNewTosUrl() {
    let url = new URL('tos-url', window.location);
    url.searchParams.set('ca-url', this.provider);

    this.isFetchingTos = true;
    fetch(url, {
      headers: {
        'Authorization': 'Bearer ' + getCookie("edge_stack_auth")
      }
    }).then((data) => {
      if (data.status == 200) {
        data.body.then((data) => {
          this.tosUrl = data;
          this.failedTOS = false;
          this.hasAgreedToTOS = false;
        }).catch((err) => {
          console.log('error', err);
          this.tosUrl = '';
          this.failedTOS = true;
        });
      } else {
        console.log('Error', data.status);
        this.tosURL = '';
        this.failedTOS = true;
      }
      this.isFetchingTos = false;
    }).catch((err) => {
      console.log('error', err);
      this.failedTOS = true;
      this.isFetchingTos = false;
    });
  }

  onHostnameChange(evt) {
    this.hostname = evt.target.value;
    this.hasChangedHostname = true;
  }

  onProviderChange(evt) {
    this.provider = evt.target.value;
    this.fetchNewTosUrl();
  }

  agreeToTOS() {
    this.hasAgreedToTOS = true;
  }

  handleOutput(str) {
    this.output += new Date();
		this.output += str;
		this.output += "\n";
		this.lastOutput = str;
		if (str == "state: Ready\n") {
			window.location = `https://${this.hostname}/ambassador-edge-stack/admin#<jwt>`;
		}
  }

  updateYaml() {
    let url = new URL('yaml', window.location);
    url.searchParams.set('hostname', this.hostname);
		url.searchParams.set('acme_authority', this.provider);
    url.searchParams.set('acme_email', this.email);

    fetch(url.toString(), {
      headers: {
        'Authorization': 'Bearer ' + getCookie("edge_stack_auth")
      }
    }).then((data) => {
      if (data.status == 200) {
        data.body.then((txt) => {
          this.yaml = txt;
        });
      }
    });
  }

  onEmailChange(evt) {
    this.email = evt.target.data;
    this.updateYaml();
  }

  refershAgain() {
    setTimeout(() => {
      this.refreshStatus();
    }, 1000/5);
  }

  refreshStatus() {
    let url = new URL('status', window.location);
    url.searchParams.set('hostname', this.hostname);

    fetch(url.toString(), {
      headers: {
        'Authorization': 'Bearer ' + getCookie("edge_stack_auth")
      }
    }).then((data) => {
      if (data.status == 200) {
        data.body.then((txt) => {
          this.handleOutput(txt);
          this.refershAgain();
        }).catch((err) => {
          console.log('Error fetching body', err);
          this.handleOutput('Error refreshing status. Trying Again.');
          this.refershAgain();
        });
      } else {
        this.handleOutput('Error refreshing status. Trying Again.');
        this.refershAgain();
      }
    }).catch((err) => {
      console.log('Error', err);
      this.handleOutput('Error refreshing status. Trying Again.');
      this.refershAgain();
    });
  }

  submitForm() {
    let url = new URL('yaml', window.location);
    url.searchParams.set('hostname', this.hostname);
		url.searchParams.set('acme_authority', this.provider);
    url.searchParams.set('acme_email', this.email);

    fetch(url.toString(), {
      method: 'POST',
      headers: {
        'Authorization': 'Bearer ' + getCookie("edge_stack_auth")
      }
    }).then((data) => {
      if (data.status == 201) {
        this.handleOutput('Applying YAML...');
        this.refreshStatus();
      } else {
        this.handleOutput('Error applying YAML!');
      }
    }).catch((error) => {
      console.log('Error applying: ', error);
    });
  }

  render() {
    return html`
<form @submit="${this.submitForm}">
  <fieldset>
    <legend>Initial TLS Setup</legend>
    <label>
      <span>Hostname to request a certificate for:</span>
      <div>
        <input type="text" name="hostname" value="${this.hostname}" @change="${this.onHostnameChange}" />
        ${this.hasChangedHostname ? html`` : html`<p>We auto-filled this from the URL you used to access this web page.  Feel free to change it, though.</p>`}
      </div>
    </label>
    <label>
      <span>ACME provider:</span>
      <input type="url" name="provider" value="${this.provider}" @change="${this.onProviderChange}" />
    </label>
    <label>
      <input type="checkbox" name="tos_agree" value="${this.hasAgreedToTOS}" @change="${this.agreeToTOS}" disabled="${(this.isFetchingTos || this.failedTOS)}">
      ${this.isAgreeingToTOS ? html`<p>Loading Spinner goes here</p>` : html``}
      ${this.failedTOS ? html`Error encoutered agreeing to TOS, see console, and refresh.` : html``}
      ${this.tosUrl != '' ? html`I have agreed to the Terms of Service at <a href="${this.tosUrl}">${this.tosUrl}.</a>` : html``}
    </label>
    <label>
      <span>Email:</span>
      <input type="email" name="email" value="${this.email}" @change="${this.onEmailChange}"></input>
    </label>
    <pre>
      ${this.yaml}
    </pre>
    <input type="submit" value="Apply" />
  </fieldset>
</form>
${this.output != '' ? html`<p>${this.output}</p>` : html``}
    `;
  }
}

customElements.define('ambassador-host', AmbassadorHost);
