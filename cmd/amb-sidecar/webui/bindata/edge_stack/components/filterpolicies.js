import {html, css} from '../vendor/lit-element.min.js'
import {SingleResource, SortableResourceSet} from './resources.js';
import './filterpolicies-rules.js';

class FilterPolicy extends SingleResource {

  // override
  constructor() {
    super();
  }

  // implement
  kind() {
    return "FilterPolicy";
  }

  // implement
  spec() {
    return {
      rules: this.shadowRoot.querySelector('dw-filterpolicy-rule-list').value,
    };
  }

  // override
  reset() {
    super.reset();
    this.shadowRoot.querySelectorAll('dw-filterpolicy-rule-list').forEach((el)=>{el.reset();});
  }

  // override
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
 * {
	margin: 0;
	padding: 0;
	border: 0;
	position: relative;
	box-sizing: border-box
 }
  
 *, textarea {
	vertical-align: top
 }
  
 .card {
	background: #fff;
	border-radius: 10px;
	padding: 10px 30px 10px 30px;
	box-shadow: 0 10px 5px -11px rgba(0, 0, 0, .6);
	width: 100%;
	-webkit-flex-direction: row;
	-ms-flex-direction: row;
	flex-direction: row;
	-webkit-flex: 1 1 1;
	-ms-flex: 1 1 1;
	flex: 1 1 1;
	margin: 30px 0 0;
	font-size: .9rem;
 }
  
 .card, .card .col .con {
	display: -webkit-flex;
	display: -ms-flexbox;
	display: flex
 }
  
 .card .col {
	-webkit-flex: 1 0 50%;
	-ms-flex: 1 0 50%;
	flex: 1 0 50%;
	padding: 0 30px 0 0
 }
  
 .card .col .con {
	margin: 10px 0;
	-webkit-flex: 1;
	-ms-flex: 1;
	flex: 1;
	-webkit-justify-content: flex-end;
	-ms-flex-pack: end;
	justify-content: flex-end;
	height: 30px
 }
  
 .card .col, .card .col .con label, .card .col2, .col2 a.cta .label {
	-webkit-align-self: center;
	-ms-flex-item-align: center;
	-ms-grid-row-align: center;
	align-self: center
 }
  
 .col2 a.cta  {
	text-decoration: none;
	border: 2px #efefef solid;
	border-radius: 10px;
	width: 90px;
	padding: 6px 8px;
	max-height: 35px;
	-webkit-flex: auto;
	-ms-flex: auto;
	flex: auto;
	margin: 10px auto;
	color: #000;
	transition: all .2s ease;
	cursor: pointer;
 }
  
 .col2 a.cta .label {
	text-transform: uppercase;
	font-size: .8rem;
	font-weight: 600;
	line-height: 1rem;
	padding: 0 0 0 10px;
	-webkit-flex: 1 0 auto;
	-ms-flex: 1 0 auto;
	flex: 1 0 auto
 }
  
 .col2 a.cta svg {
	width: 15px;
	height: auto
 }
  
 .col2 a.cta svg path, .col2 a.cta svg polygon {
	transition: fill .7s ease;
	fill: #000
 }
  
 .col2 a.cta:hover {
	color: #5f3eff;
	transition: all .2s ease;
	border: 2px #5f3eff solid
 }
  
 .col2 a.cta:hover svg path, .col2 a.cta:hover svg polygon {
	transition: fill .2s ease;
	fill: #5f3eff
 }
  
 .col2 a.cta {
	display: -webkit-flex;
	display: -ms-flexbox;
	display: flex;
	-webkit-flex-direction: row;
	-ms-flex-direction: row;
	flex-direction: row
 }
  
 .col2 a.off {
	display: none;
 }
  
 div.off, span.off, input.off, b.off {
	display: none;
 }
  
 .row {
	display: -webkit-flex;
	display: -ms-flexbox;
	display: flex;
	-webkit-flex-direction: row;
	-ms-flex-direction: row;
	flex-direction: row
 }
  
 .row .row-col {
	flex: 1;
	padding: 10px 5px;
 }
 .row .row-col:nth-child(1) {
	font-weight: 600;
 }
 .row .row-col:nth-child(2) {
	flex: 2;
 }
  
 .card .row {
	border-bottom: 1px solid rgba(0, 0, 0, .1);
 }
  
 .card div.line:nth-last-child(2) {
	border-bottom: none;
 }
  
