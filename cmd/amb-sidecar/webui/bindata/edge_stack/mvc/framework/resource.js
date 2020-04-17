import {Model} from "./model.js"
import {CoW} from "./cow.js"
import {mapMerge} from "./utilities.js"

const aes_res_source = "getambassador.io/resource-source";

// global counter for new resource keys
var newCounter = 0

export class Resource extends Model {

  static yamlKey(yaml) {
    let metadata = yaml.metadata || {}
    return (yaml.kind || "") + "::" + (metadata.name || "") + "::" + (metadata.namespace || "")
  }

  static get defaultYaml() {
    return {
      apiVersion: "getambassador.io/v2",
      metadata: {
        name: "<none>",
        namespace: "default"
      }
    }
  }

  constructor(collection) {
    super()
    // The collection that contains this resource.
    this.collection = collection

    // True iff the user has requested that this resource be deleted.
    this.deleted = false

    // The stored yaml. This is null for new resources or resources
    // that have been deleted out from under the UI.
    this.storedYaml = null
    // The stored version of the resource. This is tracked by kubernetes. Kubernetes will change this
    // version on every successful write operation. We use this to detect conflicting writes.
    this.storedVersion = null

    // A snapshot of the storedYaml and storedVersion at the moment that the resource is edited.
    this.editedYaml = null
    this.editedVersion = null

    // The yaml field for public access.
    this.yaml = {}

    // If non-null then the resource is in the process of being
    // saved/created/deleted. This field holds the promise that will
    // resolve to the eventual outcome.
    this.pending = null
    this.pendingResolver = null
    this.pendingRejector = null
  }

  get yaml() {
    return this._yaml
  }

  set yaml(value) {
    this._yaml = new CoW(value, ()=>{
      if (this.isReadOnly()) {
        throw new Error("cannot change read-only resource")
      } else {
        this.notify()
      }
    })
  }

  get store() {
    return this.collection.store
  }

  get state() {
    if (this.isNew()) {
      return this.isPending() ? "pending-create" : "new"
    } else if (this.isModified()) {
      if (this.storedYaml === null) {
        return "zombie"
      }

      if (this.storedVersion !== this.editedVersion) {
        return "conflicted"
      }

      return this.isPending() ? "pending-save" : "modified"
    } else if (this.isDeleted()) {
      return this.isPending() ? "pending-delete" : "deleted"
    }

    return "stored"
  }

  isNew() {
    return this.storedYaml === null
  }

  isModified() {
    return this.editedYaml !== null
  }

  isDeleted() {
    return this.deleted
  }

  isPending() {
    return this.pending !== null
  }

  isReadOnly() {
    if (this.isPending()) {
      return true
    }

    if (this.isNew() || this.isModified()) {
      return false
    }

    return true
  }

  resolvePending() {
    this.pendingResolver()
    this.pendingRejector = null
    this.pendingResolver = null
    this.pending = null
    this.notify()
  }

  rejectPending(e) {
    this.pendingRejector(e)
    this.pendingRejector = null
    this.pendingResolver = null
    this.pending = null
    this.notify()
  }

  inspect() {
    let parts = []
    if (this.storedVersion) {
      parts.push("V:" + this.storedVersion)
    }
    if (this.isNew()) {
      parts.push("N")
    }
    if (this.isModified()) {
      parts.push("M")
    }
    if (this.isDeleted()) {
      parts.push("D")
    }
    if (this.isPending()) {
      parts.push("P")
    }
    if (this.isReadOnly()) {
      parts.push("R")
    } else {
      parts.push("W")
    }

    parts.push(" " + this.state)
    
    return parts.join("") + " " + JSON.stringify(this.yaml)
  }

  key() {
    // We return a guaranteed unique key if we are new. This will
    // change when new resources are saved.
    if (this.collection.new_resources.has(this)) {
      if (!this._key) {
        this._key = `NEW::${newCounter++}`
      }
      return this._key
    } else {
      return Resource.yamlKey(this.yaml)
    }
  }

  load(yaml) {
    // store this here because isNew() returns this.storedYaml === null
    let wasNew = this.isNew()

    let next = JSON.stringify(yaml)
    let prev = JSON.stringify(this.storedYaml)
    this.storedYaml = JSON.parse(next)
    let metadata = this.storedYaml.metadata || {}
    let rv = metadata.resourceVersion
    if (typeof rv === "undefined") {
      this.storedVersion = prev == next ? this.storedVersion : this.storedVersion + 1
    } else {
      this.storedVersion = rv
    }

    if (wasNew) {
      if (this.isPending()) {
        this.resolvePending()
      }
    } else if (this.isModified()) {
      if (this.isPending()) {
        // When the stored version is updated, we take this to mean
        // that our write has succeeded. Right now that is not
        // technically true, someone else could have updated the
        // stored version just after we hit save. In the future we
        // might be able to come up with a scheme that is robust to
        // this if we properly use kubernetes conflict detection APIs,
        // but for right now this is good enough.
        if (this.storedVersion !== this.editedVersion) {
          this.editedYaml = null
          this.editedVersion = null
          this.resolvePending()
        }
      }
    }

    if (!this.isModified()) {
      this.yaml = this.storedYaml
    }

    if (prev != next) {
      this.notify()
    }
  }

