/**
 * Resource
 * a concrete implementation, the Resource class implements the IModel and IResource interfaces.
 *
 * Kubernetes Resource (also known as an Object) is a persistent
 * entity in the Kubernetes system that is used to represent the
 * desired state of your cluster.  Kubernetes uses Resources to
 * apply policies and procedures to ensure that the cluster
 * reaches that desired state, by allocating, deallocating,
 * configuring and running compute and networking entities.
 *
 * The Kubernetes documentation makes a distinction between
 * a Kubernetes Object (the persistent data entities stored
 * in etcd) and a Resource (an endpoint in the Kubernetes API
 * that stores a collection of Objects), however, this
 * distinction is not consistently made throughout the documentation.
 * Here we will use the term Resource to indicate a chunk
 * of state -- a kind, name, namespace, metadata, and a spec -- that
 * we display and modify in the Web user interface.  So:
 *
 * This class is the generic superclass for all Kubernetes Resources
 * that are created, viewed, modified, and deleted in the Web UI.
 *
 * There are three main variables: kind, name, and namespace. These
 * three variables define the id of the object, so are read-only
 * once set.
 */

/* Constants for annotating the YAML for a resource. */

const aes_res_editable   = "aes_res_editable";
const aes_res_changed    = "aes_res_changed";
const aes_res_source     = "aes_res_source";
const aes_res_downloaded = "aes_res_downloaded";

import { ApiFetch } from "../components/api-fetch.js";
import { union }    from "./sets.js"

/* The IResource interface class. */
import { IResource } from "./iresource.js";

export class Resource extends IResource {

  /* constructor()
    * Define our instance variables that we
    * will be modeling.  Views will access these
    * for rendering and modification.  All
    * resource objects have a kind, a name,
    * and a namespace, which together are
    * a uniqueID throughout the Kubernetes
    * system.  They may also have annotations
    * and labels, and a status.
    */

  constructor(kind, name, namespace) {
    super(kind, name, resource);

    this.kind = kind;
    this.name = name;
    this.namespace = namespace;
    this.labels = {};
    this.annotations = {};
    this.status = this.getEmptyStatus();

    /* Resources may be edited.  We keep original
     * snapshot data in case we need to revert.
    * We keep a count for _editing in case there
    * is more than one view potentially editing
    * the resource at the same time.
    */
    this._editing = 0;

    /* Initialize data in case the object has not
     * yet received snapshot data.
     */
    this._data = this.getYAML();

    /* Maintain a shadow instance while editing.
     * The shadow is the original version of the
     * resource, and the resource stops updating
     * from the snapshot during the editing process.
     * We use the shadow to compute differences
     * between the before and after-editing states.
     */
    this._shadow = null;
  }

  /* modelKeyFor(data)
    * Return a computed modelKey given resource data.  In the
    * case of a Resource, we know that the data from the snapshot
    * has a kind, and metadata giving the name and namespace.
    */

  static modelKeyFor(data) {
    return data.kind + "::" + data.metadata.name + "::" + data.metadata.namespace;
  }

  /* Return the modelKey for this model.
   * NOTE: this function must return the identical string value for a given
   * resource, instantiated from data, as the modelKeyFor static function.
   */
  modelKey() {
    return this.kind + "::" + this.name + "::" + this.namespace
  }

  /* modelExtractor(snapshot)
    * Return a list of resources from the snapshot,
    * given the modelClassname (e.g. Host, Mapping, Filter).
    * Subclasses override this.
    */

  static modelExtractor(snapshot) {
    throw new Error("please implement modelExtractor()")
  }


  /* initFromSnapshot(data)
  * Similar to updateFromSnapshot but for the original initialization of
  * the model object, thus it does not notify about updates. The notification
  * about the creation of the model object is handled by whoever creates
  * the model object.  Note that labels and annotations may be empty.
  */
  initFromSnapshot(data) {
    this.kind = data.kind;
    this.name = data.metadata.name;
    this.namespace = data.metadata.namespace;
    this.labels = data.metadata.labels || {};
    this.annotations = data.metadata.annotations || {};
    this.status = data.status || this.getEmptyStatus();

    /* Save the initialization data for future shadow objects. */
    this._data = data;
  }

