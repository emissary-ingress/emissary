import {html} from 'https://cdn.pika.dev/-/lit-element/2.2.1/dist-es2019/lit-element.min.js'
import {SingleResource, ResourceSet} from '/edge_stack/components/resources.js';
import {getCookie} from '/edge_stack/components/cookies.js';

export class Host extends SingleResource {

  constructor() {
    super()
    this.tos = html`...`
  }

  init() {
    this.state.show_tos = false
  }

  reset() {
    super.reset()
    let fields = [this.provider(), this.email(), this.hostname()]
    fields.forEach(x=>x.value = x.defaultValue)
    this.tos_agree().checked = false
    this.state.show_tos = false
  }

  validate() {
    if (this.state.show_tos && !this.tos_agree().checked) {
      this.state.messages.push("you must agree to terms of service")
    }

    var emailFormat = /^\w+([.-]?\w+)*@\w+([.-]?\w+)*(.\w{2,3})+$/

    if (!this.email().value.match(emailFormat)) {
      this.state.messages.push("you must supply a valid email address")
    }
  }

  spec() {
    return {
      hostname: this.hostname().value,
      acmeProvider: {
        authority: this.provider().value,
        email: this.email().value
      }
    }
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
        'Authorization': 'Bearer ' + getCookie("edge_stack_auth")
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

  kind() {
    return "Host"
  }

  renderResource() {
    let host = this.resource
    let spec = host.spec
    let status = host.status || {"state": "<none>"}
    let hostState = status.state
    let reason = hostState == "Error" ? `(${status.reason})` : ''

    let state = this.state
    let tos = state.show_tos || state.mode == "add" ? "right" : "off"

    return html`
  <div class="left">Hostname:</div>
  <div class="right">
    <span class="${this.visible("list")}">${spec.hostname}</span>
    <input class="${this.visible("edit", "add")}" type="text" name="hostname"  value="${spec.hostname}" />
  </div>

  <div class="left">ACME Provider:</div>
  <div class="right">
    <span class="${this.visible("list")}">${spec.acmeProvider.authority}</span>
    <input
      class="${this.visible("edit", "add")}"
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
    <span class="${this.visible("list")}">${spec.acmeProvider.email}</span>
    <input class="${this.visible("edit", "add")}" type="email" name="email" value="${spec.acmeProvider.email}" />
  </div>

  <div class="left ${this.visible("list", "edit")}">Status:</div>
  <div class="right ${this.visible("list", "edit")}">
    <span>${hostState} ${reason}</span>
  </div>
`
  }

}

customElements.define('dw-host', Host)

export default class Hosts extends ResourceSet {

  key() {
    return "Host"
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
<dw-host .resource=${addHost} .state=${this.addState}><add-button></add-button></dw-host>
<div>
  ${this.resources.map(h => html`<dw-host .resource=${h} .state=${this.state(h)}></dw-host>`)}
</div>`
  }

}

customElements.define('dw-hosts', Hosts)
