import { LitElement, html, css } from 'https://cdn.pika.dev/-/lit-element/2.2.1/dist-es2019/lit-element.min.js'
import { repeat } from '/edge_stack/components/repeat.js';

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
    super()
    this.reset()
  }

  reset() {
    this.apis = []
    this.details = {}
    // this.message = "Initial Message Prop"
  }

  connectedCallback() {
    super.connectedCallback()
    console.log("APIs doing initial load")
    this.loadFromServer()
  }

  loadFromServer() {
    $.getJSON("/openapi/services", apis => {
      console.log("APIs load succeeded")
      console.log(apis)
      this.apis = apis
    }).fail(xhr=>{
      if (xhr.status == 401 || xhr.status == 403) {
        window.location.replace("../login/")
      }
      else {
        console.log(xhr)
      }
    })

    console.log("will reload APIs in 10 seconds")
    setTimeout(this.loadFromServer.bind(this), 10000)
  }

  render() {
    console.log(`APIs rendering ${this.apis.length} API${(this.apis.length == 1) ? "" : "s"}`)

    if (this.apis.length == 0) {
      return html`
<div id="apis-div">
  <p>No API documentation is available.</p>
</div>
`
    }
    else {
      var rendered = []

      this.apis.forEach((api, index) => {
        rendered.push(this.renderAPIDocs(api, index))
      })

      console.log(rendered)

      return html`
<div id="apis-div">
  <div>APIs with documentation:</div>
  ${rendered}
</div>
`
// ${repeat(this.apis, (api) => this.apiKey(api), (api, idx) => this.renderAPIDocs(api, idx))}

    }
  }
  
  apiKey(api) {
    let apiName = `${api.service_name}.${api.service_namespace}`

    return `api-${apiName}`
  }

  renderAPIDocs(api, index) {
    let apiName = `${api.service_name}-${api.service_namespace}`

    return html`
<div id=${this.apiKey(api)}>
  ${this.renderAPIDetails(api, apiName)}
</div>
`
  }

  renderAPIDetails(api, apiName) {
    if (api.has_doc) {
      let detailKey = `${this.apiKey(api)}-details`
      let detailURL = `/openapi/services/${api.service_namespace}/${api.service_name}/openapi.json`

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

customElements.define('dw-apis', APIs)
