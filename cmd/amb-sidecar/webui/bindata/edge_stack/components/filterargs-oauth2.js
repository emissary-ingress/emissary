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

  static get styles() {
    return css`
* {
  box-sizing: border-box;
}

:host {
  display: block
}

dl > dt {
	font-weight: 600;
}
dl > dt::after {
  content: ":";
}
dl > dd {
	padding: 10px 5px;

  margin-left: 0;
  padding-left: 1.5em;
	border-bottom: 1px solid rgba(0, 0, 0, .1);
}
dl > dd:nth-last-child(2), dl > dd:last-child {
	border-bottom: none;
}
`;
  }

  render() {
    return html`<dl>

  <dt>scope</dt>
  <dd><dw-scope-values
    .mode=${this.mode}
    .data=${this.data.scope}
    @change=${(ev)=>{this.value = {...this.value, scope: ev.target.value}}}
  ></dw-scope-values></dd>

  <dt>when unauthorized</dt>
  <dd><fieldset style=${this.granttype === "ClientCredentials" ? "display: none" : ""}>
    <legend><select
        ?disabled=${this.mode !== "add" && this.mode !== "edit"}
        @change=${(ev)=>{this.value = {...this.value, useInsteadOfRedirect: (ev.target.value === "insteadOfRedirect")}}}
      >
      <option ?selected=${!this.data.useInsteadOfRedirect} value="redirect">always redirect to IDP</option>
      <option ?selected=${this.data.useInsteadOfRedirect} value="insteadOfRedirect">sometimes do something else</option>
    </select></legend>
    <dl style=${this.value.useInsteadOfRedirect ? "" : "display: none"}>

      <dt>when to do something else</dt>
      <dd style=${this.value.useInsteadOfRedirect ? "" : "display: none"}>
        <dw-header-field-selector
          .mode=${this.mode}
          .data=${this.data.ifRequestHeader}
          @change=${(ev)=>{this.value = {...this.value, ifRequestHeader: ev.target.value}}}
        ></dw-header-field-selector>
      </dd>

      <dt>what to do</dt>
      <dd><fieldset>
        <legend><select
          ?disabled=${this.mode !== "add" && this.mode !== "edit"}
          @change=${(ev)=>{this.value = {...this.value, insteadOfRedirectAction: ev.target.value}}}
        >
          <option ?selected=${this.data.insteadOfRedirectAction==="http"} value="http">simple HTTP error response</option>
          <option ?selected=${this.data.insteadOfRedirectAction==="filters"} value="filters">run other filters</option>
        </select></legend>

        <dl style=${this.value.insteadOfRedirectAction === "http" ? "" : "display: none"}>
            <dt>HTTP status code</dt>
            <dd><input type="number" min="100" max="599"
              value=${this.data.httpStatusCode}
                @change=${(ev)=>{this.value = {...this.value, httpStatusCode: ev.target.value}}}
            /></dd>
        </dl>

        <dl style=${this.value.insteadOfRedirectAction === "filters" ? "" : "display: none"}>
          <dt>filters</dt>
          <dd><dw-filterref-list
            .mode=${this.mode}
            .data=${this.data.filters}
            .namespace=${this.namespace}
            @change=${(ev)=>{this.value = {...this.value, filters: ev.target.value};}}
          ></dw-filterref-list></dd>
        </dl>

      </fieldset></dd><!-- "what to do" -->

    </dl><!-- "when unauthorized" -->
  </fieldset></dd><!-- "when unauthorized" -->

</dl>
`;
  }
}
customElements.define('dw-filterargs-oauth2', FilterArgsOAuth2);
