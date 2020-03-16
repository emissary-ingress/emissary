// The helpers in this file are intended to centralize the AES UI's
// interaction with the hash fragment of the URL and allow it to be
// shared for both authentication and deep linking into different tabs
// within the UI.
//
// Rather than directly reading and writing to/from the hash fragment,
// components should import the HASH global and use its API to
// interact with the hash.
//
// The API is intended to facilitate deep linking into the AES UI, so
// for example we can provide links not only to the projects tab, but
// into the projects tab with a specific log output being selected.
//
// How you would do this. In your tab implentation:
//
//   import {HASH} from './hash.js'
//
//   ...
//
//
//   // Wire up your tab to pay attention to hash changes.
//   connectedCallback() {
//     super.connectedCallback();
//     window.addEventListener("hashchange", this.onHashChange.bind(this), false);
//     // make sure we look at the hash on first load
//     this.onHashChange()
//   }
//
//   // Use the info from the hash to select a specific item on the tab.
//   onHashChange() {
//     let tab = HASH.tab
//     let itemId = HASH.get("selected")
//     ...
//   }
//
//   ...
//
//   // Wire up any clicks so that they change the hash.
//   onClick(e) {
//     let itemId = e.target.id
//     HASH.set("selected", itemId)
//   }
//
//   ...
//
// That's it! You can use this to make any state "linkable". Any
// clicks that change your linkable state just need to be made to
// update a parameter in the hash, and the component that contains
// that state should always look to that hash parameter to determine
// its value.
//
// The fragment format is modeled on a URL. Every hash is of the form:
//
//   #[<base>[?<param1>=<value1>&<param2>=<value2>...]]
//
// Based on Rafi's googling on March 3, 2020 this is all legal
// (surprisingly). This is deliberate so we can change this later to
// be a URL if we want to.
//
// Because the current tab and auth tokens are effectively global
// state within the AES UI. The HASH API provides a special
// abstraction for accessing these properties. The HASH.tab property
// is currently just an alias for the "<base>" portion of the
// hash. The HASH.authToken property is currently just an alias for
// the "auth" parameter. This could change in the future, for example
// we might want to have the base be <tabname>/<subtab>. For this
// reason it is important to use these abstractions when accessing the
// current tab and/or authToken.
class Hash {

  constructor() {
    // the base portion of the hash, prior to the question mark (i.e. the path)
    this._base = ""
    // the parameters that appear after the question mark in the hash
    this.params = new URLSearchParams("")
  }

  // internal
  decode() {
    let hash = window.location.hash.slice(1)

    let parts = hash.split("?", 2)
    this._base = parts[0]

    if (parts.length > 1) {
      this.params = new URLSearchParams(parts[1])
    } else {
      this.params = new URLSearchParams("")
    }

    // Edgectl sets the entire hash to *just* a jwt. We detect that by
    // seeing if there are no parameters and the base is super
    // long. We may change this at some future date when we are
    // confident that the vast majority of AES installations that are
    // out there understand parameters, but for now (March 12 2020) we
    // need to keep this code here and leave edgectl's behavior as-is
    // so that you don't need to keep around different versions of
    // edgectl for different AES installations.
    if (this.params.toString().length === 0 && this._base.length > 300) {
      this.params.set("auth", this._base)
      this._base = ""
      this.encode()
    }
  }

  // internal
  encode() {
    let hash = ""

    if (this._base.length > 0) {
      hash += this._base
    }

    let qs = this.params.toString()
    if (qs.length > 0) {
      hash += "?" + qs
    }

    if (hash.length > 0) {
      window.location.hash = "#" + hash
    } else {
      window.location.hash = ""
    }
  }

  // get the base portion of the fragment
  get base() {
    this.decode()
    return this._base
  }

  // set the base portion of the fragment
  set base(value) {
    this._base = value
    this.encode()
  }

  // get a parameter value
  get(name) {
    this.decode()
    return this.params.get(name)
  }

  // set a parameter value
  set(name, value) {
    this.params.set(name, value)
    this.encode()
  }

  // delete a parameter
  delete(name) {
    this.params.delete(name)
    this.encode()
  }

  // These functions define how we store the tab and auth tokens. It
  // is important to use these rather than e.g. directly accessing the
  // parameter values since we may change/expand how we store them
  // over time.
  get tab() {
    return this.base
  }

  set tab(name) {
    this.base = name
  }

  get authToken() {
    return this.get("auth")
  }

  set authToken(value) {
    if (value) {
      this.set("auth", value)
    } else {
      this.delete("auth")
    }
  }

}

export let HASH = new Hash();
