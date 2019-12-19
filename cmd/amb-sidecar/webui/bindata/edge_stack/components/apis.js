import { LitElement, html, css } from '../vendor/lit-element.min.js'
import {ApiFetch} from "./api-fetch.js";

export class APIs extends LitElement {

  static get styles() {
    return css`
#signup {
    margin: auto;
    width: 50%;
}

#signup-content, #signup-finished {
    display: none;
    background-color: #fefefe;
    padding: 20px;
    border: 1px solid #888;
    width: 80%;
}

#signup-error {
    color: #fe0000;
}

.invalid {
    background-color: #fe0000;
}
`
  }

  static get properties() {
    return {
      apis: { type: Array },
      details: { type: Object },
    }
  }

  constructor() {
    super();
    this.reset();
    this.doRefresh = true;             // true to allow auto-refreshing
    this.waitingForHackStyles = false; // true while we have a deferred hackStyles call
  }

  reset() {
    this.apis = [];
    this.details = {}
  }

  connectedCallback() {
    super.connectedCallback();
    this.loadFromServer()
  }

  loadFromServer() {
      ApiFetch('/openapi/services')
        .then((resp) => {
          if (resp.status === 401 || resp.status === 403) {
            return new Promise((_, reject) => {
              reject(`Invalid Status: ${resp.status}`);
            });
          } else {
            return resp.json();
          }
        })
        .then((json) => {
          this.apis = json;
        })
        .catch((err) => console.log(err));

    if (this.doRefresh) {
      //console.log("will reload APIs in 10 seconds");
      setTimeout(this.loadFromServer.bind(this), 10000)
    }
  }

  deferHackStyles() {
    if (!this.waitingForHackStyles) {
      this.waitingForHackStyles = true;
      setTimeout(this.hackStyles.bind(this), 1)
    }
  }

  linkCSS(href) {
    let link = document.createElement('link');
    link.setAttribute('rel', 'stylesheet');
    link.setAttribute('type', 'text/css');
    link.setAttribute('href', href);
    return link
  }

  hackStyles() {
    this.waitingForHackStyles = false;
    let apiDivs = this.shadowRoot.children[0].getElementsByTagName('rapi-doc');

    for (let i = 0; i < apiDivs.length; i++) {
      let aDiv = apiDivs[i];
      let needLinks = false;

      while (true) {
        if (!aDiv.shadowRoot) {
          this.deferHackStyles();
          return;
        }

        let aDivChildren = aDiv.shadowRoot.children;
        if (!aDivChildren) {
          break;
        }

        let aKid = aDivChildren[0];
        if (aKid.tagName === 'STYLE') {
          aKid.remove();
          needLinks = true
        }
        else {
          break;
        }
      }

      if (needLinks) {
        aDiv.shadowRoot.prepend(this.linkCSS("/edge_stack/styles/rapidoc-table.css"));
        aDiv.shadowRoot.prepend(this.linkCSS("/edge_stack/styles/rapidoc-input.css"));
        aDiv.shadowRoot.prepend(this.linkCSS("/edge_stack/styles/rapidoc-fonts.css"));
        aDiv.shadowRoot.prepend(this.linkCSS("/edge_stack/styles/rapidoc-flex.css"));
        aDiv.shadowRoot.prepend(this.linkCSS("/edge_stack/styles/rapidoc-endpoint.css"));
        aDiv.shadowRoot.prepend(this.linkCSS("/edge_stack/styles/rapidoc-elements.css"));
        aDiv.shadowRoot.prepend(this.linkCSS("/edge_stack/styles/rapidoc-colors.css"))
      }
    }
  }

  apiName(api) {
    let apiName = `${api.service_name}.${api.service_namespace}`;

    return `${apiName}`;
  }

  compareAPIs(api1, api2) {
    let name1 = this.apiName(api1);
    let name2 = this.apiName(api2);

    return name1.localeCompare(name2);
  }

  render() {
    window.apis = this;

    if (this.apis.length === 0) {
      return html`
<link rel="stylesheet" href="../styles/resources.css">
<div class="header_con">
  <div class="col">
    <img alt="apis logo" class="logo" src="../images/svgs/apis.svg">
      <g id="Layer_2" data-name="Layer 2">
        <g id="Layer_1-2" data-name="Layer 1"></g>
      </g>
    </img>
    </div>

  <div class="col">
    <h1>APIs</h1>
    <p>No API documentation is available.</p>
    <p><a href="https://www.getambassador.io/reference/dev-portal" target="_blank">Publish your API documentation and get started with the Dev Portal.</a></p>
  </div>
</div>
`
    }
    else {
      this.deferHackStyles();

      let rendered = [];

      this.apis.sort(this.compareAPIs.bind(this));

      this.apis.forEach((api, index) => {
        rendered.push(this.renderAPIDocs(api, index))
      });

      return html`
<link rel="stylesheet" href="../styles/resources.css">
<div class="header_con">
  <div class="col">
    <img alt="apis2 logo" src="../images/svgs/apis2.svg"> 
      <g id="Layer_2" data-name="Layer 2">
        <g id="Layer_1-2" data-name="Layer 1"></g>
      </g>
    </img>
  </div>
  <div class="col">
    <h1>APIs</h1>
    <p><a href="https://www.getambassador.io/reference/dev-portal" target="_blank">Learn how to publish your API documentation.</a></p>
    <p>APIs with documentation.</p>
  </div>
</div>
<br/>
<div>
  ${rendered}
</div>
`
    }
  }

  renderAPIDocs(api, index) {
    let apiName = this.apiName(api);

    return html`
<div id="api-${apiName}" class="api-doc">
  ${this.renderAPIDetails(api, apiName)}
</div>
`
  }

  renderAPIDetails(api, apiName) {
    if (api.has_doc) {
      let detailKey = `${apiName}-details`;
      let detailURL = `/openapi/services/${api.service_namespace}/${api.service_name}/openapi.json`;

      return html`
<rapi-doc id="${detailKey}"
  spec-url="${detailURL}"
  show-header="false"
  show-info="true"
  allow-try="false"
  allow-server-selection="false"
  allow-authentication="false"
>
  <div>Documentation for ${apiName} at ${api.routing_prefix}</div>
</rapi-doc>
`
    }
    else {
      return '';
    }
  }
}

customElements.define('dw-apis', APIs);
