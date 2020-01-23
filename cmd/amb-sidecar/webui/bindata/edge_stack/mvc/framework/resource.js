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
import { mapMerge, objectCopy } from "../framework/utilities.js"

/* Current API version. */
const aes_api_version = "getambassador.io/v2";

/* Annotation key for sourceURI. */
const aes_res_source = "getambassador.io/resource-source";

export class Resource extends Model {

  /* constructor()
   * Here the model initializes any internal state that is common to all Resources.
   * Typically a concrete Resource class would initialize the Resource kind, name, namespace,
   * and other useful state to be maintained in the Resource instance.
   */

  constructor(yaml) {
    /* Define the instance variables that are part of the model. Views and other Resource users will access
     * these for rendering and modification.  All resource objects have a kind, a name, and a namespace, which
     * together are a unique identifier throughout the Kubernetes system.  They may also have annotations,
     * labels, and a status, which are also saved as object state.
    */

    /* calling Model.constructor() */
    super();

    /* Set up our instance variables, including default values if needed. */
    this.updateFrom(yaml);

    /* Internal state for when the Resource is edited and is pending confirmation of the edit from a future snapshot.
    *  Different operations may be pending (e.g. add, save)
    */
    this._pending = new Map();
  }

  /* copySelf()
   * Create a copy of the Resource, with all Resource state (but not Model's listener list}
   */

  copySelf() {
    return new Resource(this._fullYAML);
  }

  /* applyYAML(yaml)
   * call the edge_stack API to apply the object's current YAML.  Returns null if success, an error string if not.
    */

