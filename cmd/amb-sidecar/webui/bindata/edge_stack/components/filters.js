import {html} from '../vendor/lit-element.min.js'
import {SingleResource, SortableResourceSet} from './resources.js';

/**
 * The UI component for a filter.
 */
class Filter extends SingleResource {

  // internal
  constructor() {
    super();
    this.type = "OAuth2";
    this.subspec = null;
    this.OAuth2 = null;
    this.External = null;
    this.JWT = null;
    this.Plugin = null;
    this.Internal = null;
  }

  /**
   * Implement.
   */
  kind() {
    return "Filter"
  }

  /**
   * We need to customize the merge strategy here because the filter
   * CRD uses the presence/absence of keys as a descriminator to
   * indicate what type of filter the resource represents, e.g.:
   *
   *   spec:
   *     OAuth2: { ... }
   *
   *   spec:
   *     JWT: { ... }
   *
   * The above two specs are mutually exclusive and represent
   * different kinds of filter types.
   *
   * The UI surfaces this as a type field that you can select. If you
   * do select a new type, all the fields change to represent what is
   * allowed to go underneath spec.<new-type>.
   *
   * The net of this is that if you do not change the type of the
   * filter, then you want to use the merge strategy. If you *do*
   * change the type of the filter, then you need to use the replace
   * strategy because you don't want (and can't have) OAuth2 *and* JWT
   * properties at the same time.
   */
  mergeStrategy(path) {
    if (path === `spec.${this.filterType()}`) {
      if (this.type === this.filterType()) {
        return "merge";
      } else {
        return "replace";
      }
    }
  }

  /**
   * Implement.
   */
  spec() {
    let result = {};
    result[this.type] = this.subspec;
    return result;
  }

  // internal
  filterType() {
    let spec = this.resource.spec;
    if (spec.OAuth2) {
      return "OAuth2";
    }
    if (spec.External) {
      return "External";
    }
    if (spec.JWT) {
      return "JWT";
    }
    if (spec.Plugin) {
      return "Plugin";
    }
    if (spec.Internal) {
      return "Internal";
    }
    return "<unknown>";
  }

  // override
  onAdd() {
    super.onAdd();
    let type = this.filterType();
    this.subspec = this.resource.spec[type];
    this[type] = this.subspec;
  }

  // override
  onEdit() {
    super.onEdit();
    let type = this.filterType();
    this.subspec = this.resource.spec[type];
    this[type] = this.subspec;
  }

  // internal
  input(subspec, name, type="text") {
    let value = subspec[name];
    if (value === undefined) {
      value = "";
    }
    return html`
<visible-modes list>${value}</visible-modes>
<visible-modes add edit>
  <input type="${type}" .value="${value}"
         @change=${(e)=>{subspec[name]=e.target.value; this.requestUpdate();}}
  />
</visible-modes>
`
  }

  // internal
  disabled() {
    return this.state.mode !== "add" && this.state.mode !== "edit";
  }

  // internal
  bool(subspec, name) {
    return html`
  <input ?disabled=${this.disabled()} type="checkbox" .checked=${subspec[name]}
         @change=${(e)=>{subspec[name]=e.target.checked; this.requestUpdate(); console.log(subspec[name])}}
  />
`
  }

  // internal
  select(subspec, name, options) {
      return html`
<select ?disabled=${this.disabled()} @change=${(e)=>{subspec[name]=e.target.value; this.requestUpdate();}}>
  ${options.map((opt)=>html`<option .selected=${subspec[name] === opt} value="${opt}">${opt}</option>`)}
</select>
`
  }

  // internal
  duration(subspec, name) {
    // XXX: for now just use text input
    return this.input(subspec, name);
  }

