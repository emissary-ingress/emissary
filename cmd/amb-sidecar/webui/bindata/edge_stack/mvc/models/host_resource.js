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

export class HostResource extends IResource {

  // override
  static get defaultYaml() {
    let yaml = IResource.defaultYaml
    yaml.kind = "Host"
    yaml.spec = {
      hostname: "<please enter a hostname>",
      acmeProvider: {
        authority: "https://acme-v02.api.letsencrypt.org/directory",
        email: "<specify contact email here>"
      }
    }
    return yaml
  }

  get hostname() {
    return this.spec.hostname
  }

  set hostname(value) {
    this.spec.hostname = value
  }

  get acmeProvider() {
    return this.yaml.spec.acmeProvider || {}
  }

  set acmeProvider(value) {
    this.yaml.spec.acmeProvider = value
  }

  get acmeAuthority() {
    return this.acmeProvider.authority
  }

  set acmeAuthority(value) {
    if (this.acmeProvider === undefined) {
      this.acmeProvider = {}
    }
    this.acmeProvider.authority = value
    // when changing the ACME provider, we have to clear the cached
    // terms of service because the terms of service url is linked to
    // the specific ACME provider
    this.cached_terms_of_service = null;
    this._agreed = false;
  }

  get acmeEmail() {
    return this.acmeProvider.email
  }

  set acmeEmail(value) {
    if (this.acmeProvider === undefined) {
      this.acmeProvider = {}
    }
    this.acmeProvider.email = value
  }

  get useAcme() {
    return this.acmeProvider !== "none" && this.acmeProvider !== ""
  }

  set useAcme(value) {
    if (false) {
      this.acmeProvider = "none"
    }
  }

  get agreed_terms_of_service() {
    if (typeof this._agreed === "undefined") {
      this._agreed = !this.isNew()
    } else {
      return this._agreed
    }
  }

  set agreed_terms_of_service(value) {
    this._agreed = value
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
      let value = this.acmeAuthority;
      /* if there is no acmeAuthority, then there are no terms of service */
      if(!(this.acmeAuthority !== "none" && this.acmeAuthority !== "")) {
        this.cached_terms_of_service = html`<em>none</em>`;
        return this.cached_terms_of_service;
      } else {
        /* otherwise, if there is an acmeAuthority, then make the async API call
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
                this.notify();
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
