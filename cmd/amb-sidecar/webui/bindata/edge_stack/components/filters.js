import {html} from '../vendor/lit-element.min.js'
import {SingleResource, SortableResourceSet} from './resources.js';

/**
 * The UI component for a filter.
 */
class Filter extends SingleResource {

  // internal
  init() {
    this.state.type = "OAuth2";
    this.state.subspec = null;
    this.state.OAuth2 = null;
    this.state.External = null;
    this.state.JWT = null;
    this.state.Plugin = null;
    this.state.Internal = null;
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
      if (this.state.type === this.filterType()) {
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
    result[this.state.type] = this.state.subspec;
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
  minimumNumberOfEditRows() {
    return 1; // this will show too many rows in the edit UI if the filter type has editable fields
              // but since we don't have a way to change the edit UI based on changes to the type field
              // we have to use the minimum here.
  }

  // override
  onAdd() {
    super.onAdd();
    this.onAddOrEdit();
  }

  // override
  onEdit() {
    super.onEdit();
    this.onAddOrEdit();
  }

  // internal
  onAddOrEdit() {
    let type = this.filterType();
    this.state.type = type;
    this.state.subspec = this.resource.spec[type];
    this.state[type] = this.state.subspec;
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
         @change=${(e)=>{subspec[name]=e.target.checked; this.requestUpdate()}}
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
<div class="row line">
  <div class="row-col margin-right justify-right">Authorization URL:</div>
  <div class="row-col">${this.input(subspec, "authorizationURL", "url")}</div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right">Grant Type:</div>
  <div class="row-col">${this.select(subspec, "grantType", grantTypes)}</div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right">Access Token Validation:</div>
  <div class="row-col">${this.select(subspec, "accessTokenValidation", ["auto", "jwt", "userinfo"])}</div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right">Client URL:</div>
  <div class="row-col">${this.input(subspec, "clientURL", "url")}</div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right">Client ID:</div>
  <div class="row-col">${this.input(subspec, "clientID")}</div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right">State TTL:</div>
  <div class="row-col">${this.input(subspec, "stateTTL", "number")}</div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right">Secret:</div>
  <div class="row-col">${this.input(subspec, "secret")}</div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right">Secret Name:</div>
  <div class="row-col">${this.input(subspec, "secretName")}</div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right">Secret Namespace:</div>
  <div class="row-col">${this.input(subspec, "secretNamespace")}</div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right">Insecure TLS:</div>
  <div class="row-col">${this.bool(subspec, "insecureTLS")}</div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right">Renegotiate TLS:</div>
  <div class="row-col">${this.bool(subspec, "renegotiateTLS")}</div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right">Max Stale:</div>
  <div class="row-col">${this.duration(subspec, "maxStale")}</div>
</div>
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
<div class="row line">
  <div class="row-col margin-right justify-right">Auth Service:</div>
  <div class="row-col">${this.input(subspec, "auth_service")}</div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right">Path Prefix:</div>
  <div class="row-col">${this.input(subspec, "path_prefix")}</div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right">TLS:</div>
  <div class="row-col">${this.bool(subspec, "tls")}</div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right">Protocol:</div>
  <div class="row-col">${this.select(subspec, "proto", ["http", "grpc"])}</div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right">Allow Request Body:</div>
  <div class="row-col">${this.bool(subspec, "allow_request_body")}</div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right">Timeout:</div>
  <div class="row-col">${this.input(subspec, "timeout_ms", "number")}</div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right">Allowed Request Headers:</div>
  <div class="row-col">${this.headerList(subspec, "allowed_request_headers")}</div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right">Allowed Authorization Headers:</div>
  <div class="row-col">${this.headerList(subspec, "allowed_authorization_headers")}</div>
</div>
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
<div class="row line">
  <div class="row-col margin-right justify-right">Valid algorithms:</div>
  <div class="row-col">${this.headerList(subspec, "validAlgorithms")}</div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right">JWKS URI:</div>
  <div class="row-col">${this.input(subspec, "jwksURI", "url")}</div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right">Audience:</div>
  <div class="row-col">${this.input(subspec, "audience")}</div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right">Require Audience:</div>
  <div class="row-col">${this.bool(subspec, "requireAudience")}</div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right">Issuer:</div>
  <div class="row-col">${this.input(subspec, "issuer", "url")}</div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right">Require Issuer:</div>
  <div class="row-col">${this.bool(subspec, "requireIssuer")}</div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right">Require Issued At:</div>
  <div class="row-col">${this.input(subspec, "requireIssuedAt", "url")}</div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right">Require Expires At:</div>
  <div class="row-col">${this.bool(subspec, "requireExpiresAt")}</div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right">Require Not Before:</div>
  <div class="row-col">${this.input(subspec, "requireNotBefore", "url")}</div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right">Inject Request Headers:</div>
  <div class="row-col">${this.headerTemplateList(subspec, "injectRequestHeaders")}</div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right">Insecure TLS:</div>
  <div class="row-col">${this.bool(subspec, "insecureTLS")}</div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right">Renegotiate TLS:</div>
  <div class="row-col">${this.bool(subspec, "renegotiateTLS")}</div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right">Error Headers:</div>
  <div class="row-col">${this.headerTemplateList(errorResponse, "headers")}</div>
</div>

<div class="row line">
  <div class="row-col margin-right justify-right">Error Body:</div>
  <div class="row-col"><textarea ?disabled=${this.disabled()} rows="10" cols="60"
            @change=${(e)=>{errorResponse.bodyTemplate=e.target.value; this.requestUpdate()}}>
    ${errorResponse.bodyTemplate}
  </textarea></div>
</div>
`
  }

  // internal
  renderPlugin(subspec) {
    return html`
<div class="row line">
  <div class="row-col margin-right justify-right">name:</div>
  <div class="row-col">${this.input(subspec, "name")}</div>
</div>
`
  }

  // internal
  renderInternal(subspec) {
  }

  // internal
  updateType(newType) {
    this[this.state.type] = this.state.subspec;
    if (this.state[newType] === null || this.state[newType] === undefined) {
      this.state[newType] = {}
    }
    this.state.subspec = this.state[newType];
    this.state.type = newType;
    this.requestUpdate();
  }

  /**
   * Implement.
   */
  renderResource() {
    let type = this.state.mode === "add" || this.state.mode === "edit" ? this.state.type : this.filterType();
    let render = this[`render${type}`].bind(this);
    let subspec = this.state.mode === "add" || this.state.mode === "edit" ? this.state.subspec : this.resource.spec[type];
    let rendered = render(subspec);
    return html`
<div class="row line">
  <div class="row-col margin-right justify-right">type:</div>
  <div class="row-col">
    <select ?disabled=${this.disabled()} @change=${(e)=>this.updateType(e.target.value)}>
    <option .selected=${type === "OAuth2"} value="OAuth2">OAuth2</option>
    <option .selected=${type === "JWT"} value="JWT">JWT</option>
    <option .selected=${type === "External"} value="External">External</option>
    <option .selected=${type === "Plugin"} value="Plugin">Plugin</option>
    <option .selected=${type === "Internal"} value="Internal">Internal</option>
  </select>
  </div>
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

  renderInner() {
    let shtml = super.renderInner();
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
<div class="header_con">
  <div class="col">
    <img alt="filters logo" class="logo" src="../images/svgs/filters.svg" width="32" height="32">
      <g class="nc-icon-wrapper" stroke-linecap="square" stroke-linejoin="miter" stroke-width="2" fill="#608cee" stroke="#608cee"></g>
    </img>
  </div>
  <div class="col">
    <h1>Filters</h1>
    <p>Configure middleware for your requests.</p>
  </div>
  <div class="col2">
    <a class="cta add ${this.readOnly() ? "off" : ""}" @click=${()=>this.shadowRoot.getElementById("add-filter").onAdd()}>
      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 30 30"><defs><style>.cls-a{fill:none;stroke:#000;stroke-linecap:square;stroke-miterlimit:10;stroke-width:2px;}</style></defs><title>add_1</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><line class="cls-a" x1="15" y1="9" x2="15" y2="21"/><line class="cls-a" x1="9" y1="15" x2="21" y2="15"/><circle class="cls-a" cx="15" cy="15" r="14"/></g></g></svg>
      <div class="label">add</div>
    </a>
    <div class="sortby">
      <select id="sortByAttribute" @change=${this.onChangeSortByAttribute.bind(this)}>
    ${this.sortFields.map(f => {
      return html`<option value="${f.value}">${f.label}</option>`
    })}
      </select>
    </div>
  </div>
</div>
<dw-filter id="add-filter" .resource=${newFilter} .state=${this.addState}></dw-filter>
${shtml}
`;
  }

  // implement
  renderSet() {
    return html`
<div>
  ${this.resources.map(r => {
    return html`<dw-filter .resource=${r} .state=${this.state(r)}></dw-filter>`
  })}
</div>`
  }

}

customElements.define('dw-filters', Filters);