  // internal
  renderOAuth2(subspec) {
    let grantTypes = ["AuthorizationCode", "ClientCredentials"];
    return html`
<div class="attribute-name">Authorization URL:</div>
<div class="attribute-value">${this.input(subspec, "authorizationURL", "url")}</div>

<div class="attribute-name">Grant Type:</div>
<div class="attribute-value">${this.select(subspec, "grantType", grantTypes)}</div>

<div class="attribute-name">Access Token Validation:</div>
<div class="attribute-value">${this.select(subspec, "accessTokenValidation", ["auto", "jwt", "userinfo"])}</div>

<hr>

<div class="attribute-name">Client URL:</div>
<div class="attribute-value">${this.input(subspec, "clientURL", "url")}</div>

<div class="attribute-name">Client ID:</div>
<div class="attribute-value">${this.input(subspec, "clientID")}</div>

<div class="attribute-name">State TTL:</div>
<div class="attribute-value">${this.input(subspec, "stateTTL", "number")}</div>

<div class="attribute-name">Secret:</div>
<div class="attribute-value">${this.input(subspec, "secret")}</div>

<div class="attribute-name">Secret Name:</div>
<div class="attribute-value">${this.input(subspec, "secretName")}</div>

<div class="attribute-name">Secret Namespace:</div>
<div class="attribute-value">${this.input(subspec, "secretNamespace")}</div>

<hr>

<div class="attribute-name">Insecure TLS:</div>
<div class="attribute-value">${this.bool(subspec, "insecureTLS")}</div>

<div class="attribute-name">Renegotiate TLS:</div>
<div class="attribute-value">${this.bool(subspec, "renegotiateTLS")}</div>

<div class="attribute-name">Max Stale:</div>
<div class="attribute-value">${this.duration(subspec, "maxStale")}</div>
`
  }

  // internal
  headerList(subspec, name) {
    let values = subspec[name];
    return html`
${(values || []).map((v, i)=>this.headerListEntry(values, v, i))}
<visible-modes add edit>
  <button @click=${()=>{
  if (values === null || values === undefined) {
    values = [];
    subspec[name] = values;
  }
  values.push(""); this.requestUpdate();
}}>+</button>
</visible-modes>
`;
  }

  // internal
  headerListEntry(values, value, index) {
    return html`
<div>
  <input ?disabled=${this.disabled()} .value="${value}"
         @change=${(e)=>{values[index] = e.target.value; this.requestUpdate()}}
  />
  <visible-modes add edit>
    <button @click=${()=>{values.splice(index, 1); this.requestUpdate()}}>-</button>
  </visible-modes>
</div>
`;
  }

  // internal
  renderExternal(subspec) {
    return html`
<div class="attribute-name">Auth Service:</div>
<div class="attribute-value">${this.input(subspec, "auth_service")}</div>

<div class="attribute-name">Path Prefix:</div>
<div class="attribute-value">${this.input(subspec, "path_prefix")}</div>

<div class="attribute-name">TLS:</div>
<div class="attribute-value">${this.bool(subspec, "tls")}</div>

<div class="attribute-name">Protocol:</div>
<div class="attribute-value">${this.select(subspec, "proto", ["http", "grpc"])}</div>

<div class="attribute-name">Allow Request Body:</div>
<div class="attribute-value">${this.bool(subspec, "allow_request_body")}</div>

<div class="attribute-name">Timeout:</div>
<div class="attribute-value">${this.input(subspec, "timeout_ms", "number")}</div>

<div class="attribute-name">Allowed Request Headers:</div>
<div class="attribute-value">${this.headerList(subspec, "allowed_request_headers")}</div>

<div class="attribute-name">Allowed Authorization Headers:</div>
<div class="attribute-value">${this.headerList(subspec, "allowed_authorization_headers")}</div>
`
  }

  // internal
  headerTemplateList(subspec, name) {
    let values = subspec[name];
    return html`
${(values || []).map((v, i)=>this.headerTemplateListEntry(values, v, i))}
<visible-modes add edit>
  <button @click=${()=>{
  if (values === null || values === undefined) {
    values = [];
    subspec[name] = values;
  }
  values.push({name: "", value: ""}); this.requestUpdate()}
}>+</button>
</visible-modes>
`;
  }

  // internal
  headerTemplateListEntry(values, entry, index) {
    return html`
<div>
  <input ?disabled=${this.disabled()} .value="${entry.name}"
         @change=${(e)=>{values[index].name = e.target.value; this.requestUpdate()}}
  />
  <input ?disabled=${this.disabled()} .value="${entry.value}"
         @change=${(e)=>{values[index].value = e.target.value; this.requestUpdate()}}
  />
  <visible-modes add edit>
    <button @click=${()=>{values.splice(index, 1); this.requestUpdate()}}>-</button>
  </visible-modes>
</div>
`;
  }