 .errors ul {
	color: red;
 }
  
 .margin-right {
	margin-right: 20px;
 }
 .justify-right {
	text-align: right;
 }
  
 button, input, select, textarea {
	font-family: inherit;
	font-size: 100%;
	margin: 0
 }
  
 button, input {
	line-height: normal
 }
  
 button, html input[type=button], input[type=reset], input[type=submit] {
	-webkit-appearance: button;
	cursor: pointer
 }
 input[type=search] {
	-webkit-appearance: textfield;
	box-sizing: content-box
 }
  
 input {
	background: #efefef;
	padding: 5px;
	margin: -5px 0;
	width: 100%;
 }
 input[type=checkbox], input[type=radio] {
	box-sizing: border-box;
	padding: 0;
	width: inherit;
	margin: 0.2em 0;
 }
 textarea {
	background-color: #efefef;
	padding: 5px;
	width: 100%;
 }
 div.yaml-wrapper {
	overflow-x: scroll;
 }
 div.yaml-wrapper pre {
	width: 500px;
	font-size: 90%;
 }
 div.yaml ul {
	color: var(--dw-purple);
	margin-left: 2em;
	margin-bottom: 0.5em;
 }
 div.yaml li {
 }
 div.yaml li .yaml-path {
 }
 div.yaml li .yaml-change {
 }
 div.namespace {
	display: inline;
 }
 div.namespaceoff {
	display: none;
 }
 div.namespace-input {
	margin-top: 0.1em;
 }
 .pararen {
	-webkit-justify-content: center;
	-ms-flex-pack: center;
	justify-content: center;
	-webkit-align-content: center;
	-ms-flex-line-pack: center;
	align-content: center;
	-webkit-align-items: center;
	-ms-flex-align: center;
	align-items: center;
 }
 div.namespace-input .pararen {
	font-size: 1.2rem;
	padding: 0 5px;
	display: inline;
	top: -0.2em;
 }
 div.namespace-input input {
	width: 90%;
 }
	`;
  }

  // implement
  renderResource() {
    return html`
<dl>
  <dt>rules</dt>
  <dd style="padding-top: 0">
    <dw-filterpolicy-rule-list
      .mode=${this.state.mode}
      .data=${this.resource.spec.rules}
      .namespace=${this.resource.metadata.namespace}
    ></dw-filterpolicy-rule-list>
  </dd>
</dl>
`;
  }

  // override
  minimumNumberOfAddRows() {
    return 1;
  }

  // override
  minimumNumberOfEditRows() {
    return 1;
  }
}
customElements.define('dw-filterpolicy', FilterPolicy);

class FilterPolicies extends SortableResourceSet {
  // implement
  constructor() {
    super([
      {value: "name", label: "Name"},
      {value: "namespace", label: "Namespace"},
    ]);
  }

  // implement
  sortFn(sortByAttribute) {
    return function(a, b) {
      switch (sortByAttribute) {
      case "name":
      case "namespace":
        return a.metadata[sortByAttribute].localeCompare(b.metadata[sortByAttribute]);
      default:
        throw new Error("how did sortByAttribute get set wrong!?");
      }
    }
  }

  // implement
  getResources(snapshot) {
    return snapshot.getResources("FilterPolicy");
  }

  // implement
  renderInner() {
    let shtml = super.renderInner();
    let newFilterPolicy = {
      metadata: {
        namespace: "default",
        name: ""
      },
      spec: {
        rules: []
      }
    };
    return html`
<div class="header_con">
  <div class="col">
    <img alt="filters logo" class="logo" src="../images/svgs/filters.svg">
  </div>
  <div class="col">
    <h1>FilterPolicies</h1>
    <p>Configure which middlewares apply to which requests.</p>
  </div>
  <div class="col2">
    <a class="cta add ${this.readOnly() ? "off" : ""}" @click=${()=>this.shadowRoot.getElementById("add-filterpolicy").onAdd()}>
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
<dw-filterpolicy id="add-filterpolicy" .resource=${newFilterPolicy} .state=${this.addState}></dw-filterpolicy>
${shtml}
`;
  }

  // implement
  renderSet() {
    return html`
<div>
  ${this.resources.map(r => {
    return html`<dw-filterpolicy .resource=${r} .state=${this.state(r)}></dw-filterpolicy>`;
  })}
</div>
`;
  }
}
customElements.define('dw-filterpolicies', FilterPolicies);
