/**
 * Resource
 * This is the Resource class, the base class for all Resources and the superclass for IResource.  New Resource
 * classes will subclass from IResource and implement only those methods that are defined in the IResource interface;
 * the methods here are private methods that need not be reimplemented but can be used through inheritance in
 * any subclasses that require them.  For example, a new Resource subclass may need to extend the validation
 * method but would like to use the existing Resource validate() to handle basic validation; this can be done by
 * explicitly calling Resource.validate() from the subclass's validate() method.
 *
 *
 * This class is the basis for all Kubernetes resources that are created, viewed, modified and deleted in the Web UI.
 */

/* For getting the edge-stack authorization. */
import {getCookie} from '../../components/cookies.js';

/* Kubernetes operations: apply, delete. */
import { ApiFetch } from "../../components/api-fetch.js";

/* Interface class for Model */
import { Model } from "./model.js"

/* Object merge operation */
import { objectMerge } from "../framework/utilities.js"

/* Annotation key for sourceURI. */
const aes_res_source = "getambassador.io/resource-source";

export class Resource extends Model {

  /* constructor()
   * Here the model initializes any internal state that is common to all Resources.
   * Typically a concrete Resource class would initialize the Resource kind, name, namespace,
   * and other useful state to be maintained in the Resource instance.
   */

  constructor(resourceData) {
    /* Define the instance variables that are part of the model. Views and other Resource users will access
     * these for rendering and modification.  All resource objects have a kind, a name, and a namespace, which
     * together are a unique identifier throughout the Kubernetes system.  They may also have annotations,
     * labels, and a status, which are also saved as object state.
    */

    /* calling Model.constructor() */
    super();

    /* If resourceData does not include metadata, set it to the empty object so that the name, namespace,
     * resourcevVersion, labels and annotation fields will be set to their default values during initialization.
     * Otherwise javascript would fail, trying to access a field of "null"
     */
    resourceData.metadata = resourceData.metadata || {};

    /* Initialize instance variables from resourceData.  For those fields that are unknown,
     * initialize to default values.  This is when adding a new Resource whose values will be
     * specified by the user.
     */
    this.kind        = resourceData.kind                      || "<must specify resource kind in constructor>";
    this.name        = resourceData.metadata.name             || "<resource name>";
    this.namespace   = resourceData.metadata.namespace        || "default";
    this.version     = resourceData.metadata.resourceVersion  || "0";
    this.labels      = resourceData.metadata.labels           || {};
    this.annotations = resourceData.metadata.annotations      || {};
    this.status      = resourceData.status                    || this.getEmptyStatus();

    /* Internal state for when the Resource is edited and is pending confirmation of the edit
     * from a future snapshot.
     */
    this._pending = false;
  }

  /* copySelf()
   * Create a copy of the Resource, with all Resource state (but not Model's listener list}
   */

  copySelf() {
    return new Resource(this.getYAML());
  }

  /* doApply(yaml, cookie)
   * call the edge_stack API to apply the object's current state
   * as YAML.  Returns null if success, an error string if not.
    */

  doApply(yaml) {
    let cookie = getCookie("edge_stack_auth");
    let error  = null;

    let params = {
      method: "POST",
      headers: new Headers({'Authorization': 'Bearer ' + cookie}),
      body: JSON.stringify(yaml)
    };

    /* Make the call to apply */
    ApiFetch('/edge_stack/api/apply', params).then(
      r => { r.text().then(t => {
        if (r.ok) {
          error = null;
        } else {
          error = t;
          console.error(error);
          error = `Unable to complete add or save resource because: ${error}`;
        }
      });
      });

    return error;
  }

  /* doAdd()
   * Add this Resource to Kubernetes using kubectl apply.
   */

  doAdd() {
    throw Error("Not Yet Implemented");

    let cookie = getCookie("edge_stack_auth");

    /* Note that we are pending confirmation from Kubernetes, and expect to see it in a future snapshot. */
    this._pending = true;

  }

  /* doDelete()
  * call the edge_stack API to delete this object.
  * Returns null if success, an error string if not.
   */

  doDelete() {
    let cookie = getCookie("edge_stack_auth");
    let error  = null;
    let params = {
      method: "POST",
      headers: new Headers({ 'Authorization': 'Bearer ' + cookie }),
      body: JSON.stringify({
        Namespace: this.namespace,
        Names: [`${this.kind}/${this.name}`]
      })
    };

    ApiFetch('/edge_stack/api/delete', params).then(
      r=>{
        r.text().then(t=>{
          if (r.ok) {
            error = null;
          } else {
            error = t;
            console.error(error);
            error = `Unexpected error while deleting resource: ${r.statusText}`;
          }
        });
      });

    return error;
  }

  /* doSave()
   * Save the changes in this Resource to Kubernetes using kubectl apply.
   */

  doSave() {
    throw Error("Not Yet Implemented");

    let cookie = getCookie("edge_stack_auth");

    /* Note that we are pending confirmation from Kubernetes, and expect to see it in a future snapshot. */
    this._pending = true;

  }

  /* getEmptyStatus()
   * Utility method for initializing the status of the resource.  Returns a dictionary that has the basic
   * structure of the status attribute in the Kubernetes resource structure.  This is simply a dictionary
   * with state = "none" and an empty reason string.
   */

