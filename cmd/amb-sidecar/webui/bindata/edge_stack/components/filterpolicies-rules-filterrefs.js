import {LitElement, html, css} from '/edge_stack/vendor/lit-element.min.js'
import {Snapshot} from './snapshot.js'
import './filterpolicies-common.js';
import './filterargs-oauth2.js';
import './filterargs-jwt.js';

class FilterRef extends LitElement {
  static get properties() {
    return {
      mode: {type: String}, // 'list' or 'edit'
      data: {type: Object}, // from YAML
      filters: {type: Object},

      value: {type: Object},
    };
  }

  constructor() {
    super();
    this.mode = "list";
    this.data = {
      name: "",
      namespace: "",
      onDeny: "break",
      onAllow: "continue",
      ifRequestHeader: null,
      'arguments': null,
    };
    this.filters = {};
    Snapshot.subscribe((snapshot)=>{
      this.filters = Object.fromEntries(snapshot.getResources("Filter").map((r)=>[`${r.metadata.name}.${r.metadata.namespace}`, r]));
    });
  }

  set rawValue(newVal) {
    this._value = newVal;
    this.dispatchEvent(new Event("change"));
  }

  get rawValue() {
    return JSON.parse(JSON.stringify(this._value || this.dataToRawValue(this.data)));
  }

  set value(val) {
    throw new Error("set .rawValue instead");
  }

  get value() {
    return this.rawValueToData( this.rawValue);
  }

  rawValueToData(raw) {
    let dat = {...raw, 'arguments': {}}
    let qname = `${raw.name}.${raw.namespace}`;
    let rawArgs = raw['arguments'][qname];
    let args = {};
    switch (this._filterType(qname)) {
      case 'OAuth2':
        args = {
          scopes: rawArgs.scope,
        };
        if (rawArgs.useInsteadOfRedirect) {
          args.insteadOfRedirect = {
            ifRequestHeader: rawArgs.ifRequestHeader,
          };
          switch (rawArgs.insteadOfRedirectAction) {
            case 'http':
              args.insteadOfRedirect.httpStatusCode = rawArgs.httpStatusCode;
              break;
            case 'filters':
              args.insteadOfRedirect.filters = rawArgs.filters;
              break;
          }
        }
        break;
      case 'JWT':
        args = rawArgs;
        break;
    }
    dat['arguments'] = args;
    return dat;
  }

  dataToRawValue(dat) {
    let raw = {...dat, 'arguments': {}};
    let qname = `${dat.name}.${dat.namespace}`;
    let datArgs = dat['arguments']||{};
    let args = {};
    switch (this._filterType(qname)) {
      case 'OAuth2':
        args = {
          scope: (datArgs.scopes||[]),
          useInsteadOfRedirect: (Object.keys(datArgs.insteadOfRedirect||{}).length > 0),
          ifRequestHeader: ((datArgs.insteadOfRedirect||{}).ifRequestHeader||{}),
          insteadOfRedirectAction: (((datArgs.insteadOfRedirect||{}).httpStatusCode||0) > 0 ? 'http' : 'filters'),
          httpStatusCode: ((datArgs.insteadOfRedirect||{}).httpStatusCode||403),
          filters: ((datArgs.insteadOfRedirect||{}).filters||[]),
        };
        break;
      case 'JWT':
        args = datArgs;
        break;
    }
    raw['arguments'][qname] = args;
    return raw;
  }

  reset() {
    this._value = null;
    this.shadowRoot.querySelectorAll('dw-filterargs-jwt').forEach((el)=>{el.reset();});
    this.shadowRoot.querySelectorAll('dw-filterargs-oauth2').forEach((el)=>{el.reset();});
    this.shadowRoot.querySelectorAll('dw-header-field-selector').forEach((el)=>{el.reset();});
    this.shadowRoot.querySelectorAll('select').forEach((el)=>{el.value = el.querySelector('option[selected]').value;});
  }

  _filterType(qname) {
    let filterSpec = (this.filters[qname]||{}).spec||{};
    if (filterSpec.OAuth2) {
      return "OAuth2";
    }
    if (filterSpec.External) {
      return "External";
    }
    if (filterSpec.JWT) {
      return "JWT";
    }
    if (filterSpec.Plugin) {
      return "Plugin";
    }
    return "<unknown>";
  }

  static get styles() {
    return css`
* {
  box-sizing: border-box;
}

:host {
  display: block
}

dl {
  display: grid;
  grid-template-columns: max-content;
  grid-gap: 0;
  margin: 0;
}
dl > dt {
  grid-column: 1 / 2;
  text-align: right;
	font-weight: 600;
}
dl > dt::after {
  content: ":";
}
dl > dd {
  grid-column: 2 / 3;
}
dl > * {
  margin: 0;
	padding: 10px 5px;
	border-bottom: 1px solid rgba(0, 0, 0, .1);
}
dl > :nth-last-child(2), dl > :last-child {
	border-bottom: none;
}
`;
  }

