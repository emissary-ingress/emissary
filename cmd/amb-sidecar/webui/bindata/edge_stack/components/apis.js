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
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24.02 23.53"><title>apis</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><path class="cls-1" d="M12,5.19a6.58,6.58,0,1,0,6.58,6.58A6.58,6.58,0,0,0,12,5.19Zm0,12.37a5.8,5.8,0,1,1,5.8-5.79A5.8,5.8,0,0,1,12,17.56Z"/><polygon class="cls-1" points="9.68 9.47 7.28 11.88 9.68 14.28 10.23 13.73 8.38 11.88 10.23 10.02 9.68 9.47"/><polygon class="cls-1" points="13.79 10.02 15.65 11.88 13.79 13.73 14.35 14.28 16.75 11.88 14.35 9.47 13.79 10.02"/><rect class="cls-1" x="9.09" y="11.38" width="5.84" height="0.78" transform="translate(-2.74 19.84) rotate(-73.09)"/><path class="cls-1" d="M23.39,10.05l-2.34-.38A8.8,8.8,0,0,0,20,7l1.4-1.94a.76.76,0,0,0-.08-1L19.89,2.76a.75.75,0,0,0-1-.08L17,4.07A9.1,9.1,0,0,0,14.35,3L14,.64A.75.75,0,0,0,13.22,0H10.8a.75.75,0,0,0-.75.63L9.67,3A9,9,0,0,0,7,4.07L5.11,2.68a.76.76,0,0,0-1,.08L2.76,4.13a.77.77,0,0,0-.08,1L4.07,7A9,9,0,0,0,3,9.67l-2.35.38a.75.75,0,0,0-.63.74v2a.75.75,0,0,0,.63.74L3,13.86A9.21,9.21,0,0,0,4.07,16.5L2.68,18.43a.75.75,0,0,0,.08,1l1.38,1.38a.74.74,0,0,0,1,.07L7,19.46a9.06,9.06,0,0,0,2.63,1.1l.38,2.34a.76.76,0,0,0,.75.63h2.43A.74.74,0,0,0,14,22.9l.38-2.34A9.12,9.12,0,0,0,17,19.46l1.93,1.39a.76.76,0,0,0,1-.08l1.36-1.37a.75.75,0,0,0,.08-1L20,16.5a9,9,0,0,0,1.1-2.64l2.34-.38a.74.74,0,0,0,.63-.74v-2A.74.74,0,0,0,23.39,10.05Zm-3,3.13-.06.26a8.12,8.12,0,0,1-1.19,2.85l-.14.23,1.7,2.33-1.34,1.36L17,18.52l-.22.14a8.16,8.16,0,0,1-2.86,1.19l-.26,0-.44,2.85-2.41,0-.46-2.87-.27,0a8.19,8.19,0,0,1-2.85-1.19L7,18.52l-2.32,1.7L3.31,18.89,5,16.52l-.15-.23a8.12,8.12,0,0,1-1.19-2.85l-.05-.26L.78,12.74l0-1.92,2.87-.46.05-.27A8.19,8.19,0,0,1,4.87,7.24L5,7,3.32,4.68,4.65,3.32,7,5l.23-.15a8.19,8.19,0,0,1,2.85-1.19l.27-.05L10.8.78l2.4,0,.47,2.87.26.05a8.16,8.16,0,0,1,2.86,1.19L17,5l2.33-1.71,1.37,1.34L19,7l.14.23a8.19,8.19,0,0,1,1.19,2.85l.06.27,2.84.43,0,1.92Z"/></g></g></svg>
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
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24.02 23.53"><title>apis</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><path class="cls-1" d="M12,5.19a6.58,6.58,0,1,0,6.58,6.58A6.58,6.58,0,0,0,12,5.19Zm0,12.37a5.8,5.8,0,1,1,5.8-5.79A5.8,5.8,0,0,1,12,17.56Z"/><polygon class="cls-1" points="9.68 9.47 7.28 11.88 9.68 14.28 10.23 13.73 8.38 11.88 10.23 10.02 9.68 9.47"/><polygon class="cls-1" points="13.79 10.02 15.65 11.88 13.79 13.73 14.35 14.28 16.75 11.88 14.35 9.47 13.79 10.02"/><rect class="cls-1" x="9.09" y="11.38" width="5.84" height="0.78" transform="translate(-2.74 19.84) rotate(-73.09)"/><path class="cls-1" d="M23.39,10.05l-2.34-.38A8.8,8.8,0,0,0,20,7l1.4-1.94a.76.76,0,0,0-.08-1L19.89,2.76a.75.75,0,0,0-1-.08L17,4.07A9.1,9.1,0,0,0,14.35,3L14,.64A.75.75,0,0,0,13.22,0H10.8a.75.75,0,0,0-.75.63L9.67,3A9,9,0,0,0,7,4.07L5.11,2.68a.76.76,0,0,0-1,.08L2.76,4.13a.77.77,0,0,0-.08,1L4.07,7A9,9,0,0,0,3,9.67l-2.35.38a.75.75,0,0,0-.63.74v2a.75.75,0,0,0,.63.74L3,13.86A9.21,9.21,0,0,0,4.07,16.5L2.68,18.43a.75.75,0,0,0,.08,1l1.38,1.38a.74.74,0,0,0,1,.07L7,19.46a9.06,9.06,0,0,0,2.63,1.1l.38,2.34a.76.76,0,0,0,.75.63h2.43A.74.74,0,0,0,14,22.9l.38-2.34A9.12,9.12,0,0,0,17,19.46l1.93,1.39a.76.76,0,0,0,1-.08l1.36-1.37a.75.75,0,0,0,.08-1L20,16.5a9,9,0,0,0,1.1-2.64l2.34-.38a.74.74,0,0,0,.63-.74v-2A.74.74,0,0,0,23.39,10.05Zm-3,3.13-.06.26a8.12,8.12,0,0,1-1.19,2.85l-.14.23,1.7,2.33-1.34,1.36L17,18.52l-.22.14a8.16,8.16,0,0,1-2.86,1.19l-.26,0-.44,2.85-2.41,0-.46-2.87-.27,0a8.19,8.19,0,0,1-2.85-1.19L7,18.52l-2.32,1.7L3.31,18.89,5,16.52l-.15-.23a8.12,8.12,0,0,1-1.19-2.85l-.05-.26L.78,12.74l0-1.92,2.87-.46.05-.27A8.19,8.19,0,0,1,4.87,7.24L5,7,3.32,4.68,4.65,3.32,7,5l.23-.15a8.19,8.19,0,0,1,2.85-1.19l.27-.05L10.8.78l2.4,0,.47,2.87.26.05a8.16,8.16,0,0,1,2.86,1.19L17,5l2.33-1.71,1.37,1.34L19,7l.14.23a8.19,8.19,0,0,1,1.19,2.85l.06.27,2.84.43,0,1.92Z"/></g></g></svg>
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
