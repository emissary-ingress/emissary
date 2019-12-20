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
    <img alt="filters logo" class="logo" src="../images/svgs/filters.svg" width="32" height="32">
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
