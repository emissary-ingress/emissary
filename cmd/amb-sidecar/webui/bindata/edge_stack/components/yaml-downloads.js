import {css, html} from '/edge_stack/vendor/lit-element.min.js'
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

  /* CleanResource:
   * remove selected properties from the resource that are
   * Kubernetes-internal or otherwise not gitOps friendly.
   * metadata:
   *   creationTimestamp:...
   *   generation:...
   *   resourceVersion:...
   *   selfLink:...
   *   uid:...
   *   status:...
   *   annotations:
   *    kubectl.kubernetes.io/last-applied-configuration:...
   *    aes_res_changed: '2019-11-30T04:40:02.132Z'
   *    aes_res_downloaded:...
   */

  cleanResource(resource) {
    let metadata    = resource.metadata;
    let annotations = metadata.annotations;

    /* Delete specific annotations: */
    delete annotations["kubectl.kubernetes.io/last-applied-configuration"]
    delete annotations.aes_res_changed
    delete annotations.aes_res_downloaded

    /* Delete other metadata */
    delete metadata.creationTimestamp
    delete metadata.generation
    delete metadata.resourceVersion
    delete metadata.selfLink
    delete metadata.uid

    /* Delete the resource status. */
    delete resource.status

    /* Return the cleaned resource. Note that this has changed
     * the original resource, so we expect a new snapshot to
     * restore all the previous values from the server.
     */
    return resource
  }

  /* Download the resources listed in the YAML Downloads tab.
   * When done, reset the annotations in each resource:
   * set aes_res_changed to "false" and
   * set aes_res_downloaded to "true".
   */
  doDownload() {
    console.log("Clicked on doDownload")

    /* dump each resource as YAML */
    var res_yml = this.resources.map((res) => {
      res = this.cleanResource(res)
      return "---\n" + jsyaml.safeDump(res)
    })

    /* Write out a single file with all the changed resources */
    var blob = new Blob(res_yml, {type: "text/plain;charset=utf-8"});
    saveAs(blob, "resources.yml");

    /* Tell Kubernetes to reset the aes_res_changed to false
     * for each resource that we wrote out.
     */

    /* this.resources.map((res) => {
      this.applyResChanges(res, )
    }) */
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
    
    ${count > 0 ?
      html`<div align="center">
        <button @click=${this.doDownload.bind(this)} style="display:"block" id="click_to_dl">
        Download ${count} changed resources
        </button>
        </div>` : html``}
`
  }
}

customElements.define('dw-yaml-dl', YAMLDownloads);
