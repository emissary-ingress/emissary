import {html} from '../vendor/lit-element.min.js'
import {SingleResource, ResourceSet} from './resources.js';
import {getCookie} from './cookies.js';
import {ApiFetch} from "./api-fetch.js";

/**
 * A SingleHost is the UI-side object for a "getambassador.io/v2 Host" resource.
 */
export class SingleHost extends SingleResource {

  constructor() {
    super();
    this.tos = html`...`;
  }

  init() {
    /*
     * The Host object has an extra UI state of showing or not showing the Terms of Service checkbox.
     * Once the user has agreed to the Terms of Service, we no longer show the checkbox (and link)
     * in the Host detail display.
     */
    this.state.show_tos = false;
  }

  spec() {
    return {
      hostname: this.hostname().value,
      acmeProvider: this.useAcme()
        ? { authority: this.provider().value, email: this.email().value }
        : { authority: "none" }
    };
  }

  reset() {
    super.reset();
    /*
     * A Host has three fields in the spec, so these need
     * to be initialized whenever this UI-object is reset.
     */
    let fields = [this.provider(),
                  this.email(),
                  this.hostname()];
    fields.forEach(x=>x.value = x.defaultValue);
    /*
     * A Host also has UI-only state (not stored in the resource)
     * of whether the Terms of Service have been agreed to or not.
     */
    this.tos_agree().checked = false;
    this.state.show_tos = false;
    this.hostnameChanged();
  }

  validate() {
    super.validate();
    /*
     * We validate that the user has agreed to the Terms of Service,
     * which is either: (i) if we are not showing the Terms of Service,
     * then we assume that they have already agreed, or (ii) if we are
     * showing the TOS, then the checkbox needs to be checked.
     */
    if (this.useAcme()) {
      if (this.isTOSshowing() && !this.tos_agree().checked) {
        this.state.messages.push("You must agree to terms of service")
      }
      if (!this.email().checkValidity()) {
        this.state.messages.push("That doesn't look like a valid email address")
      }
    }
  }

  hostname() {
    return this.shadowRoot.querySelector('input[name="hostname"]')
  }

  use_acme_element() {
    return this.shadowRoot.querySelector('input[name="use_acme"]')
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
    this.hostnameChanged();
    this.providerChanged(false)
  }

  useAcme() {
    return (this.use_acme_element()||{checked:false}).checked;
  }

  hostnameChanged() {
    /*
     * This is called when the hostname field changes in an Edit or Add
     * dialog to check if the new hostname can be used with ACME.
     * If it can be, we check the checkbox, otherwise we uncheck it.
     */
    let url = new URL('/edge_stack/api/acme-host-qualifies', window.location);
    url.searchParams.set('hostname', this.hostname().value);
    ApiFetch(url, {
      headers: new Headers({
        'Authorization': 'Bearer ' + getCookie("edge_stack_auth")
      })
    })
      .then(r=>{
        r.json().then(qualifies=>{
          if( this.resource.spec.acmeProvider.authority === "none") {
            this.use_acme_element().checked = false; // if the spec says "none", then the user has explicitly said "no" so don't re-check the box
          } else {
            this.use_acme_element().checked = qualifies; // if the spec is an ACME provider (not "none") and the hostname qualifies, then check the box
          }
        })
      })
  }

  providerChanged(userEdit) {
    this.requestUpdate();
    if (userEdit) {
      this.state.show_tos = true
    }
    /*
     * Here we get the Terms of Service url from the ACME provider
     * so that we can show it to the user. We do this by calling
     * an API on AES that then turns around and calls an API on
     * the ACME provider. We cannot call the API on the ACME provider
     * directly due to CORS restrictions.
     */
    let value = this.provider().value;
    let url = new URL('/edge_stack/api/tos-url', window.location);
    url.searchParams.set('ca-url', value);
    ApiFetch(url, {
      headers: new Headers({
        'Authorization': 'Bearer ' + getCookie("edge_stack_auth")
      })
    })
      .then(r=>{
        r.text().then(t=>{
          if (r.ok) {
            let domain_matcher = /\/\/([^\/]*)\//;
            let d = t.match(domain_matcher);
            if(d) { d = d[1]; } else { d = t; }
            this.tos = html`<a href="${t}" target="_blank">${d}</a>`
            this.requestUpdate();
          } else {
            console.error("tos-url result: " + t);
            this.tos = html`....`
          }
        })
      })
  }

