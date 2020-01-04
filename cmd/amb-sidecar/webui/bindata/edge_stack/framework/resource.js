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

/* Interface class for Model */
import { Model } from "./model.js"

/* Annotation key for sourceURI. */
const aes_res_source = "aes_res_source";

export class Resource extends Model {

  /* constructor()
   * Here the model initializes any internal state that is common to all Resources.
   * Typically a concrete Resource class would initialize the Resource kind, name, namespace,
   * and other useful state to be maintained in the Resource instance.
   */

  constructor(data) {
    /* Define the instance variables that are part of the model. Views and other Resource users will access
     * these for rendering and modification.  All resource objects have a kind, a name, and a namespace, which
     * together are a unique identifier throughout the Kubernetes system.  They may also have annotations,
     * labels, and a status, which are also saved as object state.
    */

    /* calling Model.constructor() */
    super();

    /* Initialize instance variables */
    this.kind        = data.kind;
    this.name        = data.metadata.name;
    this.namespace   = data.metadata.namespace;
    this.labels      = data.metadata.labels || {};
    this.annotations = data.metadata.annotations || {};
    this.status      = data.status || this.getEmptyStatus();

    /* Save the initialization data */
    this._data = data;
  }

  /* static uniqueKeyFor(data)
   * Return a computed modelKey given some structured data (a hierarchical key/value
   * structure).  This is a static function that is given the data block from a snapshot and returns
   * the model key for that data.  Each Resource subclass will know the structure and extract
   * the appropriate information to create the Resource's key.  This is needed for identity in a
   * collection of Resources.  It is a static function because a given Resource may not yet exist in
   * the collection and its key must be created from the raw data.
   *
   * A basic Resource uses the kind, name, and namespace to build the resource key.
   */

  static uniqueKeyFor(data) {
    return data.kind + "::" + data.metadata.name + "::" + data.metadata.namespace;
  }

   /* updateFrom(data)
   * Update the Resource object state from the snapshot data block for this Resource.  Compare the values in the
   * data block with the stored state in the Resource.  If the data block has different data than is currently
   * stored, update that instance variable with the new data and set a flag to indicate an update has been made.
   * If any of the state has changed, notify listeners.
   */

  updateFrom(data) {
    let updated = false;

    /* get the new labels value from the data, or an empty object if undefined. */
    let new_labels = data.metadata.labels || {};

    if (this.labels !== new_labels) {
      this.labels = new_labels;
      updated = true;
    }

    /* get the new annotations value from the data, or an empty object if undefined. */
    let new_annotations = data.metadata.annotations || {};

    if (this.annotations !== new_annotations) {
      this.annotations = new_annotations;
      updated = true;
    }

    /* get the new status value from the data, or the emptyStatus object if undefined. */
    let new_status = data.status || this.getEmptyStatus();

    if ((this.status.state  !== new_status.state) ||
        (this.status.reason !== new_status.reason)) {
      this.status = new_status;
      updated = true;
    }

    /* Give subclasses a chance to update themselves. */
    updated = updated || this.updateSelfFrom(data);

    /* Notify listeners if any updates occurred. */
    if (updated) {
      this.notifyListenersUpdated();
    }
  }

  /* getEmptyStatus()
   * Utility method for initializing the status of the resource.  Returns a dictionary that has the basic
   * structure of the status attribute in the Kubernetes resource structure.  This is simply a dictionary
   * with state = "none" and an empty reason string.
   */

  getEmptyStatus() {
    return {
      "state": "none",
      "reason": ""
    };
  }

  /* getSpec()
   * Return the spec attribute of the Resource.  This method is needed for the implementation of the Save
   * function which uses kubectl apply.  This method must return an object that will be serialized with JSON.stringify
   * and supplied as the "spec:" portion of the Kubernetes YAML that is passed to kubectl.  See the Host class for
   * an example implementation.
   */

  getSpec() {
    throw new Error("Please implement Resource:getSpec()");
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
        name:        this.name,
        namespace:   this.namespace,
        labels:      this.labels,
        annotations: this.annotations
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
     * validateSelf() overrides.  The spread operator (...) converts the Map into an Array which the Map
     * constructor then uses for the new key/value entries.
     */
    errors = new Map(...errors, ...this.validateSelf());

    return errors;
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


  /* validateEmail(name)
   * We validate that the user has provided a plausible looking email address. In the future, we should actually
   * validate that it's a real email address using something like
   * https://www.textmagic.com/free-tools/email-validation-tool
   * with an appropriate fallback if we are unable to reach outside the firewall (if we can't reach the outside
   * system, then use simple pattern matching).
   *
   * returns null if email is valid, error string if not.
   */

  validateEmail(email) {
    return null;
  }

  /* _validateURL(url)
  * returns null if url is valid, error string if not.
  */

  validateURL(email) {
      return null;
    }
}

