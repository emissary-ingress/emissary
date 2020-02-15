/*
 * HostResource
 * This class implements the Host-specific state and methods
 * that are needed to model a single Host CRD.
 */

/* These three imports are needed for the getTermsOfService() method which makes an API
 * call to the Ambassador back-end and then returns an html fragment. */
import {getCookie} from "../../components/cookies.js";
import { ApiFetch } from "../../components/api-fetch.js";
import { html } from '../../vendor/lit-element.min.js'

/* The interface class we extend. */
import { IResource } from "../interfaces/iresource.js";

export const _defaultAcmeProvider = "https://acme-v02.api.letsencrypt.org/directory";
export const _defaultAcmeEmail    = "<specify contact email here>";

export class HostResource extends IResource {
  /* override and extend method from interface */
  constructor(yaml = { kind: "Host"}) {
    /*
     * Define the instance variables that are part of the model. Views and other Resource users will access
     * these for rendering and modification.  All resource objects have a kind, a name, and a namespace, which
     * together are a unique identifier throughout the Kubernetes system.  They may also have annotations,
     * labels, and a status, which are also saved as instance variables (but handled by the internal
     * framework classes we inherit from, thus we don't have to deal with them).  The only thing we have
     * to deal with are the instance variables specific to a Host: hostname,
     * an acmeProvider and email, and a flag specifying whether acme is being used.
    */
    super(yaml);

    this.hostname     = this.hostname     || "";  /* we use this conditional initialization..         */
    this.acmeProvider = this.acmeProvider || "";  /* ..scheme because the super(yaml) will..          */
    this.acmeEmail    = this.acmeEmail    || "";  /* ..call updateSelfFrom() which will usually..     */
    this.useAcme      = this.useAcme      || false; /* ..initialize the variables and we don't want.. */
    this.cached_terms_of_service = null;          /* ..to override that initialization if it happens  */
  }

  /* override */
  copySelf() {
    return new HostResource(this._fullYAML);
  }

  /* override and extend */
  getYAML() {
    let yaml = super.getYAML();
    /*
     * Set hostname, acmeProvider in the existing spec. Leave any other spec info there as needed,
     * such as acmeProvider.privateKeySecret and acmeProvider.registration.  Note that these are
     * guaranteed to exist since they are set to defaults in updateSelfFrom() if needed.
     */
    yaml.spec.hostname               = this.hostname;
    yaml.spec.acmeProvider.authority = this.useAcme ? this.acmeProvider : "none";
    yaml.spec.acmeProvider.email     = this.useAcme ? this.acmeEmail : "";

    return yaml;
  }

  /* override */
  updateSelfFrom(yaml) {
    let changed = false;
    /*
     * If yaml does not include a spec, set it, and its subfield acmeProvider, to a default object so that
     * the hostname, acmeProvider, and acmeEmail fields will be set to their default values during initialization.
     * Otherwise javascript would fail, trying to access a field of "null".  Set other fields to default values
     * if they do not exist.
     */
    yaml.spec                         = yaml.spec                         || { acmeProvider: {}};
    yaml.spec.hostname                = yaml.spec.hostname                || "<specify new hostname>";
    let has_acmeProvider              = (yaml.spec.acmeProvider.authority || false) !== false;
    yaml.spec.acmeProvider.authority  = yaml.spec.acmeProvider.authority  || _defaultAcmeProvider;
    yaml.spec.acmeProvider.email      = yaml.spec.acmeProvider.email      || _defaultAcmeEmail;
    /*
     * Initialize host-specific instance variables from yaml. For those fields that are unknown, initialize
     * to default values (this occurs when adding a new HostResource whose values will be specified by the user).
     */
    if (this.hostname !== yaml.spec.hostname) {
      this.hostname = yaml.spec.hostname;
      changed = true;
    }

    if (this.acmeProvider !== yaml.spec.acmeProvider.authority) {
      this.acmeProvider = yaml.spec.acmeProvider.authority;
      this.cached_terms_of_service = null;
      this.agreed_terms_of_service = has_acmeProvider;
      changed = true;
    }

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

  /* override */
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

  /* ================================ Utility Functions ================================ */

  /* setAcmeProvider()
   * when changing the ACME provider, we have to clear the cached terms of service because the terms of service
   * url is linked to the specific ACME provider
   */
  setAcmeProvider(value) {
    if (this.acmeProvider !== value) {
      this.acmeProvider = value;
      this.cached_terms_of_service = null;
    }
  }

  /* getTermsOfService()
   * Here we get the Terms of Service url from the ACME provider so that we can show it to the user. We do this
   * by calling an API on AES that then turns around and calls an API on the ACME provider. We cannot call the API
   * on the ACME provider directly due to CORS restrictions.
   */
  getTermsOfService() {
    if( this.cached_terms_of_service ) {
      /* if we have a cached copy, return that */
      return this.cached_terms_of_service;
    } else {
      let value = this.acmeProvider;
      /* if there is no acmeProvider, then there are no terms of service */
      if(!(this.acmeProvider !== "none" && this.acmeProvider !== "")) {
        this.cached_terms_of_service = html`<em>none</em>`;
        return this.cached_terms_of_service;
      } else {
        /* otherwise, if there is an acmeProvider, then make the async API call
         * to get the terms of service */
        let url = new URL('/edge_stack/api/tos-url', window.location);
        url.searchParams.set('ca-url', value);
        ApiFetch(url, {
          headers: new Headers({
            'Authorization': 'Bearer ' + getCookie("edge_stack_auth")
          })
        })
          .then(r => {
            /* when the async call to the API returns, the original call
             * to getTermsOfServiceURL() will have already returned (see below),
             * so when we get a good result from the API, we store it
             * in the cache and then notify our listeners that we've
             * changed. The listeners will then re-call getTermsOfServiceURL()
             * and retrieve the cached value. */
            r.text().then(t => {
              if (r.ok) {
                let domain_matcher = /\/\/([^\/]*)\//;
                let d = t.match(domain_matcher);
                if (d) {
                  d = d[1];
                } else {
                  d = t;
                }
                this.cached_terms_of_service = html`<a href="${t}" target="_blank">${d}</a>`;
                this.notifyListenersUpdated();
              } else {
                console.error("not-understood tos-url result: " + t);
              }
            })
          });
        /* we didn't have a cached copy, so we made the async call to the API
         * to fetch the value. While that async call is happening, the rest of
         * the UI continues executing, so we have to return a filler value. Later,
         * we (the model) notify the view that we've been updated (because the
         * async API call has returned a value and we've stored that value in
         * the cache), the view can call this function again and get the cached
         * (real) value.
         */
        return html`<em>...loading...</em>`;
      }
    }
  }
}