  kind() {
    return "Host"
  }

  isTOSshowing() {
    return (this.state.show_tos || this.state.mode === "add") && this.useAcme();
  }

  renderResource() {
    let host = this.resource;
    let spec = host.spec;
    let status = host.status || {"state": "<none>"};
    let hostState = status.state;
    let reason = (hostState === "Error") ? `(${status.errorReason})` : '';

    let state = this.state;
    let tos = this.isTOSshowing() ? "attribute-value" : "off";
    let editing = state.mode === "add" || state.mode === "edit";

    return html`
<div class="row line">
  <div class="row-col margin-right justify-right">hostname:</div>
  <div class="row-col">
    <span class="${this.visible("list")}">${spec.hostname}</span>
    <input class="${this.visible("edit", "add")}" type="text" name="hostname"  value="${spec.hostname}" @change="${this.hostnameChanged.bind(this)}"/>
  </div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right"></div>
  <div class="row-col">
    <input type="checkbox"
      name="use_acme"
      ?disabled="${!editing}"
      ?checked="${spec.acmeProvider.authority !== "none"}"
    /> Use ACME to manage TLS</label>
  </div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right">acme provider:</div>
  <div class="row-col">
    <span class="${this.visible("list")}">${spec.acmeProvider.authority}</span>
    <input
        class="${this.visible("edit", "add")}"
        type="url"
        size="60"
        name="provider"
        value="${spec.acmeProvider.authority}"
        @change=${()=>this.providerChanged(true)}
        ?disabled="${!this.useAcme()}"
      />
  </div>
</div>

<div class="${tos} row line">
  <div class="row-col margin-right justify-right"></div>
  <div class="row-col">
    <input type="checkbox" name="tos_agree" ?disabled="${!this.useAcme()}" />
      <span>I have agreed to to the Terms of Service: ${this.tos}</span>
  </div>
</div>

<div class="row ${this.state.mode !== "add" ? "line" : ""}">
  <div class="row-col margin-right justify-right">email:</div>
  <div class="row-col">
    <span class="${this.visible("list")}">${spec.acmeProvider.email}</span>
    <input class="${this.visible("edit", "add")}" type="email" name="email" value="${spec.acmeProvider.email}" ?disabled="${!this.useAcme()}" />
  </div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right ${this.visible("list", "edit")}">status:</div>
  <div class="row-col">
    <span class="${this.visible("list", "edit")}">${hostState} ${reason}</span>
  </div>
</div>

`
  }

}

customElements.define('dw-host', SingleHost);

export class Hosts extends ResourceSet {

  constructor() {
    super();
    this.addIfNone = true
  }

  getResources(snapshot) {
    let ret = snapshot.getResources("Host");
    if (this.addIfNone) {
      this.addState.mode = (ret.length < 1) ? "add" : "off";
      this.addIfNone = false;
    }
    return ret;
  }

  renderInner() {
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
      status: {}};
    return html`
<div class="header_con">
  <div class="col">
    <img alt="hosts logo" class="logo" src="../images/svgs/hosts.svg"> 
      <defs><style>.cls-1{fill:#fff;}</style></defs>
        <g id="Layer_2" data-name="Layer 2">
          <g id="Layer_1-2" data-name="Layer 1"></g>
        </g>
    </img>
  </div>
  <div class="col">
    <h1>Hosts</h1>
    <p>Hosts are domains that are managed by Ambassador Edge Stack, e.g., example.org</p>
  </div>
  <div class="col2">
    <a class="cta add ${this.readOnly() ? "off" : ""}" @click=${()=>this.shadowRoot.getElementById("add-host").onAdd()}>
      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 30 30"><defs><style>.cls-a{fill:none;stroke:#000;stroke-linecap:square;stroke-miterlimit:10;stroke-width:2px;}</style></defs><title>add_1</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><line class="cls-a" x1="15" y1="9" x2="15" y2="21"/><line class="cls-a" x1="9" y1="15" x2="21" y2="15"/><circle class="cls-a" cx="15" cy="15" r="14"/></g></g></svg>
      <div class="label">add</div>
    </a>
  </div>
</div>
<dw-host id="add-host" .resource=${addHost} .state=${this.addState}></dw-host>
<div>
  ${this.resources.map(h => html`<dw-host .resource=${h} .state=${this.state(h)}></dw-host>`)}
</div>`
  }

}

customElements.define('dw-hosts', Hosts);
