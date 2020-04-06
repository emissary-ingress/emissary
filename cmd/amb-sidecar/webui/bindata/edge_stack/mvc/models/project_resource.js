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

  get commits() {
    return (this.yaml.children || {}).commits || []
  }

  get errors() {
    let result = []
    // add project errors
    result.push(...(this.yaml.children || {}).errors || [])
    // add errors for all the child commits
    for (let c of this.commits) {
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
      if (!/^\S+$/.test(this.prefix)) {
        errors.set("prefix", "must not contain whitespace")
      }

      if (this.prefix[0] === "/") {
        errors.set("prefix start", "cannot begin with /")
      }

      if (this.prefix.length > 1 && this.prefix[this.prefix.length-1] === "/") {
        errors.set("prefix end", "cannot end with /")
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
