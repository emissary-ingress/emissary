import {html} from '/edge_stack/vendor/lit-element.min.js'
import {SingleResource, ResourceSet} from '/edge_stack/components/resources.js';
import {getCookie} from '/edge_stack/components/cookies.js';

/**
 * A SingleHost is the UI-side object for a "getambassador.io/v2 Host" resource.
 */
export class SingleHost extends SingleResource {

  constructor() {
    super();
    this.tos = html`...`
  }

  init() {
    /*
     * The Host object has an extra UI state of showing or not showing the Terms of Service checkbox.
     * Once the user has agreed to the Terms of Service, we no longer show the checkbox (and link)
     * in the Host detail display.
     */
    this.state.show_tos = false
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
    this.state.show_tos = false
  }

  validate() {
    /*
     * We validate that the user has agreed to the Terms of Service,
     * which is either: (i) if we are not showing the Terms of Service,
     * then we assume that they have already agreed, or (ii) if we are
     * showing the TOS, then the checkbox needs to be checked.
     */
    if (this.state.show_tos && !this.tos_agree().checked) {
      this.state.messages.push("You must agree to terms of service")
    }
    /*
     * We validate that the user has provided a plausible looking
     * email address. In the future, we should actually validate that
     * it's a real email address using something like
     * https://www.textmagic.com/free-tools/email-validation-tool
     * with an appropriate fallback if we are unable to reach
     * outside the firewall (if we can't reach the outside system,
     * then use simple pattern matching).
     */
    var emailFormat = /^\w+([.-]?\w+)*@\w+([.-]?\w+)*(.\w{2,3})+$/;
    if (!this.email().value.match(emailFormat)) {
      this.state.messages.push("That doesn't look like a valid email address")
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
    fetch(url, {
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
    let host = this.resource;
    let spec = host.spec;
    let status = host.status || {"state": "<none>"};
    let hostState = status.state;
    let reason = (hostState === "Error") ? `(${status.reason})` : '';

    let state = this.state;
    let tos = state.show_tos || state.mode === "add" ? "attribute-value" : "off";

    return html`
  <div class="attribute-name">hostname:</div>
  <div class="attribute-value">
    <span class="${this.visible("list")}">${spec.hostname}</span>
    <input class="${this.visible("edit", "add")}" type="text" name="hostname"  value="${spec.hostname}" />
  </div>

  <div class="attribute-name">acme provider:</div>
  <div class="attribute-value">
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

  <div class="attribute-name">email:</div>
  <div class="attribute-value">
    <span class="${this.visible("list")}">${spec.acmeProvider.email}</span>
    <input class="${this.visible("edit", "add")}" type="email" name="email" value="${spec.acmeProvider.email}" />
  </div>

  <div class="attribute-name ${this.visible("list", "edit")}">status:</div>
  <div class="attribute-value ${this.visible("list", "edit")}">
    <span>${hostState} ${reason}</span>
  </div>
`
  }

}

customElements.define('dw-host', SingleHost);

export class Hosts extends ResourceSet {

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
      status: {}};
    return html`
<dw-host .resource=${addHost} .state=${this.addState}><add-button></add-button></dw-host>
<div>
  ${this.resources.map(h => html`<dw-host .resource=${h} .state=${this.state(h)}></dw-host>`)}
</div>`
  }

}

customElements.define('dw-hosts', Hosts);
