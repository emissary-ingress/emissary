import { LitElement, html, css } from 'https://cdn.pika.dev/-/lit-element/2.2.1/dist-es2019/lit-element.min.js'
import { repeat } from '/edge_stack/components/repeat.js';
import {registerContextChangeHandler, useContext} from '/edge_stack/components/context.js';

export class HostAdd extends LitElement {

  static get properties() {
    return {
      state: { type: String },
      tos: { type: String },
      messages: { type: Array }
    }
  }

  static get styles() {
    return css`
#form.start { display: none; }
#add.entry { display: none; }

div {
  margin: 0.5em;
}

.error {
  color: red;  
}
`
  }

  constructor() {
    super()
    this.state = 'start'
    this.tos = html`...`
    this.messages = []
  }

  firstUpdated(e) {
    this.providerChanged()
  }

  providerChanged() {
    let value = this.provider().value
    let url = new URL('/edge_stack/api/tos-url', window.location)
    url.searchParams.set('ca-url', value)
    fetch(url, {
      headers: new Headers({
        'Authorization': 'Bearer ' + window.location.hash.slice(1)
      })
    })
      .then(r=>{
        r.text().then(t=>{
          if (r.ok) {
            this.tos = html`<a href="${t}">${t}</a>`
          } else {
            this.tos = html`...`
          }
        })
      })
  }

  handleStart() {
    this.state = "entry"
  }

  handleCancel() {
    this.state = "start"
  }

  handleAdd() {
    this.messages = []

    if (!this.tos_agree().checked) {
      this.messages.push("you must agree to terms of service")
    }

    var emailFormat = /^\w+([.-]?\w+)*@\w+([.-]?\w+)*(.\w{2,3})+$/

    if (!this.email().value.match(emailFormat)) {
      this.messages.push("you must supply a valid email address")
    }

    if (this.messages.length > 0) {
      return
    }

    let yaml = `
---
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: "${this.hostname().value}"
  namespace: default
spec:
  hostname: "${this.hostname().value}"
  acmeProvider:
    authority: "${this.provider().value}"
    email: "${this.email().value}"
`
    fetch('/edge_stack/api/apply',
          {
            method: "POST",
            headers: new Headers({
              'Authorization': 'Bearer ' + window.location.hash.slice(1)
            }),
            body: yaml
          })
      .then(r=>{
        r.text().then(t=>{
          if (r.ok) {
            alert("OK\n" + t)
          } else {
            alert("BAD\n\n" + yaml + "\n\n" + t)
          }
          this.state = "start"
        })
      })
  }

  hostname() {
    return this.shadowRoot.querySelector('input[name="hostname"]')
  }

  provider() {
    return this.shadowRoot.querySelector('input[name="provider"]')
  }

  tos_agree() {
    return this.shadowRoot.querySelector('input[name="tos_agree"]')
  }

  email() {
    return this.shadowRoot.querySelector('input[name="email"]')
  }

  renderErrors() {
    if (this.messages.length > 0) {
      return html`<label>
  <ul>
    ${this.messages.map(m=>html`<li><span class="error">${m}</span></li>`)}
  </ul>
</label>`
    } else {
      return html``
    }
  }

  render() {
    return html`
<slot class="${this.state}" id="add" @click=${this.handleStart}></slot>
<div class="${this.state}" id="form">
  <fieldset>
    <legend>Add a Host</legend>
    <label>
      <span>Hostname:</span>
      <input type="text" name="hostname"  value="${window.location.hostname}" />
    </label>
    <label>
      <span>ACME provider:</span>
      <input type="url" name="provider" @change=${this.providerChanged} value="https://acme-v02.api.letsencrypt.org/directory" />
    </label>
    <div>
      <label>
        <input type="checkbox" name="tos_agree" />
        <span>I have agreed to to the Terms of Service at: ${this.tos}</span>
      </label>
    </div>
    <label>
      <span>Email:</span>
      <input type="email" name="email" />
    </label>
    <div>
      <label>
        <button @click=${this.handleCancel}>Cancel</button>
      </label>
      <label>
        <button @click=${this.handleAdd}>Add</button>
      </label>
    </div>
    ${this.renderErrors()}
  </fieldset>
</div>`
  }
}

customElements.define('dw-host-add', HostAdd)

// XXX: dw-hosts is not used yet!

export default class Hosts extends LitElement {
  static get properties() {
    return {
      hosts: Array
    };
  }

  constructor() {
    super();

    const arr = useContext('aes-api-snapshot', null);
    if (arr[0] != null) {
      this.hosts = arr[0]['Host'] || [];
    } else {
      this.hosts = [];
    }
    registerContextChangeHandler('aes-api-snapshot', this.onSnapshotChange.bind(this));
  }

  onSnapshotChange(snapshot) {
    let kube = snapshot['Kubernetes'] || {'Host': []}
    this.hosts = kube['Host'] || [];
  }

  renderHost(host) {
    let name = host.spec.hostname
    let state = host.status.state
    let reason = state == "Error" ? `(${host.status.reason})` : ''
    return html`<div>${name}: ${state} ${reason}</div>`
  }

  render() {
    return html`<div>${this.hosts.map(h => this.renderHost(h))}</div>`
  }

}

customElements.define('dw-hosts', Hosts);