  applyYAML(yaml) {
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

  /* doDelete()
   * call the edge_stack API to delete this Resource.  Returns null if success, an error string if not.
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
   * call the edge_stack API to save the changes in this Resource.  Returns null if success, an error string if not.
   */

  doSave() {
    let yaml  = this.getYAML();
    let error = this.applyYAML(yaml);

    if (error) {
      this.addMessage(error);
      console.log('Model failed to apply changes: ${error}');
    }

    return error;
  }

  /* getYAML()
   * Return YAML that has the Resource's values written back into the _fullYAML, and has been pruned so that only
   * the necessary attributes exist in the structure for use as the parameter to applyYAML().  Subclasses will
   * call super.getYAML() to fill out most of the YAML, and will only need to write those parts of the YAML
   * that the subclass requires.
   */
  getYAML() {
    /* Make a copy of our full YAML, stripping out extraneous attributes */
    let yaml  =  this.yamlStrip(this._fullYAML, this.yamlIgnorePaths());

    /* Write back our editable values for updating with apply. */
    yaml.apiVersion = aes_api_version;
    yaml.kind = this.kind;
    yaml.metadata.name = this.name;
    yaml.metadata.namespace = this.namespace;

    return yaml;
  }

  /* getEmptyStatus()
   * Utility method for initializing the status of the resource.  Returns a dictionary that has the basic
   * structure of the status attribute in the Kubernetes resource structure.  This is simply a dictionary
   * with state = "none" and an empty reason string.
   */

  getEmptyStatus() {
    return {
      "state":  "none",
      "errorReason": ""
    };
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

  /* updateFrom(yaml)
   * Update the Resource object state from the snapshot data block for this Resource.  Compare the values in the
   * data block with the stored state in the Resource.  If the data block has different data than is currently
   * stored, update that instance variable with the new data and set a flag to indicate an update has been made.
   * If any of the state has changed, notify listeners.
   */

  updateFrom(yaml) {
    let updated = false;

    /* In the case of incomplete yaml -- such as when adding a new Resource -- set proper defaults. */
    yaml.kind                     = yaml.kind                      || "<must specify resource kind in constructor>";
    yaml.metadata                 = yaml.metadata                  || {};
    yaml.metadata.name            = yaml.metadata.name             || "<specify resource name here>";
    yaml.metadata.namespace       = yaml.metadata.namespace        || "default";
    yaml.metadata.resourceVersion = yaml.metadata.resourceVersion  || "0";
    yaml.metadata.labels          = yaml.metadata.labels           || {};
    yaml.metadata.annotations     = yaml.metadata.annotations      || {};
    yaml.status                   = yaml.status                    || this.getEmptyStatus();

    /* Copy back to our instance variables */
    this.kind        = yaml.kind;
    this.name        = yaml.metadata.name;
    this.namespace   = yaml.metadata.namespace;

    /* Since we are being updated, we know that our version is out of date; get the new version value. */
    this.version = yaml.metadata.resourceVersion;

    /* get the new labels value from the data, or an empty object if undefined. */
    let new_labels = yaml.metadata.labels || {};

    if (this.labels !== new_labels) {
      this.labels = new_labels;
      updated = true;
    }

    /* get the new annotations value from the data, or an empty object if undefined. */
    let new_annotations = yaml.metadata.annotations || {};

    if (this.annotations !== new_annotations) {
      this.annotations = new_annotations;
      updated = true;
    }

    /* get the new status value from the data, or the emptyStatus object if undefined.  Must initialize our
     * own status as empty initially, since it is being dereferenced in the update check below. */
    let new_status = yaml.status || this.getEmptyStatus();
    this.status    = this.status || this.getEmptyStatus();

    if ((this.status.state       !== new_status.state) ||
        (this.status.errorReason !== new_status.errorReason)) {
      this.status = new_status;
      updated = true;
    }

    /* Give subclasses a chance to update themselves. */
    updated = this.updateSelfFrom(yaml) || updated;

    /* Remember the full YAML for merging later, to send to Kubernetes. */
    this._fullYAML = yaml;

    /* Clear any pending flags, since the resource has now been updated. */
    this.clearAllPending();

    /* Notify listeners if any updates occurred. */
    if (updated) {
      this.notifyListenersUpdated();
    }
  }


  /* clearAllPending()
   * Clear all pending flags.
   */

  clearAllPending() {
    this._pending = new Map();
  }

  /* clearPending(operation)
   * Clear the pending flag for the given operation.
   */

  clearPending(op) {
    this._pending[op] = false;
  }

  /* setPending(operation)
   * Set the pending flag for the given operation.
   */

  setPending(op) {
    this._pending[op] = true;
  }

  /* pending(ops)
   * Return whether the Resource is pending any of a set of operations after adding or editing.  This is used for
   * rendering the Resource differently in the View if the current state in the Resource object has been modified,
   * and not yet resolved from a snapshot.  Typical call would be myResource.pending("add", "delete", "save").
   */

  pending() {
    for (let op of [...arguments]) {
      if (this._pending[op] === true) {
        return true;
      }
    }

    return false;
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

    return mapMerge(errors, this.validateSelf());
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

  /* yamlIgnorePaths()
  * Return an array of paths arrays to be ignored when sending YAML to Kubernetes.  This is needed because Kubernetes
  * sends extra information in the Resource object that confuses it when sent back; only the data that is needed
  * (e.g. name, namespace, kind, and desired labels/annotations/spec) should be sent back.
  *
  * NOTE: one would think that a full path could be described by a string with the path delimiter "."
  * to separate the path elements.  BUT, Kubernetes allows keys in the YAML to use the same delimiter,
  * so we have to have arrays of path elements.  e.g. you can't parse at "." to get the full path for
  * "metadata.annotations.kubectl.kubernetes.io/last-applied-configuration"
  * because it is really
  * "metadata"."annotations"."kubectl.kubernetes.io/last-applied-configuration"
  */

  yamlIgnorePaths() {
    return [
      ["status"],
      ["metadata", "uid"],
      ["metadata", "selfLink"],
      ["metadata", "generation"],
      ["metadata", "resourceVersion"],
      ["metadata", "creationTimestamp"],
      ["metadata", "annotations", "kubectl.kubernetes.io/last-applied-configuration"]
      ];
  }

  /* yamlStrip(originalYaml, pathsToIgnore)
   * Given an object, remove the subobjects named in the pathsToIgnore array.
   */

  yamlStrip(originalYAML, pathsToIgnore) {
    /* Clone the existing YAML.  This does a "deep copy" of the object, recursively copying subtrees. */
    let cleanYaml = objectCopy(originalYAML);

    /* For each path to ignore, remove it from the cleanYaml */
    for (let pathElements of pathsToIgnore) {
      /* Traverse down the YAML tree, starting at the top.  node will traverse down the child path,
       * checking for existence of the attribute at that point in the tree.
       */
      let parent = cleanYaml;
      let elementParent = undefined;
      let elementName   = "";

      /* Walk down the path to make sure the attribute at that path exists. */
      for (let element of pathElements) {
        let child = parent[element];
        if (child === undefined) {
          elementParent = undefined;
          break;
        }
        else {
          /* Remember the last element in the path, and its parent object */
          elementParent = parent;
          elementName   = element;

          /* Traverse down the tree */
          parent = child;
        }
      }
      /* Was the traverse through the path successful?  Is there an object and a defined attribute
       * at this point in the path?
       */
      if (elementParent !== undefined) {
        delete elementParent[elementName];
      }
    }

    return cleanYaml;
  }


}

