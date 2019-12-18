import {LitElement, html, css} from '/edge_stack/vendor/lit-element.min.js'
import './filterpolicies-common.js';

class FilterArgsOAuth2 extends LitElement {
  static get properties() {
    return {
      mode: {type: String}, // 'list' or 'edit'
      granttype: {type: String}, // 'ClientCredentials' or 'AuthorizationCode'
      data: {type: Object}, // mangled YAML; from FilterRef.rawValueToData/FilterRef.DataToRawValue
    };
  }

  set value(newVal) {
    this._value = newVal;
    this.dispatchEvent(new Event("change"));
  }

  get value() {
    return JSON.parse(JSON.stringify(this._value || this.data));
  }

  reset() {
    this.shadowRoot.querySelectorAll('dw-scope-values').forEach((el)=>{el.reset();});
    this._value = null;
  }

  constructor() {
    super();
    this.mode = "list";
    this.granttype = "AuthorizationCode";
    this.data = {
      scope: [],
      useInsteadOfRedirect: false,
      ifRequestHeader: {},
      insteadOfRedirectAction: "http",
      httpStatusCode: 403,
      filters: [],
    };
  }

  render() {
    return html`
<div class="row line">

  <div class="row-col margin-right justify-right">scope:</div>
  <div class="row-col">
    <dw-scope-values
      .mode=${this.mode}
      .data=${this.data.scope}
      @change=${(ev)=>{this.value = {...this.value, scope: ev.target.value}}}
    ></dw-scope-values>
  </div>

  <fieldset style=${this.granttype === "ClientCredentials" ? "display: none" : ""}>>
    <legend>When unauthorized: <select
        ?disabled=${this.mode !== "add" && this.mode !== "edit"}
        @change=${(ev)=>{this.value = {...this.value, useInsteadOfRedirect: (ev.target.value === "insteadOfRedirect")}}}
      >
      <option ?selected=${!this.data.useInsteadOfRedirect} value="redirect">Always redirect to IDP</option>
      <option ?selected=${this.data.useInsteadOfRedirect} value="insteadOfRedirect">Sometimes do something else</option>
    </select></legend>

    <div style=${this.value.useInsteadOfRedirect ? "" : "display: none"}>
      <div class="row-col margin-right justify-right">when to do something else:</div>
      <div class="row-col">
        <dw-header-field-selector
          .mode=${this.mode}
          .data=${this.data.ifRequestHeader}
          @change=${(ev)=>{this.value = {...this.value, ifRequestHeader: ev.target.value}}}
        ></dw-header-field-selector>
      </div>

      <fieldset>
        <legend>what to do: <select
            ?disabled=${this.mode !== "add" && this.mode !== "edit"}
            @change=${(ev)=>{this.value = {...this.value, insteadOfRedirectAction: ev.target.value}}}
          >
          <option ?selected=${this.data.insteadOfRedirectAction==="http"} value="http">simple HTTP error response</option>
          <option ?selected=${this.data.insteadOfRedirectAction==="filters"} value="filters">run other filters</option>
        </select></legend>
  
        <div style=${this.value.insteadOfRedirectAction === "http" ? "" : "display: none"}>
          <div class="row-col margin-right justify-right">HTTP status code:</div>
          <div class="row-col">
            <input type="number" min="100" max="599"
              value=${this.data.httpStatusCode}
              @change=${(ev)=>{this.value = {...this.value, httpStatusCode: ev.target.value}}}
            />
         </div>
        </div>

        <div style=${this.value.insteadOfRedirectAction === "filters" ? "" : "display: none"}>
          <div class="row-col margin-right justify-right">filters:</div>
          <div class="row-col">
            <dw-filterref-list
              .mode=${this.mode}
              .data=${this.data.filters}
              .namespace=${this.namespace}
              @change=${(ev)=>{this.value = {...this.value, filters: ev.target.value};}}
            ></dw-filterref-list>
          </div>
        </div>

      </fieldset><!-- insteadOfRedirectAction -->

    </div><!-- useInsteadOfRedirect===true -->

  </fieldset><!-- useInsteadOfRedirect -->

</div>
`;
  }
}
customElements.define('dw-filterargs-oauth2', FilterArgsOAuth2);