  getEmptyStatus() {
    return {
      "state":  "none",
      "reason": ""
    };
  }

  /* getYAML()
  * Return the YAML object to JSON.stringify for the implementation of the Save function which uses kubectl apply.
  * Like getSpec, this method must return an object to be serialized and supplied to kubectl apply.  This requires
  * getSpec to be implemented
  */
  getYAML() {
    return {
      apiVersion: "getambassador.io/v2",
      kind: this.kind,
      metadata: {
        name:            this.name,
        namespace:       this.namespace,
        labels:          this.labels,
        annotations:     this.annotations,
        resourceVersion: this.version
      },
      spec: this.getSpec()
    }
  }

  /* sourceURI()
   * Return the source URI for this resource, if one exists.  IN the case we have a source URI, the view may provide
   * a button which, when clicked, opens a window on that source URI.  This is useful for tracking resource as they
   * are applied using GitOps, though an annotation specifying the sourceURI must be applied in the GitOps pipeline
   * in order for this to have a value.  If there is no sourceURI, return undefined.
   */
  sourceURI() {
    /* Make sure we have annotations, and return the aes_res_source, or undefined */
    let annotations = this.annotations;
    if (aes_res_source in annotations) {
      return annotations[aes_res_source];
    } else {
      /* Return undefined (same as nonexistent property, vs. null) */
      return undefined;
    }
  }

  /* updateFrom(resourceData)
   * Update the Resource object state from the snapshot data block for this Resource.  Compare the values in the
   * data block with the stored state in the Resource.  If the data block has different data than is currently
   * stored, update that instance variable with the new data and set a flag to indicate an update has been made.
   * If any of the state has changed, notify listeners.
   */

  updateFrom(resourceData) {
    let updated = false;

    /* Since we are being updated, we know that our version is out of date; get the new version value. */
    this.version = resourceData.metadata.resourceVersion;

    /* get the new labels value from the data, or an empty object if undefined. */
    let new_labels = resourceData.metadata.labels || {};

    if (this.labels !== new_labels) {
      this.labels = new_labels;
      updated = true;
    }

    /* get the new annotations value from the data, or an empty object if undefined. */
    let new_annotations = resourceData.metadata.annotations || {};

    if (this.annotations !== new_annotations) {
      this.annotations = new_annotations;
      updated = true;
    }

    /* get the new status value from the data, or the emptyStatus object if undefined. */
    let new_status = resourceData.status || this.getEmptyStatus();

    if ((this.status.state  !== new_status.state) ||
      (this.status.reason !== new_status.reason) ||
      (this.status.errorReason !== new_status.errorReason)) {
      this.status = new_status;
      updated = true;
    }

    /* Give subclasses a chance to update themselves. */
    updated = updated || this.updateSelfFrom(resourceData);

    /* Notify listeners if any updates occurred. */
    if (updated) {
      this.notifyListenersUpdated();
    }
  }


  /* updatePending()
   * Return whether the Resource is pending an update after doSave().  This is used for rendering the
   * Resource differently in the View if the current state in the Resource object has been added or edited and not
   * yet resolved from a snapshot.
   */

  updatePending() {
    return this._pending;
  }

  /* validate()
   * Validate this Resource's state by checking each object instance variable for correctness (e.g. email address
   * format, URL format, date/time, name restrictions).  Returns a dictionary of property: errorString if there
   * are any errors. If the dictionary is empty, there are no errors.
   *
   * In a basic Resource, only the name and namespace is checked.
   */

  validate() {
    let errors  = new Map();
    let message = "";

    /* Perform basic validation.  This can be extended by subclasses that implement validateSelf() */
    message = this.validateName(this.name);
    if (message) errors.set("name", message);

    message = this.validateName(this.namespace);
    if (message) errors.set("namespace", message);

    /* Any errors from self validation? Merge the results of validateSelf with the existing results from above.
     * validateSelf() overrides any errors returned above with the same name (i.e. name or namespace)
     */

    return objectMerge(errors, this.validateSelf());
  }

  /* ============================================================
   * Utility methods -- Validation
   * ============================================================
   */

  /* validateName(name)
   * name and namespaces rules as defined by
   * https://kubernetes.io/docs/concepts/overview/working-with-objects/names/\
   * returns null if name is valid, error string if not.
   */

  validateName(name) {
    // lower-case letters, numbers, dash, and dot allowed.
    let format = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$/;
    if (name.match(format) && name.length <= 253) {
      return null;
    } else {
      return "Name must be {a-z0-9-.}, length <= 253";
    }
  }


  /* validateEmail(email)
   * We validate that the user has provided a plausible looking email address. In the future, we should actually
   * validate that it's a real email address using something like
   * https://www.textmagic.com/free-tools/email-validation-tool
   * with an appropriate fallback if we are unable to reach outside the firewall (if we can't reach the outside
   * system, then use simple pattern matching).
   *
   * returns null if email is valid, error string if not.
   *
   * TODO: Implement real email validation.
   */

  validateEmail(email) {
    return null;
  }

  /* _validateURL(url)
  * returns null if url is valid, error string if not.
  *
  * TODO: Implement real URL validation.
  */

  validateURL(url) {
      return null;
    }
}

