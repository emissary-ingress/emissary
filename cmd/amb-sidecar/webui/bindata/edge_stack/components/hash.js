// The helpers in this file are intended to centralize the AES UI's
// interaction with the hash fragment of the URL and allow it to be
// shared for both authentication and deep linking into different tabs
// within the UI.
//
// Rather than directly writing/reading to/from the hash fragment,
// components should import the HASH global and use it to write/read
// to/from the hash:
//
//   import {HASH} from './hash.js'
//
//   ...
//
//   onHashChange() {
//       let tab = HASH.tab
//       let value = HASH.get("key")
//       ...
//   }
//
// The HASH global also provides HASH.tab that can be used to get/set
// the value of the current tab, and HASH.authToken which can be used
// to get/set the auth token.
//
// The fragment format is modeled on a URL. Based on my gooling this
// is all legal (surprinsgly). This is deliberate so we can change
// this later to be a URL if we want to.
class Hash {

  constructor() {
    this._base = ""
    this.params = new Map()
  }

  // internal
  decode() {
    this._base = ""
    this.params.clear()

    let hash = window.location.hash.slice(1)

    let parts = hash.split("?", 2)
    this._base = parts[0]

    if (parts.length > 1) {
      for (let p of parts[1].split("&")) {
        let pair = p.split("=", 2)
        let name = pair[0]
        let value = pair.length > 1 ? pair[1] : ""
        this.params.set(name, value)
      }
    }

    // for backwards compatibility with older edgectls, we used to use
    // the length of the hash to figure out if it was a jwt or a tab
    if (this.params.size === 0 && this._base.length > 300) {
      this.params.set("auth", this._base)
      this._base = ""
      this.encode()
    }
  }

  // internal
  encode() {
    let entries = []
    this.params.forEach((v, k)=>{
      if (v) {
        entries.push(`${k}=${v}`)
      } else {
        entries.push(`${k}`)
      }
    })

    let hash = ""

    if (this._base.length > 0) {
      hash += this._base
    }

    if (entries.length > 0) {
      hash += "?" + entries.join("&")
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

  // convenience functions that canonicalize how the current tab and
  // auth are stored in the hash
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