  render() {
    return html`<dl>

  <dt>filter</dt>
  <dd>
    <visible-modes .mode=${this.mode} list><span>${this.data.name}.${this.data.namespace}</span></visible-modes>
    <visible-modes .mode=${this.mode} edit add><select
      @change=${(ev)=>{
        let qname = ev.target.value;
        let sep = qname.lastIndexOf('.');
        let name = qname.slice(0, sep);
        let namespace = qname.slice(sep+1);
        this.rawValue = {
          ...this.rawValue,
          name: name,
          namespace: namespace,
        };
      }}
    >
      ${Object.entries(this.filters).sort((a, b) => b[0].localeCompare(a[0])).map(([k, v])=>{
        return html`<option ?selected=${k===`${this.data.name}.${this.data.namespace}`} value=${k}>${k} (${this._filterType(k)})</option>`;
      })}
      ${ (this.filters[`${this.data.name}.${this.data.namespace}`]) ? html`` : html`
        <option selected value=${`${this.data.name}.${this.data.namespace}`}>${this.data.name}.${this.data.namespace} (missing)</option>
      `}
    </select></visible-modes>
  </dd>

  <dt>onDeny</dt>
  <dd>
    <visible-modes .mode=${this.mode} list><span><tt>${this.data.onDeny||"break"}</tt></span></visible-modes>
    <visible-modes .mode=${this.mode} edit add><select
      @change=${(ev)=>{this.rawValue = {...this.rawValue, onDeny: ev.target.value};}}
    >
      <option ?selected=${(this.data.onDeny||"break")==="break"} value="break"><tt>break</tt> (default)</option>
      <option ?selected=${(this.data.onDeny||"break")==="continue"} value="continue"><tt>continue</tt></option>
    </select></visible-modes>
  </dd>

  <dt>onAllow</dt>
  <dd>
    <visible-modes .mode=${this.mode} list><span><tt>${this.data.onAllow||"continue"}</tt></span></visible-modes>
    <visible-modes .mode=${this.mode} edit add><select
      @change=${(ev)=>{this.rawValue = {...this.rawValue, onAllow: ev.target.value};}}
    >
      <option ?selected=${(this.data.onAllow||"continue")==="break"} value="break"><tt>break</tt></option>
      <option ?selected=${(this.data.onAllow||"continue")==="continue"} value="continue"><tt>continue</tt> (default)</option>
    </select></visible-modes>
  </dd>

  <dt>ifRequestHeader</dt>
  <dd><dw-header-field-selector
    .mode=${this.mode}
    .data=${this.data.ifRequestHeader}
    @change=${(ev)=>{this.rawValue = {...this.rawValue, ifRequestHeader: ev.target.value};}}
  ></dw-header-field-selector></dd>

  ${(()=>{switch (this._filterType(`${this.rawValue.name}.${this.rawValue.namespace}`)) {
    case 'OAuth2':
      return html`
        <dt>arguments</dt>
        <dd><dw-filterargs-oauth2
          .mode=${this.mode}
          .data=${(this.rawValue['arguments'][`${this.rawValue.name}.${this.rawValue.namespace}`]||{})}
          @change=${(ev)=>{
            let dat = this.rawValue;
            dat['arguments'][`${dat.name}.${dat.namespace}`] = ev.target.value;
            this.rawValue = dat;
          }}
        ></dw-filterargs-oauth2></dd>
      `;
    case 'JWT':
      return html`
        <dt>arguments</dt>
        <dd><dw-filterargs-jwt
          .mode=${this.mode}
          .data=${(this.rawValue['arguments'][`${this.rawValue.name}.${this.rawValue.namespace}`]||{})}
          @change=${(ev)=>{
            let dat = this.rawValue;
            dat['arguments'][`${dat.name}.${dat.namespace}`] = ev.target.value;
            this.rawValue = dat;
          }}
        ></dw-filterargs-jwt></dd>
      `;
    default:
      return html``;
  }})()}

</dl>`;
  }
}
customElements.define('dw-filterref', FilterRef);

class FilterRefList extends LitElement {
  static get properties() {
    return {
      mode: {type: String}, // 'list' or 'edit'
      data: {type: Array}, // from YAML
      namespace: {type: String},
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
    this._value = null;
    this.shadowRoot.querySelectorAll('dw-filterref').forEach((el)=>{el.reset();});
  }

  // implement
  constructor() {
    super();
    this.mode = "list";
    this.data = [];
  }

  static get styles() {
    return css`
* {
  box-sizing: border-box;
}
ol {
  margin: 0;
  padding: 0;
  list-style: none;
  counter-reset: mycounter;
}
li::before {
  /* content */
  counter-increment: mycounter;
  content: counter(mycounter) ".";
  font-weight: 600;

  /* same as a row */
	padding: 10px 5px;

  /* positioning */
  float: left;
  margin: 0;
  width: 1.1em;
  vertical-align: middle;
  text-align: center;
}
li {
	border-bottom: 1px solid rgba(0, 0, 0, .1);
}
li:last-child {
	border-bottom: none;
}
`;
  }

  // implement
  render() {
    let newRefData = {
      name: "",
      namespace: this.namespace,
      onDeny: "break",
      onAllow: "continue",
      ifRequestHeader: null,
      'arguments': null,
    };
    return html`<div>

  ${(this.data.length == 0 && this.mode != "add" && this.mode != "edit") ? html`(none)` : ``}

  <ol>

    ${this.value.map((refData, i) => {
      return html`<li><dw-filterref
        .mode=${this.mode}
        .data=${refData}
        @change=${(ev)=>{
          let dat = this.value;
          dat[i] = ev.target.value;
          this.value = dat;
        }}
      ></dw-filterref></li>`;
    })}

    ${ (this.mode === "add" || this.mode === "edit") ? html`
     <li><button
        @click=${(ev)=>{this.value = [...this.value, newRefData];}}
      >Add filter reference</button></li>
    ` : html`` }

  </ol>

</div>`;
  }

}
customElements.define('dw-filterref-list', FilterRefList);

