import { LitElement, html, css } from 'https://cdn.pika.dev/-/lit-element/2.2.1/dist-es2019/lit-element.min.js'
import {registerContextChangeHandler, useContext} from '/edge_stack/components/context.js'

// holds the UI state of a host widget, this can be merged with Host when we have repeat
class HostUIState {

  constructor() {
    this.mode = "list" // one of add, edit, list, off
    this.messages = []
    this.show_tos = false
  }

  renderErrors() {
    if (this.messages.length > 0) {
      return html`
<div class="both">
  <ul>
    ${this.messages.map(m=>html`<li><span class="error">${m}</span></li>`)}
  </ul>
</div>`
    } else {
      return html``
    }
  }
}

export class Host extends LitElement {

  static get styles() {
    return css`
.error {
  color: red;
}

div {
  margin: 0.4em;
}

div.frame {
  display: grid;
  grid-template-columns: max-content;
  border: 2px solid #ede7f3;
  border-radius: 0.4em;
}

div.title {
  grid-column: 1 / 3;
  background: #ede7f3;
  margin: 0;
  padding: 0.5em;
}

div.left {
  grid-column: 1 / 2;
}

div.right {
  grid-column: 2 / 3;
}

div.both {
  grid-column: 1 / 3;
}

.off { display: none; }
`
  }

  static get properties() {
    return {
      host: {type: Map},
      state: {type: Object},
    }
  }

  constructor() {
    super()
    this.host = {}
    this.state = {}
    this.tos = html`...`
  }

  onAdd() {
    this.state.mode = "add"
    this.requestUpdate()
  }

  onEdit(host) {
    if (this.state.mode != "edit") {
      this.state.mode = "edit"
    } else {
      this.state.mode = "list"
    }
    this.requestUpdate()
  }

  onDelete(host) {
    alert("TODO")
  }

  onCancel(host) {
    this.requestUpdate()

    if (this.state.mode == "add") {
      this.state.mode = "off"
    } else {
      this.state.mode = "list"
    }

    this.reset()
  }

  reset() {
    this.state.messages.length = 0
    let fields = [this.provider(), this.email(), this.hostname()]
    fields.forEach(x=>x.value = x.defaultValue)
    this.tos_agree().checked = false
    this.state.show_tos = false
  }

  onSave(host) {
    this.requestUpdate()

    this.state.messages.length = 0
    if (!this.tos_agree().checked) {
      this.state.messages.push("you must agree to terms of service")
    }

    var emailFormat = /^\w+([.-]?\w+)*@\w+([.-]?\w+)*(.\w{2,3})+$/

    if (!this.email().value.match(emailFormat)) {
      this.state.messages.push("you must supply a valid email address")
    }

    if (this.state.messages.length > 0) {
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
          if (this.state.mode == "add") {
            this.state.mode = "off"
          } else {
            this.state.mode = "list"
          }
          this.reset()
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

  firstUpdated(e) {
    this.providerChanged(false)
  }

  providerChanged(userEdit) {
    this.requestUpdate()
    if (userEdit) {
      this.state.show_tos = true
    }
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

  render() {
    let host = this.host
    let spec = host.spec
    let status = host.status.state
    let reason = status == "Error" ? `(${host.status.reason})` : ''

    let state = this.state

    let add = state.mode == "add" ? "" : "off"
    let list = state.mode == "list" ? "" : "off"
    let edit = state.mode == "edit" || state.mode == "add" ? "" : "off"
    let list_or_edit = state.mode == "list" || state.mode == "edit" ? "" : "off"
    let tos = this.state.show_tos || state.mode == "add" ? "right" : "off"

    return html`
<slot class="${state.mode == "off" ? "" : "off"}" @click=${this.onAdd.bind(this)}></slot>
<div class="${state.mode == "off" ? "off" : "frame"}">
  <div class="title">
    Host: <span class="${list_or_edit}">${host.metadata.name}</span>
          <input class="${add}" type="text" value="${host.metadata.name}"/>


      (<span class="${list_or_edit}">${host.metadata.namespace}</span>
       <input class="${add}" type="text" value="${host.metadata.namespace}"/>)</div>

  <div class="left">Hostname:</div>
  <div class="right">
    <span class="${list}">${spec.hostname}</span>
    <input class="${edit}" type="text" name="hostname"  value="${spec.hostname}" />
  </div>

  <div class="left">ACME Provider:</div>
  <div class="right">
    <span class="${list}">${spec.acmeProvider.authority}</span>
    <input
      class="${edit}"
      type="url"
      size="60"
      name="provider"
      value="${spec.acmeProvider.authority}"
      @change=${()=>this.providerChanged(true)}
    />
  </div>

  <div class="${tos}">
    <input type="checkbox" name="tos_agree" />
    <span>I have agreed to to the Terms of Service at: ${this.tos}</span>
  </div>

  <div class="left">Email:</div>
  <div class="right">
    <span class="${list}">${spec.acmeProvider.email}</span>
    <input class="${edit}" type="email" name="email" value="${spec.acmeProvider.email}" />
  </div>

  <div class="left ${list_or_edit}">Status:</div>
  <div class="right ${list_or_edit}">
    <span>${status} ${reason}</span>
  </div>

  <div class="both">
    <label>
      <button class="${list}" @click=${()=>this.onEdit(host)}>Edit</button>
      <button class="${list}" @click=${()=>this.onDelete(host)}>Delete</button>
      <button class="${edit}" @click=${()=>this.onCancel(host)}>Cancel</button>
      <button class="${edit}" @click=${(e)=>this.onSave(host, e)}>Save</button>
    </label>
  </div>

  ${this.state.renderErrors()}
</div>`
  }

}

customElements.define('dw-host', Host)

export default class Hosts extends LitElement {

  static get properties() {
    return {
      hosts: {type: Array},
      _states: {type: Map},
      addState: {type: Object}
    };
  }

  constructor() {
    super();

    const arr = useContext('aes-api-snapshot', null);
    if (arr[0] != null) {
      this.hosts = arr[0]['Host'] || []
    } else {
      this.hosts = []
    }
    this._states = {}
    this.addState = new HostUIState()
    this.addState.mode = "off"
    registerContextChangeHandler('aes-api-snapshot', this.onSnapshotChange.bind(this))
  }

  onSnapshotChange(snapshot) {
    let kube = snapshot['Kubernetes'] || {'Host': []}
    this.hosts = kube['Host'] || []
  }

  state(host) {
    let key = host.metadata.namespace + ":" + host.metadata.name
    if (this._states[key] == undefined) {
      this._states[key] = new HostUIState()
    }
    return this._states[key]
  }

  render() {
    let addHost = {
      metadata: {
        namespace: "default",
        name: window.location.hostname
      },
      spec: {
        hostname: window.location.hostname,
        acmeProvider: {
          authority: "https://acme-v02.api.letsencrypt.org/directory",
          email: ""
        }
      },
      status: {}}
    return html`
<dw-host .host=${addHost} .state=${this.addState}><button>Add</button></dw-host>
<div>
  ${this.hosts.map(h => html`<dw-host .host=${h} .state=${this.state(h)}></dw-host>`)}
</div>`
  }

}

customElements.define('dw-hosts', Hosts)