  /* updateFromSnapshot(data)
   * Update the model object from the data in the snapshot as received
   * from the backend via snapshot.js. If any of the values have changed,
   * notify my listeners via an 'updated' notification. For example:
   *
   * let updated = super.updateFromSnapshot();
   * if( this.abc !== data.abc ) { this.abc = data.abc; updated = true; }
   * if( this.def !== data.def ) { this.def = data.def; updated = true; }
   *
   * if (updated) notifyListenersUpdated(this);
   */

  updateFromSnapshot(data) {
    /* kind, name, and namespace are read-only once set.
     * but we can check the status (subclasses will call super).
     * Subclasses must notify listeners; this method will not.
     */

    /* If the resource is currently being edited, ignore snapshots
    *  and return false (e.g. not updated).
    */
    if (this.editing())
      return false;

    let updated = false;
    let other_labels = data.metadata.labels || {};

    if (this.labels !== other_labels) {
      this.labels = other_labels;
      updated = true;
    }

    let other_annotations = data.metadata.annotations || {};

    if (this.annotations !== other_annotations) {
      this.annotations = other_annotations;
      updated = true;
    }

    let other_status = data.status || this.getEmptyStatus();

    if ((this.status.state !== other_status.state) ||
      (this.status.reason !== other_status.reason)) {
      this.status = data.status;
      updated = true;
    }

    return updated;
  }

  /* editable()
   * Return true if the resource may be edited,
   * false otherwise.  This is determined by
   * an annotation: aes_res_editable, which is
   * either true, false, or nonexistent.  If
   * nonexistent, we default to editable.
   */

  editable() {
    let annotations = this.annotations;
    if (aes_res_editable in annotations) {
      return annotations[aes_res_editable];
    } else {
      return true;
    }
  }

  /* readOnly()
   * Is this resource editable or read-only?  This is determined by
   * an annotation: aes_res_editable.
   */

  readOnly() {
    return !this.editable();
  }

  /* validate()
   * Validate this Resource's state.  Returns
   * a dictionary of property: errorString if
   * there are any errors.  If the dictionary
   * is empty, there were no errors.
   */

  validate() {
    let errors  = new Map();
    let message = "";

    message = this._validateName(this.name);
    if (message) errors.set("name", message);

    message = this._validateName(this.namespace);
    if (message) errors.set("namespace", message);

    return errors;
  }

  /* editing()
   * Is the resource being edited by one or more views?
   */

  editing() {
    return this._editing > 0;
  }

  /* beginEdit()
   * Start editing the resource, which stops
   * handling updateFromSnapshot.  Create a
   * shadow copy for determining what has changed
   * during editing.
   */

  beginEdit() {
    if (!this.editing()) {
      /* Note that we are changing the resource */
      this.annotations[aes_res_changed] = true;

      /* bump the editing count. */
      this._editing += 1;

      /* Make a shadow copy for revert and compare. */
      this._shadow = this.shadowCopy();
    } else {
      console.log("Can't beginEdit if already editing.")
    }
  }

  /* commitEdit()
 * Commit the edit--write the data back to the server
 * and allow snapshot updates to resume.
 */

  commitEdit() {
    if (this.editing()) {
      this._editing -= 1;
      this._shadow = null;
    } else {
      console.log("Can't commitEdit if not editing.")
    }
  }

  /* cancelEdit()
   * Cancel editing the resource.  Revert to the last
   * known snapshot data and allow snapshot
   * updates to resume.
   */

  cancelEdit() {
    if (this.editing()) {
      /* Restore the last known snapshot data for this object. */
      this.initFromSnapshot(this._data);

      /* No longer editing, delete the shadow object. */
      this._editing -= 1;
      this._shadow = null;
    } else {
      console.log("Can't cancelEdit if not editing.")
    }
  }

  /* getEmptyStatus()
   * Utility method for initializing the status of the resource.
   */

  getEmptyStatus() {
    return {
      "state": "none",
      "reason": ""
    };
  }

