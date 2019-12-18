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
    this.shadowRoot.querySelectorAll('dw-filterref-list').forEach((el)=>{el.reset();});
    this.shadowRoot.querySelectorAll('dw-header-field-selector').forEach((el)=>{el.reset();});
    this.shadowRoot.querySelectorAll('dw-scope-values').forEach((el)=>{el.reset();});
    this.shadowRoot.querySelectorAll('input').forEach((el)=>{el.value = el.defaultValue;});
    this.shadowRoot.querySelectorAll('select').forEach((el)=>{el.value = el.querySelector('option[selected]').value;});
    this._value = null;
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

  render() {
    return html`
<div class="row line">

  <div class="row-col margin-right justify-right">filter:</div>
  <div class="row-col">
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
    </select></visible-modes> 
  </div>

  <div class="row-col margin-right justify-right">onDeny:</div>
  <div class="row-col">
    <visible-modes .mode=${this.mode} list><span><tt>${this.data.onDeny}</tt></span></visible-modes>
    <visible-modes .mode=${this.mode} edit add><select
      @change=${(ev)=>{this.rawValue = {...this.rawValue, onDeny: ev.target.value};}}
    >
      <option ?selected=${this.data.onDeny==="break"} value="break"><tt>break</tt> (default)</option>
      <option ?selected=${this.data.onDeny==="continue"} value="continue"><tt>continue</tt></option>
    </select></visible-modes> 
  </div>

  <div class="row-col margin-right justify-right">onAllow:</div>
  <div class="row-col">
    <visible-modes .mode=${this.mode} list><span><tt>${this.data.onAllow}</tt></span></visible-modes>
    <visible-modes .mode=${this.mode} edit add><select
      @change=${(ev)=>{this.rawValue = {...this.rawValue, onAllow: ev.target.value};}}
    >
      <option ?selected=${this.data.onAllow==="break"} value="break"><tt>break</tt></option>
      <option ?selected=${this.data.onAllow==="continue"} value="continue"><tt>continue</tt> (default)</option>
    </select></visible-modes> 
  </div>

  <div class="row-col margin-right justify-right">ifRequestHeader:</div>
  <div class="row-col"><dw-header-field-selector
    .mode=${this.mode}
    .data=${this.data.ifRequestHeader}
    @change=${(ev)=>{this.rawValue = {...this.rawValue, ifRequestHeader: ev.target.value};}}
  ></dw-header-field-selector></div>

  ${(()=>{switch (this._filterType(`${this.rawValue.name}.${this.rawValue.namespace}`)) {
    case 'OAuth2':
      return html`
        <div class="row-col margin-right justify-right">arguments:</div>
        <div class="row-col">
          <dw-filterargs-oauth2
            .mode=${this.mode}
            .data=${(this.rawValue['arguments'][`${this.rawValue.name}.${this.rawValue.namespace}`]||{})}
            @change=${(ev)=>{
              let dat = this.rawValue;
              dat['arguments'][`${dat.name}.${dat.namespace}`] = ev.target.value;
              this.rawValue = dat;
            }}
          ></dw-scope-values>
        </div>
      `;
    case 'JWT':
      return html`
        <div class="row-col margin-right justify-right">arguments:</div>
        <div class="row-col">
          <dw-filterargs-jwt
            .mode=${this.mode}
            .data=${(this.rawValue['arguments'][`${this.rawValue.name}.${this.rawValue.namespace}`]||{})}
            @change=${(ev)=>{
              let dat = this.rawValue;
              dat['arguments'][`${dat.name}.${dat.namespace}`] = ev.target.value;
              this.rawValue = dat;
            }}
          ></dw-scope-values>
        </div>
      `;
    default:
      return html``;
  }})()}

</div>
`;
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
    this.shadowRoot.querySelectorAll('dw-filterref').forEach((el)=>{el.reset();});
    this._value = null;
  }

  // implement
  constructor() {
    super();
    this.mode = "list";
    this.data = [];
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
    return html`
<visible-modes list>
  ${this.data.length == 0 ? html`(none)` : html``}
</visible-modes>

<ul>
${this.data.map((refData, i) => {
  return html`<li>
    <dw-filterref
      .mode=${this.mode}
      .data=${refData}
      @change=${(ev)=>{
        let dat = this.value;
        dat[i] = ev.target.value;
        this.value = dat;
      }}
    ></dw-filterref>
  </li>`;
})}
</ul>

<visible-modes add edit>
  <button
    @click=${(ev)=>{this.value = [...this.value, newRefData];}}
  >Add filter reference</button>
</visible-modes>
`;
  }

}
customElements.define('dw-filterref-list', FilterRefList);

