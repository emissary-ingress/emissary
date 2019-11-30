import {css, html} from '/edge_stack/vendor/lit-element.min.js'
//import { saveAs } from '/edge_stack/vendor/FileSaver.min.js'
import {SingleResource, ResourceSet} from '/edge_stack/components/resources.js';


/* Extremely simple SingleResource subclass to list resource items. */
class YAMLItem extends SingleResource {
  /** Return the kind of this specific resource. */
  kind() {
    let res = this.resource;
    return ("kind" in res ? res.kind : "")
  };

  /* Don't allow editing, since we're just listing the resources. */
  readOnly() {
    return true;
  };

  /* renderResource: no content, resources.js will draw titles. */
  renderResource() {
    return html``
  };
}

customElements.define('dw-yaml-item', YAMLItem);


/* Extremely simple ResourceSet subclass to list changed resources. */
export class YAMLDownloads extends ResourceSet {

    /* styles() returns the styles for the YAML downloads list. */
  static saveCSS = css`
    .section-heading {
      margin: 0.1em;
      font-size: 120%;
      font-weight: bold;
      margin-top: 0;
    }
`;

  static get styles() {
    return this.saveCSS
  };

  getResources(snapshot) {
    return snapshot.getChangedResources()
  }

  /* Download the resources listed in the YAML Downloads tab.
   * When done, reset the annotations in each resource:
   * set aes_res_changed to "false" and
   * set aes_res_downloaded to "true".
   */
  doDownload() {
    console.log("Clicked on doDownload")

    for (const res of this.resources) {
      console.log("============")
      console.log(res.getYaml())
    }

    // var blob = new Blob(["Hello, world!"], {type: "text/plain;charset=utf-8"});
    // saveAs()


  }

  render() {
    /* Template for ResourceSet*/
    let newItem = {
      metadata: {
        namespace: "default",
        name: ""
      },
      spec: {
        prefix: "",
        service: ""
      }
    };

    let count = this.resources.length;

    /* Title depending on whether there are changes to download. */
    let changed_title =
      count > 0
        ? "Changed Resources to download:"
        : "No Changed Resources";

    /* The HTML: */
    return html`
    <div class="section-heading">${changed_title}</div>
    <dw-yaml-item .resource=${newItem} .state=${this.addState}>
    </dw-yaml-item>
    
    <div>
    ${this.resources.map(r => {
    return html`<dw-yaml-item .resource=${r} .state=${this.state(r)}></dw-yaml-item>`
    })}
    </div>
    
    <div align="center">
    <button @click=${this.doDownload} style="display:"block">
    Download ${count} changed resources
    </button>
    </div>
`
  }
}

customElements.define('dw-yaml-dl', YAMLDownloads);
