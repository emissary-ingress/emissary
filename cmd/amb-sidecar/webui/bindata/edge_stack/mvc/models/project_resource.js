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
      githubRepo: "<the github repo>",
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

  /* override */
  validateSelf() {
    let errors  = new Map();

    if (!/^\S+$/.test(this.prefix)) {
      errors.set("prefix", "must not contain whitespace")
    }

    if (!/^\S+\/\S+$/.test(this.repo)) {
      errors.set("github repo", "must be of the form: owner/repo")
    }

    if (!/^\S+$/.test(this.token)) {
      errors.set("github token", "please supply a github token")
    }

    return errors;
  }

}
