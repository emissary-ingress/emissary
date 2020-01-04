/*
 * HostResource
 * This is the HostResource class, a subclass of Resource.  It implements the Host-specific state and methods
 * that are needed to model a single Host CRD.
 *
 * See the comments in ./resource.js, ./iresource.js and ./imodel.js for more details on the expected interfaces
 * for a HostResource.
 */

/* Interface class for Model */
import { Resource } from "./resource.js"

/* Annotation key for sourceURI. */
const aes_res_source = "aes_res_source";

export class HostResource extends IResource {

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

    /* calling Resource.constructor(data) */
    super(data);

    /* host-specific instance variables. */
    this.hostname     = data.spec.hostname;
    this.acmeProvider = data.spec.acmeProvider.authority || "";
    this.acmeEmail    = data.spec.acmeProvider.email || "";
    this.useAcme      = (this.acmeEmail !== "" && this.acmeProvider !== "");
  }

  /* static uniqueKeyFor(data)
   * Return a computed modelKey given some structured data (a hierarchical key/value
   * structure).  This is a static function that is given the data block from a snapshot and returns
   * the model key for that data.  Each Resource subclass will know the structure and extract
   * the appropriate information to create the Resource's key.  This is needed for identity in a
   * collection of Resources.  It is a static function because a given Resource may not yet exist in
   * the collection and its key must be created from the raw data.
   */

  static uniqueKeyFor(data) {
    return Resource.uniqueKeyFor(data);
  }
  /* updateFrom(data)
   * Update the HostResource object state from the snapshot data block for this HostResource.  Compare the values
   * in the data block with the stored state in the Host.  If the data block has different data than is currently
   * stored, update that instance variable with the new data and set a flag to notify listeners of the changed
   * state once the HostResource has been fully updated.
   */

  updateFrom(data) {
    /* Let Resources do its part on the update. notifyListenersUpdated will not be called by Resources.updateFrom
     * but will return whether it made a change or not.  Since the HostResource class (for now) is a final class,
     * it calls this.notifyListenersUpdated.
     */
    let changed = super.updateFrom(data);

    /* Check hostname */
    if (this.hostname !== data.spec.hostname) {
      this.hostname = data.spec.hostname;
      changed = true;
    }

    /* Check acmeProvider */
    if (this.acmeProvider !== data.spec.acmeProvider.authority) {
      this.acmeProvider = data.spec.acmeProvider.authority;
      changed = true;
    }

    /* Check acmeEmail */
    if (this.acmeEmail !== data.spec.acmeProvider.email) {
      this.acmeEmail = data.spec.acmeProvider.email;
      changed = true;
    }

    /* Are we using Acme or not? we just check to see
     * if the authority is "none".
     */
    let useAcme = (this.acmeProvider != "none");

    if (this.useAcme !== useAcme) {
      this.useAcme = useAcme;
      changed = true;
    }

    /* Notify listeners - HostResource is a final class. */
    if (changed) {
      this.notifyListenersUpdated();
    }
  }

  /* getSpec()
   * Return the spec attribute of the Resource.  This method is needed for the implementation of the Save
   * function which uses kubectl apply.  This method must return an object that will be serialized with JSON.stringify
   * and supplied as the "spec:" portion of the Kubernetes YAML that is passed to kubectl.
   */

  getSpec() {
    return {
      hostname:     this.hostname,
      acmeProvider: this.useAcme
        ? {authority: this.acmeProvider, email: this.acmeEmail}
        : {authority: "none"}
    }
  }

  /* validateSelf()
   * Validate this HostResource's state by checking each object instance variable for correctness (e.g. email address
   * format, URL format, date/time, name restrictions).  Returns a dictionary of property: errorString if there
   * are any errors. If the dictionary is empty, there are no errors.
   *
   * in a HostResource, we need to validate the hostname, the acmeProvider, and the acmeEmail.
   */

  validateSelf() {
    let errors  = new Map();
    let message = null;

    message = this._validateName(this.hostname);
    if (message) errors.set("hostname", message);

    if (this.useAcme) {
      message = this._validateURL(this.acmeProvider);
      if (message) errors.set("acmeProvider", message);

      message = this._validateEmail(this.acmeEmail);
      if (message) errors.set("acmeEmail", message);
    }

    return errors;
  }
}