  // internal
  renderJWT(subspec) {
    let errorResponse = subspec.errorResponse;
    if (errorResponse === null || errorResponse === undefined) {
      errorResponse = {}
      subspec.errorResponse = errorResponse;
    }
    // XXX: the validAlgorithms field should probably be something
    // more custom, but for now we will use headerList
    return html`
<div class="attribute-name">Valid Algorithms:</div>
<div class="attribute-value">${this.headerList(subspec, "validAlgorithms")}</div>

<div class="attribute-name">JWKS URI:</div>
<div class="attribute-value">${this.input(subspec, "jwksURI", "url")}</div>

<div class="attribute-name">Audience:</div>
<div class="attribute-value">${this.input(subspec, "audience")}</div>

<div class="attribute-name">Require Audience:</div>
<div class="attribute-value">${this.bool(subspec, "requireAudience")}</div>

<div class="attribute-name">Issuer:</div>
<div class="attribute-value">${this.input(subspec, "issuer", "url")}</div>

<div class="attribute-name">Require Issuer:</div>
<div class="attribute-value">${this.bool(subspec, "requireIssuer")}</div>

<div class="attribute-name">Require Issued At:</div>
<div class="attribute-value">${this.bool(subspec, "requireIssuedAt")}</div>

<div class="attribute-name">Require Expires At:</div>
<div class="attribute-value">${this.bool(subspec, "requireExpiresAt")}</div>

<div class="attribute-name">Require Not Before:</div>
<div class="attribute-value">${this.bool(subspec, "requireNotBefore")}</div>

<div class="attribute-name">Inject Request Headers:</div>
<div class="attribute-value">${this.headerTemplateList(subspec, "injectRequestHeaders")}</div>

<div class="attribute-name">Insecure TLS:</div>
<div class="attribute-value">${this.bool(subspec, "insecureTLS")}</div>

<div class="attribute-name">Renegotiate TLS:</div>
<div class="attribute-value">${this.bool(subspec, "renegotiateTLS")}</div>

<div class="attribute-name">Error Headers:</div>
<div class="attribute-value">${this.headerTemplateList(errorResponse, "headers")}</div>

<div class="attribute-name">Error Body:</div>
<div class="attribute-value">
  <textarea ?disabled=${this.disabled()} rows="10" cols="60"
            @change=${(e)=>{errorResponse.bodyTemplate=e.target.value; this.requestUpdate()}}>
    ${errorResponse.bodyTemplate}
  </textarea>
</div>
`
  }

  // internal
  renderPlugin(subspec) {
    return html`
<div class="attribute-name">Name:</div>
<div class="attribute-value">${this.input(subspec, "name")}</div>
`
  }

  // internal
  renderInternal(subspec) {
  }

  // internal
  updateType(newType) {
    this[this.type] = this.subspec;
    if (this[newType] === null || this[newType] === undefined) {
      this[newType] = {}
    }
    this.subspec = this[newType];
    this.type = newType;
    this.requestUpdate();
  }

  /**
   * Implement.
   */
  renderResource() {
    let type = this.state.mode === "add" || this.state.mode == "edit" ? this.type : this.filterType();
    let render = this[`render${type}`].bind(this);
    let subspec = this.state.mode === "add" || this.state.mode == "edit" ? this.subspec : this.resource.spec[type];
    let rendered = render(subspec)
    return html`
<div class="attribute-name">Type:</div>
<div class="attribute-value">
  <select ?disabled=${this.disabled()} @change=${(e)=>this.updateType(e.target.value)}>
    <option .selected=${type === "OAuth2"} value="OAuth2">OAuth2</option>
    <option .selected=${type === "JWT"} value="JWT">JWT</option>
    <option .selected=${type === "External"} value="External">External</option>
    <option .selected=${type === "Plugin"} value="Plugin">Plugin</option>
  </select>
</div>
${rendered}
`
  }

}

customElements.define('dw-filter', Filter);

export class Filters extends SortableResourceSet {

  // implement
  constructor() {
    super([
      {value: "name", label: "Name"},
      {value: "namespace", label: "Namespace"}
    ]);
  }

  // implement
  getResources(snapshot) {
    return snapshot.getResources('Filter')
  }

  // implement
  sortFn(sortByAttribute) {
    return function(r1, r2) {
      if (sortByAttribute === "name" || sortByAttribute === "namespace") {
        return r1.metadata[sortByAttribute].localeCompare(r2.metadata[sortByAttribute]);
      } else {
        return r1.spec[sortByAttribute].localeCompare(r2.spec[sortByAttribute]);
      }
    }
  }

  // implement
  renderSet() {
    let newFilter = {
      metadata: {
        namespace: "default",
        name: ""
      },
      spec: {
        OAuth2: {}
      }
    };
    return html`
<dw-filter
  .resource=${newFilter}
  .state=${this.addState}>
  <add-button></add-button>
</dw-filter>
<div>
  ${this.resources.map(r => {
    return html`<dw-filter .resource=${r} .state=${this.state(r)}></dw-filter>`
  })}
</div>`
  }

}

customElements.define('dw-filters', Filters);
