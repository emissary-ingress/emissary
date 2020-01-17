/*
 * HostResource
 * This is the HostResource class, an implementation of IResource.  It implements the Host-specific state and methods
 * that are needed to model a single Host CRD.
 *
 * See the comments in ./resource.js, ./iresource.js and ./imodel.js for more details.
 */

/* For merging our spec with the rest of the resource's YAML */
import { objectMerge } from "../framework/utilities.js"

/* Interface class for Resource */
import { IResource } from "../interfaces/iresource.js"

export class HostResource extends IResource {
  /* constructor()
   * Here the model initializes any internal state that is common to all Resources.
   * Typically a concrete Resource class would initialize the Resource kind, name, namespace,
   * and other useful state to be maintained in the Resource instance.
   */

  constructor(yaml = { kind: "Host"}) {
    /* Define the instance variables that are part of the model. Views and other Resource users will access
     * these for rendering and modification.  All resource objects have a kind, a name, and a namespace, which
     * together are a unique identifier throughout the Kubernetes system.  They may also have annotations,
     * labels, and a status, which are also saved as object state.  A HostResource adds a hostname,
     * an acmeProvider and email, and a flag specifying whether acme is being used.
    */

    /* calling Resource.constructor(data) */
    super(yaml);

  }

  /* copySelf()
   * Create a copy of the Resource, with all Resource state (but not Model's listener list}
   */

  copySelf() {
    return new HostResource(this._fullYAML);
  }


  /* getYAML()
   * Return YAML that has the Resource's values written back into the _fullYAML, and has been pruned so that only
   * the necessary attributes exist in the structure for use as the parameter to applyYAML().
   */
  getYAML() {
    let yaml = super.getYAML();

    /* Set hostname, acmeProvider in the existing spec. Leave any other spec info there as needed,
     * such as acmeProvider.privateKeySecret and acmeProvider.registration.  Note that these are
     * guaranteed to exist since they are set to defaults in updateSelfFrom below if needed.
     */
    yaml.spec.hostname               = this.hostname;
    yaml.spec.acmeProvider.authority = this.useAcme ? this.acmeProvider : "none";
    yaml.spec.acmeProvider.email     = this.useAcme ? this.acmeEmail : "";

    return yaml;

  }

  /* updateSelfFrom(data)
   * Update the HostResource object state from the snapshot data block for this HostResource.  Compare the values
   * in the data block with the stored state in the Host.  If the data block has different data than is currently
   * stored, update that instance variable with the new data and set a flag to return true if any changes have
   * occurred.  The Resource class's method, updateFrom, will call this method and then notify listeners as needed.
   */

  updateSelfFrom(yaml) {
    let changed = false;

    /* If yaml does not include a spec, set it, and its subfield acmeProvider, to a default object so that
     * the hostname, acmeProvider, and acmeEmail fields will be set to their default values during initialization.
     * Otherwise javascript would fail, trying to access a field of "null".  Set other fields to default values
     * if they do not exist.
     */
    yaml.spec                         = yaml.spec                         || { acmeProvider: {}};
    yaml.spec.hostname                = yaml.spec.hostname                || "<specify new hostname>";
    yaml.spec.acmeProvider.authority  = yaml.spec.acmeProvider.authority  || "https://acme-v02.api.letsencrypt.org/directory";
    yaml.spec.acmeProvider.email      = yaml.spec.acmeProvider.email      || "<specify your email address here>";

    /* Initialize host-specific instance variables from yaml. For those fields that are unknown, initialize
     * to default values.  This occurs when adding a new HostResource whose values will be specified by the user.
     */

    /* Update the hostname if it has changed since the last snapshot */
    if (this.hostname !== yaml.spec.hostname) {
      this.hostname = yaml.spec.hostname;
      changed = true;
    }

    /* Update the acmeProvider if it has changed */
    if (this.acmeProvider !== yaml.spec.acmeProvider.authority) {
      this.acmeProvider = yaml.spec.acmeProvider.authority;
      changed = true;
    }

    /* Update the acmeEmail if it has changed. */
    if (this.acmeEmail !== yaml.spec.acmeProvider.email) {
      this.acmeEmail = yaml.spec.acmeProvider.email;
      changed = true;
    }

    /* Are we using Acme or not? we just check to see if the authority is "none" or "" and assume if there is an
     * authority, the user intends to use Acme.
     */
    let useAcme = (this.acmeProvider !== "none" && this.acmeProvider !== "");

    /* Update the useAcme flag if it is different than before, e.g. there is a provider now and there wasn't before,
     * or there is no longer a provider when there once was one specified.
     */
    if (this.useAcme !== useAcme) {
      this.useAcme = useAcme;
      changed = true;
    }

    return changed;
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

    message = this.validateName(this.hostname);
    if (message) errors.set("hostname", message);

    if (this.useAcme) {
      message = this.validateURL(this.acmeProvider);
      if (message) errors.set("acmeProvider", message);

      message = this.validateEmail(this.acmeEmail);
      if (message) errors.set("acmeEmail", message);
    }

    return errors;
  }
}

