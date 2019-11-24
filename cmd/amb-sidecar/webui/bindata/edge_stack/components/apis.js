import { LitElement, html, css } from 'https://cdn.pika.dev/-/lit-element/2.2.1/dist-es2019/lit-element.min.js'

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
      // message: { type: String }
    }
  }

  constructor() {
    super();
    this.reset();
    this.doRefresh = true;             // true to allow auto-refreshing
    this.waitingForHackStyles = false   // true while we have a deferred hackStyles call
  }

  reset() {
    this.apis = [];
    this.details = {}
  }

  connectedCallback() {
    super.connectedCallback();
    console.log("APIs doing initial load");
    this.loadFromServer()
  }

  loadFromServer() {
    if(1) {//TODO temporarily removed because it was filling the console with error messages making it hard to debug other problems MOREMORE
      return {} //TODO part of the temporary removal MOREMORE
    } else {//TODO part of the temporary removal MOREMORE

    $.getJSON("/openapi/services", apis => {
      console.log("APIs load succeeded");
      console.log(apis);
      this.apis = apis
    }).fail(xhr=>{
      if (xhr.status === 401 || xhr.status === 403) {
        window.location.replace("../login/")
      }
      else {
        console.log(xhr)
      }
    });

    if (this.doRefresh) {
      console.log("will reload APIs in 10 seconds");
      setTimeout(this.loadFromServer.bind(this), 10000)
    }
    }//TODO part of the temporary removal MOREMORE
  }

  deferHackStyles() {
    if (!this.waitingForHackStyles) {
      this.waitingForHackStyles = true;
      // console.log("DEFERRING HACKSTYLES")
      setTimeout(this.hackStyles.bind(this), 1)      
    }
    // else {
    //   console.log("ALREADY DEFERRING HACKSTYLES")
    // }
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

    // console.log("HACKSTYLES", api/Divs)

    for (let i = 0; i < apiDivs.length; i++) {
      let aDiv = apiDivs[i];

      // console.log("HACKING", aDiv)

      var needLinks = false;

      while (true) {
        if (!aDiv.shadowRoot) {
          // Reschedule. FFS.
          // console.log("NOT YET READY")
          this.deferHackStyles();
          return
        }

        let aDivChildren = aDiv.shadowRoot.children;
        // console.log("  CHILDREN", aDivChildren)

        if (!aDivChildren) {
          break
        }

        let aKid = aDivChildren[0];
        // console.log("  CHILD0", aKid)

        if (aKid.tagName === 'STYLE') {
          // console.log("  SMITE")
          aKid.remove();
          needLinks = true
        }
        else {
          break
        }
      }

      if (needLinks) {
        console.log("  ADDLINKS");

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

    return `${apiName}`
  }

  compareAPIs(api1, api2) {
    let name1 = this.apiName(api1);
    let name2 = this.apiName(api2);

    return name1.localeCompare(name2)
  }

  render() {
    window.apis = this;
    console.log(`APIs rendering ${this.apis.length} API${(this.apis.length === 1) ? "" : "s"}`);

    if (this.apis.length === 0) {
      return html`
<div id="apis-div">
  <p>No API documentation is available.</p>
</div>
`
    }
    else {
      this.deferHackStyles();

      var rendered = [];
      
      this.apis.sort(this.compareAPIs.bind(this));

      this.apis.forEach((api, index) => {
        rendered.push(this.renderAPIDocs(api, index))
      });

      return html`
<div id="apis-div">
  <div>APIs with documentation:</div>
  ${rendered}
</div>
`
// ${repeat(this.apis, (api) => this.apiKey(api), (api, idx) => this.renderAPIDocs(api, idx))}
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
      return ''
    }
  }
}

customElements.define('dw-apis', APIs);
