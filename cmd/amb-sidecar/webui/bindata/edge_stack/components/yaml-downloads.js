import { getCookie } from '../components/cookies.js';
import { css, html } from '../vendor/lit-element.min.js'
import { SingleResource, ResourceSet } from '../components/resources.js';
import { aes_res_editable, aes_res_changed, aes_res_downloaded } from '../components/snapshot.js'

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

  modifiedStyles() {
    return html`
<style>
form div.card {
  margin-top: 10px;
  padding-top: 0;
  padding-bottom: 0;
}
</style>
    `;
  }

  /* renderResource: no content, resources.js will draw titles. */
  renderResource() {
    return html``
  };
}

customElements.define('dw-yaml-item', YAMLItem);


/* Extremely simple ResourceSet subclass to list changed resources. */
export class YAMLDownloads extends ResourceSet {

  // override; this tab is read-only
  readOnly() {
    return true;
  }

  getResources(snapshot) {
    return snapshot.getChangedResources()
  }

  modifiedStyles() {
    return html`
<style>
.yaml-download a.cta {
	width: 120px;
}
.col2 a.cta img {
  width: 15px;
  height: auto
}
</style>
    `;
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
    delete annotations["kubectl.kubernetes.io/last-applied-configuration"];
    delete annotations[aes_res_editable];
    delete annotations[aes_res_changed];

    /* but add the timestamp for download.
     */
    annotations[aes_res_downloaded] = timestamp;

    /* Delete other metadata */
    delete metadata.creationTimestamp;
    delete metadata.generation;
    delete metadata.resourceVersion;
    delete metadata.selfLink;
    delete metadata.uid;

    /* Delete the resource status. */
    delete resource.status;

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
    let res_yml = this.resources.map((res) => {
      res = this.cleanResource(res, timestamp)
      return "---\n" + jsyaml.safeDump(res)
    });

    /* Write back to Kubernetes, with change flag cleared
     * and a download timestamp.
     */
    this.resources.map((res) => {
      this.applyResource(res, timestamp)
    });

    /* Write out a single file with all the changed resources */
    let blob = new Blob(res_yml, {type: "text/plain;charset=utf-8"});
    saveAs(blob, "resources.yml");

    /* update the page -- changed resources should disappear... */
    this.requestUpdate()
  }

  renderInner() {
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
        ? "Changed resources to download to your configuration-as-code source files"
        : "No changed resources";

    /* The HTML: */
    return html`
<div class="header_con yaml-download">
  <div class="col">
    <?xml version="1.0" encoding="utf-8"?>
<!-- Generator: Adobe Illustrator 24.0.0, SVG Export Plug-In . SVG Version: 6.00 Build 0)  -->
      <img alt="yaml-downloads logo" class="logo" src="../images/svgs/yaml-downloads2.svg"></img>
  </div>
  <div class="col">
    <h1>Download YAML</h1>
    <p>${changed_title}</p>
  </div>
  <div class="col2">
    <a class="cta download ${count > 0 ? "" : "off"}" @click=${()=>this.doDownload()}>
      <img alt="yaml-downloads logo" src="../images/svgs/yaml-downloads.svg">
        <g id="Layer_2" data-name="Layer 2">
          <g id="iconmonstr"></g>
        </g>
      </img>
      <div class="label">download</div>
    </a>
  </div>
</div>

    <dw-yaml-item .resource=${newItem} .state=${this.addState}>
    </dw-yaml-item>
    
    <div>
    ${this.resources.map(r => {
    return html`<dw-yaml-item .resource=${r} .state=${this.state(r)}></dw-yaml-item>`
    })}
    </div>    
`
  }
}

customElements.define('dw-yaml-dl', YAMLDownloads);
