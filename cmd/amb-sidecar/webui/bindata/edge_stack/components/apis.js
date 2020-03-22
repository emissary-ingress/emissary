import { LitElement, html, css } from '../vendor/lit-element.min.js'
import {ApiFetch} from "./api-fetch.js";
import {getCookie} from './cookies.js';

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
    .header_con, .header_con .col {
      display: -webkit-flex;
      display: -ms-flexbox;
      display: flex;
      -webkit-justify-content: center;
      -ms-flex-pack: center;
      justify-content: center
    }
    .header_con {
      margin: 30px 0 0;
      -webkit-flex-direction: row;
      -ms-flex-direction: row;
      flex-direction: row
    }
    .header_con .col {
      -webkit-flex: 0 0 80px;
      -ms-flex: 0 0 80px;
      flex: 0 0 80px;
      -webkit-align-content: center;
      -ms-flex-line-pack: center;
      align-content: center;
      -webkit-align-self: center;
      -ms-flex-item-align: center;
      align-self: center;
      -webkit-flex-direction: column;
      -ms-flex-direction: column;
      flex-direction: column
    }
    .header_con .col svg {
      width: 100%;
      height: 60px
    }
    .header_con .col img {
      width: 100%;
      height: 60px;
    }
    .header_con .col img path {
      fill: #5f3eff
    }
    .header_con .col svg path {
      fill: #5f3eff
    }
    .header_con .col:nth-child(2) {
      -webkit-flex: 2 0 auto;
      -ms-flex: 2 0 auto;
      flex: 2 0 auto;
      padding-left: 20px
    }
    .header_con .col h1 {
      padding: 0;
      margin: 0;
      font-weight: 400
    }
    .header_con .col p {
      margin: 0;
      padding: 0
    }
    .header_con .col2, .col2 a.cta .label {
      -webkit-align-self: center;
      -ms-flex-item-align: center;
      -ms-grid-row-align: center;
      align-self: center
    }
    .logo {
      filter: invert(19%) sepia(64%) saturate(4904%) hue-rotate(248deg) brightness(107%) contrast(101%);
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
    .header_con .col2 a.cta  {
      border-color: #c8c8c8;
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
      if (getCookie("edge_stack_auth")) {  // if user is authenticated start fetching
        // console.log("will reload APIs in 10 seconds");
        setTimeout(this.loadFromServer.bind(this), 10000);
      }  
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
