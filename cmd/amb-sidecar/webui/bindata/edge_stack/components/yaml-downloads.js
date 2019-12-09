import { getCookie } from '/edge_stack/components/cookies.js';
import { css, html } from '/edge_stack/vendor/lit-element.min.js'
import { SingleResource, ResourceSet } from '/edge_stack/components/resources.js';
import { aes_res_editable, aes_res_changed, aes_res_downloaded } from '/edge_stack/components/snapshot.js'
//MOREMORE do the new look for the YAML page

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

  static get styles() {
    return css`
    .section-heading {
      margin: 0.1em;
      font-size: 120%;
      font-weight: bold;
      margin-top: 0;
    }`
  };

  getResources(snapshot) {
    return snapshot.getChangedResources()
  }

  /* cleanResource:
   * remove selected properties from the resource that are
   * Kubernetes-internal or otherwise not gitOps friendly.
   */
  cleanResource(resource, timestamp) {
    let metadata    = resource.metadata;
    let annotations = metadata.annotations;

    /* Delete specific annotations.  Note the semantics of
     * delete allow the attempted deletion of properties that
     * do not exist (in which case true is returned) so no
     * exception handling is needed here.
     */
    delete annotations["kubectl.kubernetes.io/last-applied-configuration"]
    delete annotations.aes_res_editable
    delete annotations.aes_res_changed

    /* but add the timestamp for download.
     */
    annotations.aes_res_downloaded = timestamp

    /* Delete other metadata */
    delete metadata.creationTimestamp
    delete metadata.generation
    delete metadata.resourceVersion
    delete metadata.selfLink
    delete metadata.uid

    /* Delete the resource status. */
    delete resource.status

    /* Return the cleaned resource. Note that this has changed
     * the original resource object, so we expect a new snapshot
     * to restore all the previous values from the server.
     */
    return resource
  }

  /* Tell Kubernetes that the resource has changed the
     aes_res_changed and aes_res_downloaded annotations.
   */
  applyResource(resource, timestamp) {
    let yaml = `
---
apiVersion: getambassador.io/v2
kind: ${resource.kind}
metadata:
  name: "${resource.metadata.name}"
  namespace: "${resource.metadata.namespace}"
  annotations:
    ${aes_res_changed}: "false"
    ${aes_res_downloaded}: "${timestamp}"
spec: ${JSON.stringify(resource.spec)}
`;

    fetch('/edge_stack/api/apply',
          {
            method: "POST",
            headers: new Headers({
              'Authorization': 'Bearer ' + getCookie("edge_stack_auth")
            }),
            body: yaml
          })
      .then(r=>{
        r.text().then(t=>{
          if (r.ok) {
            // happy path
          } else {
            console.error(t);
            this.addError(`Unexpected error while updating resource annotations: ${r.statusText}`); // Make sure we add this error to the stack after calling this.reset();
          }
        })
      })
  }

  /* Download the resources listed in the YAML Downloads tab.
   * When done, reset the annotations in each resource:
   * set aes_res_changed to "false" and
   * set aes_res_downloaded to "true".
   */
  doDownload() {
    let timestamp = new Date().toISOString();

    /* dump each resource as YAML */
    var res_yml = this.resources.map((res) => {
      res = this.cleanResource(res, timestamp)
      return "---\n" + jsyaml.safeDump(res)
    })

    /* Write back to Kubernetes, with change flag cleared
     * and a download timestamp.
     */
    this.resources.map((res) => {
      this.applyResource(res, timestamp)
    })

    /* Write out a single file with all the changed resources */
    var blob = new Blob(res_yml, {type: "text/plain;charset=utf-8"});
    saveAs(blob, "resources.yml");

    /* update the page -- changed resources should disappear... */
    this.requestUpdate()
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
        Download ${count} changed
        </button>
        </div>` : html``}
    
`
  }
}

customElements.define('dw-yaml-dl', YAMLDownloads);