  edit() {
    if (this.isNew()) {
      throw new Error("cannot edit a new resource")
    }

    if (this.isDeleted()) {
      throw new Error("cannot edit a deleted resource")
    }

    if (this.editedYaml === null) {
      this.editedYaml = JSON.parse(JSON.stringify(this.storedYaml))
      this.editedVersion = this.storedVersion
      this.yaml = this.editedYaml
    }

    this.notify()
  }

  delete() {
    this.deleted = true
    this.notify()
  }

  cancel() {
    if (this.isNew()) {
      this.collection.new_resources.delete(this)
      this.collection.notify()
    } else if (this.isModified()) {
      this.editedYaml = null
      this.editedVersion = null
      this.yaml = this.storedYaml
      this.notify()
    } else if (this.isDeleted()) {
      this.deleted = false
      this.notify()
    } else {
      throw new Error("cannot cancel a non-new, non-edited, non-deleted resource")
    }
  }

  save() {
    if (!(this.isNew() || this.isModified() || this.isDeleted())) {
      throw new Error("can only save new or modified or deleted resources")
    }

    let result = new Promise((good, bad)=>{
      this.pendingResolver = good
      this.pendingRejector = bad

      if (this.isDeleted()) {
        if (this.storedYaml !== null) {
          this.store.delete(this.collection, this.yaml)
            .catch((e)=>{
              this.rejectPending(e)
            })
        } else {
          this.resolvePending()
        }
      } else {
        if (this.isModified() && !CoW.changed(this.yaml)) {
          // applying unmodifed yaml is a noop which means we will
          // never get a confirmation that our application had any
          // effect, so we treat this as a cancel to avoid getting
          // stuck in the pending state forever
          this.cancel()
          this.resolvePending()
        } else {
          this.store.apply(this.collection, this.yaml)
            .then(()=>{
              // move it from new to resources, but we don't count success
              // until we hear back from the store
              if (this.isNew()) {
                this.collection.new_resources.delete(this)
                this.collection.resources.set(this.key(), this)
              }
            })
            .catch((e)=>{
              if (this.isNew()) {
                // move us back to the new state, but record the error
                this.collection.resources.delete(this.key())
                this.collection.new_resources.add(this)
              }
              this.rejectPending(e)
            })
        }
      }
    })
    this.pending = result
    this.notify()
    return result
  }

  validate() {
    let errors  = new Map();
    let message = "";

    /* Perform basic validation.  This can be extended by subclasses that implement validateSelf() */
    message = this.validateName(this.name);
    if (message) errors.set("name", message);

    message = this.validateName(this.namespace);
    if (message) errors.set("namespace", message);

    /* If this resource is being added, check the kind, name, and namespace against existing resources
     * to be sure that the resource's values are unique in the system.
     * TODO: have collection determine if this resource complies with the uniqueness criterion.
     */

    /* Any errors from self validation? Merge the results of validateSelf with the existing results from above.
     * validateSelf() overrides any errors returned above with the same name (i.e. name or namespace)
     */

    return mapMerge(errors, this.validateSelf());
  }

  // A bunch of convenience getters/setters. These are all just
  // aliases for directly accessing the yaml field.

  get kind() {
    return this.yaml.kind
  }

  get metadata() {
    return this.yaml.metadata
  }

  set metadata(value) {
    this.yaml.metadata = value
  }

  get name() {
    return this.metadata.name
  }

  set name(value) {
    if (!this.metadata) {
      this.metadata = {}
    }
    this.metadata.name = value
  }

  get namespace() {
    return this.metadata.namespace
  }

  set namespace(value ) {
    if (!this.metadata) {
      this.metadata = {}
    }
    this.metadata.namespace = value
  }

  get annotations() {
    return this.metadata.annotations
  }

  set annotations(value) {
    if (!this.metadata) {
      this.metadata = {}
    }
    this.metadata.annotations = value
  }

  get spec() {
    return this.yaml.spec
  }

  set spec(value) {
    this.yaml.spec = {}
  }

  get status() {
    return this.yaml.status
  }

  sourceURI() {
    let annotations = this.annotations;
    if (annotations && aes_res_source in annotations) {
      return annotations[aes_res_source];
    } else {
      /* Return undefined (same as nonexistent property, vs. null) */
      return undefined;
    }
  }

  /* ============================================================
   * Utility methods -- Validation
   * ============================================================
   */

  /**
   * validateName(name)
   *
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


  /**
   * validateEmail(email)
   *
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
    let format = /^(([^<>()[\]\\.,;:\s@\"]+(\.[^<>()[\]\\.,;:\s@\"]+)*)|(\".+\"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$/;
    if (email.match(format) && email.length <= 253) {
      return null;
    } else {
      return "Email address does not match required format.";
    }
  }

  /**
   * validateURL(url)
   *
   * returns null if url is valid, error string if not.
   *
   * TODO: Implement real URL validation.
   */
  validateURL(url) {
    return null;
  }

}
