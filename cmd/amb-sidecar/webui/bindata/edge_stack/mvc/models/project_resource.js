/*
 * HostResource
 * This class implements the Host-specific state and methods
 * that are needed to model a single Host CRD.
 */

import { IResource } from "../interfaces/iresource.js";

export class ProjectResource extends IResource {

  // override
  static get defaultYaml() {
    let yaml = IResource.defaultYaml
    yaml.kind = "Project"
    yaml.spec = {
      host: "<the published hostname>",
      prefix: "<the published prefix>",
      githubRepo: "",
      githubToken: ""
    }
    return yaml
  }

  get host() {
    return this.spec.host
  }

  set host(value) {
    this.spec.host = value
  }

  get prefix() {
    return this.spec.prefix
  }

  set prefix(value) {
    this.spec.prefix = value
  }

  get repo() {
    return this.spec.githubRepo
  }

  set repo(value) {
    this.spec.githubRepo = value
  }

  get token() {
    return this.spec.githubToken
  }

  set token(value) {
    this.spec.githubToken = value
  }

  get revisions() {
    return (this.yaml.children || {}).revisions || []
  }

  get errors() {
    let result = []
    // add project errors
    result.push(...(this.yaml.children || {}).errors || [])
    // add errors for all the child revisions
    for (let c of this.revisions) {
      result.push(...((c.children || {}).errors || []))
    }
    return result
  }

  /* override */
  validateSelf() {
    let errors  = new Map();

    if (this.name && this.name.length > 22) {
      errors.set("project name", "too long, please choose a name that is less than 22 characters")
    }

    if (!this.prefix) {
      errors.set("prefix", "please supply a prefix")
    } else {
      let messages = []
      if (!/^\/\S+\/$/.test(this.prefix)) {
        messages.push('must not contain whitespace and must begin and end with a "/"')
      }

      let prefixes = new Set()
      for (let p of this.collection) {
        if (p !== this) {
          prefixes.add(p.prefix)
        }
      }

      if (prefixes.has(this.prefix)) {
        messages.push("already in use")
      }

      if (messages.length > 0) {
        errors.set("prefix", messages.join(", "))
      }
    }

    if (!/^\S+$/.test(this.token)) {
      errors.set("github token", "please supply a github token")
    }

    if (!this.repo) {
      errors.set("github repo", "please choose a github repo")
    } else if (!/^[a-zA-Z0-9._-]+\/[a-zA-Z0-9._-]+$/.test(this.repo)) {
      errors.set("github repo", "must be of the form: owner/repo")
    }

    return errors;
  }

}