  /* getSpec()
   * Override this method to implement the save behavior of a
   * resource.  This method must return an object that will get
   * rendered with JSON.stringify and supplied as the 'spec:' portion
   * of the kubernetes yaml that is passed to 'kubectl apply'. For example:
   *
   *   class Host extends Resource {
   *     ...
   *   getSpec() {
   *    return {
   *      hostname: this.hostname,
   *     acmeProvider: {
   *       this.useAcme
   *          ? { authority: this.acmeProvider, email: this.acmeEmail }
   *          : { authority: "none"}
   *      }
   *    }
   *  }
   *
   * The above spec will result in the following yaml being applied:
   *
   *    ---
   *    apiVersion: getambassador.io/v2
   *    kind: Host
   *    metadata:
   *      name: rhs.bakerstreet.io
   *      namespace: default
   *    spec:
   *      hostname: rhs.bakerstreet.io
   *      acmeProvider:
   *        authority: https://acme-v02.api.letsencrypt.org/directory
   *        email: rhs@alum.mit.edu
   *
   */

  getSpec() {
    throw new Error("please implement getSpec()")
  }

  /* sourceURI()
   * Return the source URI for this resource, if one exists.
   * In the case we have a source URI, provide a button next to the
   * Edit button which, when clicked, opens a window on that source URI.
   * Basically this is useful for tracking resources as they are applied
   * using GitOps, though the annotation must be applied in the GitOps
   * pipeline for this to work.
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


  /* getYAML()
    * Return the YAML object to JSON.stringify for Kubernetes apply.
    * Note that this is only the partial YAML; the full YAML for the
    * resource is saved in this._data.
    */
  getYAML() {
    return {
      apiVersion: "getambassador.io/v2",
      kind: this.kind,
      metadata: {
        name: this.name,
        namespace: this.namespace,
        labels: this.labels,
        annotations: this.annotations
      }
    }
  }

  /* yamlMergeStrategy(pathName)
    * Given a pathName, this method returns whether to
    * ignore, merge, or replace the attribute at the given
    * path within the YAML structure.  This is the default
    * set of cases for a standard resource; subclasses
    * will need to override this for customization
    * (e.g. filters require special merging).
   */


  yamlMergeStrategy(pathName) {
    switch (pathName) {
      /* The following subtrees are written and managed by Kubernetes,
       * so they will be ignored
       */
      case "status":
      case "metadata.uid":
      case "metadata.selfLink":
      case "metadata.generation":
      case "metadata.resourceVersion":
      case "metadata.creationTimestamp":
      case "metadata.annotations.kubectl.kubernetes.io/last-applied-configuration":
        return "ignore";

      /* The empty path, or any other path that we don't recognize,
       * merge the attributes.
       */
      case "":
      default:
        return "merge";
    }
  }

  /* getMergedYAML()
    * Return a YAML structure and a edits mapping,
    * that shows what has changed between the shadow model (the original)
    * and the current, edited model.  Returns:
    * { yaml: { ... }, edits: { attribute: difference string, ... }}
   */

  getMergedYAML() {
    let yaml   = {};
    let diffs  = new Map();

    /* Do we have a shadow resource to merge? */
    if (this._shadow) {
      let original = this._shadow._data;
      let edited   = this.getYAML();
      yaml = this._mergeObject(original, edited, diffs);
    }
    /* No shadow resource, just return the original */
    else {
      yaml = this._data;
    }

    // Return
    return { yaml: yaml, diffs: diffs }
  }

  /* doApply(yaml, cookie)
   * call the edge_stack API to apply the object's current state
   * as YAML.  Returns null if success, an error string if not.
    */

