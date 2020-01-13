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
import { objectMerge, setUnion } from "../framework/utilities.js"

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
    this.name        = resourceData.metadata.name             || "";
    this.namespace   = resourceData.metadata.namespace        || "default";
    this.version     = resourceData.metadata.resourceVersion  || "0";
    this.labels      = resourceData.metadata.labels           || {};
    this.annotations = resourceData.metadata.annotations      || {};
    this.status      = resourceData.status                    || this.getEmptyStatus();

    /* Internal state for when the Resource is edited and is pending confirmation of the edit from a future snapshot. */
    this._pendingUpdate = false;
  }

  /* computeYAMLMerge(other)
    * Compare this Resource's YAML with another's. Return a structure with a new YAML object, and a difference set.
    * { yaml: <the new merged YAML, preserving this>, diffs: { <delta between original and other> } }
    */

  computeYAMLMerge(other) {
    let delta    = new Map();
    let original = this.getYAML();
    let changed  = other.getYAML();
    let merged   = this._mergeObject(original, changed, delta);

    return { yaml: merged, diffs: delta }
   }

  /* copySelf()
   * Create a copy of the Resource, with all Resource state (but not Model's listener list}
   */

  copySelf() {
    return new Resource(this.getYAML());
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

  /* doAdd()
   * Call the edge_stack API to add this Resource. Returns null if success, an error string if not.
   */

  doAdd() {
    let cookie = getCookie("edge_stack_auth");
    let error  = null;

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
      console.log('Model failed to apply changes: ${applyErr}');
      return false;
    }

    return error;
  }

  /* getApplyYAML()
   * Return YAML that has been pruned so that only the necessary attributes exist in the structure for use
   * as the parameter to applyYAML().
   */
  getApplyYAML() {
    /* for now, return everything */
    return this.getYAML();
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


  /* clearPendingUpdate()
  * Clear the pendingUpdate flag.
  */

  clearPendingUpdate() {
    return this._pendingUpdate = false;
  }

  /* setPendingUpdate()
  * set the pendingUpdate flag.
  */

  setPendingUpdate() {
    return this._pendingUpdate = true;
  }

  /* pendingUpdate()
    * Return whether the Resource is pending an update after doSave().  This is used for rendering the
    * Resource differently in the View if the current state in the Resource object has been added or edited and not
    * yet resolved from a snapshot.
    */

  pendingUpdate() {
    return this._pendingUpdate;
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

  /* ============================================================
   * Private methods -- Merging YAML
   * ============================================================
   */

  /* _yamlMergeStrategy(pathName)
    * Given a pathName, this method returns whether to ignore, merge, or replace the attribute at the given
    * path within the YAML structure.  This is the default set of cases for a standard resource; subclasses will
    * need to override this for customization (e.g. filters require special merging).
   */


  _yamlMergeStrategy(pathName) {
    switch (pathName) {
      /* The following subtrees are written and managed by Kubernetes, so they will be ignored */
      case "status":
      case "metadata.uid":
      case "metadata.selfLink":
      case "metadata.generation":
      case "metadata.resourceVersion":
      case "metadata.creationTimestamp":
      case "metadata.annotations.kubectl.kubernetes.io/last-applied-configuration":
        return "ignore";

      /* The empty path, or any other path that we don't recognize, merge the attributes. */
      case "":
      default:
        return "merge";
    }
  }

  /* _recursiveMerge(original, updated, diffs, path)
   *
   * Merge the original and updated values based on the result of this._yamlMergeStrategy(pathName).  Subclasses
   * may want to define their own mergeStrategies depending on the attributes in the YAML that they want to merge,
   * replace, or ignore.
   *
   * Returns an object that is the merge of the original aud updated, recursively merging subtrees by traversing
   * each object in parallel.
   *
   * We also track the changes that we make to the original object. This was originally for debugging, but it's also
   * useful feedback for users.
   */

  _recursiveMerge(original, updated, diffs = new Map(), path=[]) {
    let pathName = path.join('.');
    let strategy = this._yamlMergeStrategy(pathName);

    /* If ignore or replace: */
    switch (strategy) {
      case "ignore":  diffs.set(pathName, "ignored");  return undefined;
      case "replace": diffs.set(pathName, "replaced"); return updated;
    }

    /* If merge:
     * We check the type of the original and the type of the updated objects to determine how to merge.
     * Handle null as a special case because typeof null returns "object", and we want to simply update.
     */

    if (original === null) {
      diffs.set(pathName, "updated");
      return updated;
    }

    /* The type of the original object at this point in the tree: undefined, object, string, number, bigint, boolean? */
    switch (typeof original) {
      case "undefined":
        /* original undefined - check type of updated */
        if (typeof updated === "object") {
          /* Special case of Array object */
          if (Array.isArray(updated)) {
            diffs.set(pathName, "updated");
            return updated;
          } else {
            return this._mergeSubtree(original, updated, diffs, path);
          }
        }
        /* Normal case: original undefined, and updated is an object.
         * Return the updated object.
         */
        else {
          diffs.set(pathName, "updated");
          return updated;
        }
      case "object":
        /* Special case of Array object. */
        if (Array.isArray(original)) {
          /* just return the new updated value */
          return updated;
        }
        else {
          /*  otherwise perform the object merge at the current path. */
          return this._mergeSubtree(original, updated, diffs, path);
        }
      /* The normal (leaf) case: values */
      case "string":
      case "number":
      case "bigint":
      case "boolean":
        /* Return the original if same as updated, or updated isn't defined. */
        if (original === updated || updated === undefined) {
          /* no diff */
          return original;
        }
        else {
          /* Save diff */
          diffs.set(pathName, "updated");
          return updated;
        }
      default:
        throw new Error(`don't know how to merge ${typeof original}`);
    }
  }

  /* _mergeSubtree(original, updated, diffs, path)
   * Merge the subtrees of two objects at a given point in the path.
   */

  _mergeSubtree(original, updated, diffs, path) {
    /* Initialize original, updated to empty objects if undefined. */
    original = (original === undefined ? {} : original);
    updated  = (updated  === undefined ? {} : updated);

    /* Get the set of all keys in the original and updated objects. */
    let allKeys = setUnion(new Set(Object.keys(original)), new Set(Object.keys(updated)));
    let result  = {};

    for (let key of allKeys) {
      let keyInOriginal = original.hasOwnProperty(key);
      let keyInUpdated  = updated.hasOwnProperty(key);
      let merged = undefined;

      /* Do both objects have this key?
       * If so, recursively merge at that attribute.
       */
      if (keyInOriginal && keyInUpdated) {
        merged = this._recursiveMerge(original[key], updated[key], diffs, path.concat([key]));
      }

      /* Does the original have the key, and updated not? If so, continue merging the original's object's tree, and
       * pass undefined down the updated path.
       */
      else if (keyInOriginal && !keyInUpdated) {
        merged = this._recursiveMerge(original[key], undefined, diffs, path.concat([key]));
      }
      /* Does the updated object have a key that the original doesn't? If so, continue merging down the updated
       * object's tree, and pass undefined down the original path.
       */
      else if (!keyInOriginal && keyInUpdated) {
        merged = this._recursiveMerge(undefined, updated[key], diffs, path.concat([key]));
      }
      /* If we get this far then apparently Object.hasOwnProperty is broken! */
      else {
        throw new Error("this should be impossible");
      }

      /* If we have a result from the recursive merges above, set the
       * result key:value to the merged
       */
      if (merged !== undefined) {
        result[key] = merged;
      }
    }

    return result;
  }

}