  doApply(yaml, cookie) {
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

  /* doDelete(cookie)
  * call the edge_stack API to delete this object.
  * Returns null if success, an error string if not.
   */

  doDelete(cookie) {
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

  /* create a shadow copy of this Resource.
   * This could be the single shadowCopy
   * method for all subclasses if we could
   * get the object's true class directly
   * and instantiate it.  For now, each
   * subclass must implement shadowCopy.
  */

  shadowCopy() {
    let copy = new Resource();
    copy.initFromSnapshot(this._data);
    return copy;
  }


  /* ============================================================
   * Private methods -- Validation
   * ============================================================
   */

  /* validateName(name)
   * name and namespaces rules as defined by
   * https://kubernetes.io/docs/concepts/overview/working-with-objects/names/\
   * returns null if name is valid, error string if not.
   */

  _validateName(name) {
    // lower-case letters, numbers, dash, and dot allowed.
    let format = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$/;
    if (name.match(format) && name.length <= 253) {
      return null;
    } else {
      return "Name must be {a-z0-9-.}, length <= 253";
    }
  }


  /* _validateEmail(name)
   * We validate that the user has provided a plausible looking
   * email address. In the future, we should actually validate that
   * it's a real email address using something like
   * https://www.textmagic.com/free-tools/email-validation-tool
   * with an appropriate fallback if we are unable to reach
   * outside the firewall (if we can't reach the outside system,
   * then use simple pattern matching).
   *
   * returns null if email is valid, error string if not.
   */

  _validateEmail(email) {
    return null;
  }

  /* _validateURL(url)
  * returns null if url is valid, error string if not.
  */

  _validateURL(email) {
    return null;
  }


  /* ============================================================
   * Private methods -- Merging YAML
   * ============================================================
   */

  /* _mergeObject(original, updated, diffs, path)
   *
   * Merge the original and updated values based on the result of
   * this.yamlMergeStrategy(pathName).  Subclasses will want to define
   * their own mergeStrategies depending on the attributes in the
   * YAML that they want to merge, replace, or ignore.
   *
   * Returns an object that is the merge of the original aud updated,
   * recursively merging subtrees by traversing each object in parallel.
   *
   * We also track the changes that we make to the original object. This
   * was originally for debugging, but it's also useful feedback for
   * users.
   */

  _mergeObject(original, updated, diffs = new Map(), path=[]) {
    let pathName = path.join('.');
    let strategy = this.yamlMergeStrategy(pathName);

    /* If ignore or replace: */
    switch (strategy) {
      case "ignore":  diffs.set(pathName, "ignored");  return undefined;
      case "replace": diffs.set(pathName, "replaced"); return updated;
    }

    /* If merge:
     * We check the type of the original and the type of the
     * updated objects to determine how to merge.
     *
     * Handle null as a special case because typeof null
     * returns "object", and we want to simply update.
     */

    if (original === null) {
      diffs.set(pathName, "updated");
      return updated;
    }

    /* The type of the original object at this point in the
     * object tree: undefined, object, string, number, bigint,
     * boolean?
     */

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
   *
   * Merge the subtrees of two objects at a given point in the path.
   */

  _mergeSubtree(original, updated, diffs, path) {
    /* Initialize original, updated to empty objects if undefined. */
    original = (original === undefined ? {} : original);
    updated  = (updated  === undefined ? {} : updated);

    /* Get the set of all keys in the original and updated objects. */
    let allKeys = union(new Set(Object.keys(original)), new Set(Object.keys(updated)));
    let result  = {};

    for (let key of allKeys) {
      let keyInOriginal = original.hasOwnProperty(key);
      let keyInUpdated  = updated.hasOwnProperty(key);
      let merged = undefined;

      /* Do both objects have this key?
       * If so, recursively merge at that attribute.
       */
      if (keyInOriginal && keyInUpdated) {
        merged = this._mergeObject(original[key], updated[key], diffs, path.concat([key]));
      }

      /* Does the original have the key, and updated not?
       * If so, continue merging the original's object's tree, and
       * pass undefined down the updated path.
       */
      else if (keyInOriginal && !keyInUpdated) {
        merged = this._mergeObject(original[key], undefined, diffs, path.concat([key]));
      }
      /* Does the updated object have a key that the original doesn't?
       * If so, continue merging down the updated object's tree, and
       * pass undefined down the original path.
       */
      else if (!keyInOriginal && keyInUpdated) {
        merged = this._mergeObject(undefined, updated[key], diffs, path.concat([key]));
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
