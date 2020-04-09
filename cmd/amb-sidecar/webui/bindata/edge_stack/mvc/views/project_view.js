import { Model } from '../framework/model.js'
import { html, css } from '../framework/view.js'
import { IResourceView } from '../interfaces/iresource_view.js'
import './terminal.js'
import './errors.js'

import { getCookie } from '../../components/cookies.js';
import { ApiFetch } from '../../components/api-fetch.js';
import { HASH } from '../../components/hash.js';

import { AllHosts } from '../models/host_collection.js'
import { parseGithubPagination } from './helpers.js'

export class ProjectView extends IResourceView {

  static get properties() {
    let props = super.properties
    props.source = {type: String}
    props.hosts = {type: Model}
    props.repos = {type: Array}
    props.repo_error = {type: String}
    return props
  }

  static get styles() {
    return css`
      ${super.styles}
      label.errors {
        font-weight: 600;
        padding-left: 5px;
      }
`
  }

  /* extend */
  constructor() {
    super();
    this.source = ""
    this.hosts = AllHosts
    this.repos = []
    this.repo_error = ""
  }

  // alias for readability, our model is a project
  get project() {
    return this.model
  }

  /* override */
  validateSelf() {
    let errors = new Map();

    return errors;
  }

  onDeleteButton() {
    super.onDeleteButton(...arguments)
    // clear the log parameter in case we are narrowed when this
    // project is deleted
    HASH.delete("log")
  }

  /* override */
  renderSelf() {
    let hostnames = Array.from(this.hosts).map((h)=>h.hostname)
    return html`
<div class="${this.visibleWhen("list")}">
  ${this.renderDeployedCommits(this.project.prefix, this.project.commits)}
</div>
<div class="${this.visibleWhen("add", "edit")}">
  <div class="row line">
    <label class="row-col margin-right justify-right">host:</label>
    <div class="row-col">
      ${this.select("host", hostnames)}
    </div>
  </div>
  <div class="row line">
    <label class="row-col margin-right justify-right">prefix:</label>
    <div class="row-col">
      ${this.input("text", "prefix")}
    </div>
  </div>
  <div class="row line">
    <label class="row-col margin-right justify-right">github token:</label>
    <div class="row-col">
      ${this.input("password", "token", this.fetchRepos.bind(this))}
      ${this.tokenInstructions()}
    </div>
  </div>
  <div class="row line">
    <label class="row-col margin-right justify-right">github repo:</label>
    <div class="row-col">${this.repoPicker()}</div>
  </div>
</div>

<div class=${this.project.errors.length === 0 ? "off" : ""}>
  <label class="errors">Project Errors:</label>
  <dw-errors .errors=${this.project.errors}></dw-errors>
</div>
`
  }

  tokenInstructions() {
    if (!this.project.token || this.repo_error) {
      return html`
<div style="padding-top: 0.5em">
  <div style="color: red">${this.repo_error ? html`Error: ${this.repo_error}</span>` : ""}</div>
  <a target="_blank" href="https://github.com/settings/tokens/new">Click here</a> to obtain a token from github. Make sure you select the <b>repo</b> scope!
</div>
`
    } else {
      return ""
    }
  }

  repoPicker() {
    return html`
  ${this.repos.length > 0 ? this.select("repo", this.repos) : "..."}
`
  }

  onEditButton() {
    super.onEditButton()
    this.fetchRepos()
  }

  fetchRepos() {
    console.log("fetching repos")
    // We choose a 30 character threshold to consider a token valid
    // because github tokens appear to always be 40 characters, but we
    // don't know for sure and that could change. It would be unlikely
    // to go lower though due to randomness requirements. The reason
    // to have a threshold at all is to avoid spamming github with
    // repo requests if people type in a few characters by accident or
    // something. People will almost certainly need to use cut and
    // paste for a valid token.
    if (this.project.token.length < 30) {
      this.repo_error = "Please supply a valid github token."
      return
    } else {
      this.repo_error = ""
    }

    let startLink = `https://api.github.com/user/repos?per_page=100`
    let opts = {
      headers: {
        Authorization: `Bearer ${this.project.token}`
      }
    }

    let depaginate = (r) => {
      let hdr = r.headers.get("Link")
      if (hdr) {
        let links = parseGithubPagination(hdr)
        if (links.next) {
          fetch(links.next, opts).then(depaginate).then(addRepos)
        }
      }
      return r.json()
    }

    let repo_errors = new Set()
    let allRepos = new Set()
    let addRepos = (repos) => {
      if (!Array.isArray(repos)) {
        let message = repos.message
        if (typeof message === "string") {
          repo_errors.add(message)
        } else {
          repo_errors.add(JSON.stringify(repos, undefined, 2))
        }
        this.repo_error = Array.from(repo_errors).join(", ")
      } else {
        for (let r of repos) {
          allRepos.add(r.full_name)
        }
        this.repos = Array.from(allRepos)
        this.repos.sort()
      }
    }

    fetch(startLink, opts).then(depaginate).then(addRepos)
  }

  renderDeployedCommits(prefix, commits) {
    commits = Array.from(commits);

    commits.sort((a,b) => {
      let delta = Date.parse(a.metadata.creationTimestamp) - Date.parse(b.metadata.creationTimestamp)
      if (delta === 0) {
        delta = a.spec.rev.localeCompare(b.spec.rev)
      }
      if (a.spec.isPreview && b.spec.isPreview) {
        return delta
      } else if (a.spec.isPreview) {
        return 1
      } else if (b.spec.isPreview) {
        return -1
      } else {
        return 0
      }
    })

    return html`
<div class="row line">
  <label class="row-col margin-right justify-right">Deployed Commits:</label>
  <div class="row-col">
    ${commits.length > 0 ? "" : "..."}
    <div style="display:grid; grid-template-columns: 0.5fr 1fr 1fr 2fr;">
      ${commits.map((c)=>this.renderCommit(c))}
    </div>
  </div>
</div>
`
  }

  renderCommit(commit) {
    return html`
  <div>
    ${this.renderPull(commit)}
  </div>
  <div>
    ${commit ? html`<a target="_blank" href="https://github.com/${this.project.repo}/tree/${commit.spec.ref}">${shortenRefName(commit.spec.ref)}</a>` : ""}
  </div>
  <div>
    <a target="_blank" href="https://github.com/${this.project.repo}/commit/${commit.spec.rev}">${commit.spec.rev.slice(0, 7)}...</a>
  </div>
  <div class="justify-right">
    ${(commit.children.builders || []).length > 0 ? commit.children.builders.map(p=>this.renderBuild(commit, p)) : html`<span style="opacity:0.5">build</span>`} |
    ${(commit.children.runners || []).length > 0 ? commit.children.runners.map(p=>this.renderPreview(commit, p)) : html`<span style="opacity:0.5">log</span> | <span style="opacity:0.5">url</span>`}
  </div>
`
  }

  renderPull(commit) {
    let matches = commit.spec.ref.match(/^refs\/pull\/([0-9]+)\/(head|merge)$/);
    if (!matches)
      return "";
    let prNumber = matches[1];
    return html`<a target="_blank" href="https://github.com/${this.project.repo}/pull/${prNumber}/">PR#${prNumber}</a>`;
  }

  renderBuild(commit, job) {
    var styles = "color:blue"
    if ((job.status.conditions||[]).some((cond)=>{return cond.type==="Complete" && cond.status==="True"})) {
      styles = "color:green"
    } else if ((job.status.conditions||[]).some((cond)=>{return cond.type==="Failed" && cond.status==="True"})) {
      styles = "color:red"
    }
    let selected = this.logSelected("build", commit) ? "background-color:#dcdcdc" : ""
    return html`<a style="cursor:pointer;${styles};${selected}" @click=${()=>this.openTerminal("build", commit)}>build</a>`
  }

  renderPreview(commit, statefulset) {
    var styles = "color:blue"
    if ((statefulset.status.observedGeneration === statefulset.metadata.generation) &&
        (statefulset.status.currentRevision === statefulset.status.updateRevision) &&
        (statefulset.status.readyReplicas >= statefulset.spec.replicas)) {
      styles = "color:green"
    }
    // TODO: We'd have to inspect individual pods to detect a failure :(
    //styles = "color:red"
    let selected = this.logSelected("deploy", commit) ? "background-color:#dcdcdc" : ""
    let url = `https://${this.project.host}/` + (commit.spec.isPreview
                                                 ? `.previews/${this.project.prefix}/${commit.spec.rev}/`
                                                 : `${this.project.prefix}/`);
    return html`
<a style="cursor:pointer;${styles};${selected}" @click=${()=>this.openTerminal("deploy", commit)}>log</a> |
<a style="text-decoration:none;${styles}" target="_blank" href="${url}">url</a>
`
  }

  logSelected(logType, commit) {
    return HASH.get("log") === this.logParam(logType, commit)
  }

  logParam(logType, commit) {
    return `${logType}/${commit.metadata.name}.${commit.metadata.namespace}`;
  }

  openTerminal(logType, commit) {
    HASH.set("log", this.logParam(logType, commit))
  }

  input(type, name, onchange) {
    return html`<input type="${type}"
                       name="${name}"
                       .value="${this.project[name]}"
                       @input=${(e)=>{
  this.project[name]=e.target.value
  if (onchange) { onchange() }
}}/>`
  }

  select(name, options) {
    let sorted = Array.from(options)
    sorted.sort()
    if (this.project.isNew() && !this.project.isReadOnly() && !sorted.includes(this.project[name])) {
      this.project[name] = sorted[0]
    }
    return html`
<select ?disabled=${this.project.isReadOnly()}
        @change=${(e)=>{this.project[name]=e.target.value}}>
  ${sorted.map((opt)=>html`<option .selected=${this.project[name] === opt} value="${opt}">${opt}</option>`)}
</select>
`
  }

}

/* Bind our custom elements to the HostView. */
customElements.define('dw-mvc-project', ProjectView);

function shortenRefName(refname) {
  // These are the same rules as used by git in
  // shorten_unambiguous_ref().
  // See: https://github.com/git/git/blob/e0aaa1b6532cfce93d87af9bc813fb2e7a7ce9d7/refs.c#L417
  var rules = [
    /^(.*)$/,
    /^refs\/(.*)$/,
    /^refs\/tags\/(.*)$/,
    /^refs\/heads\/(.*)$/,
    /^refs\/remotes\/(.*)$/,
    /^refs\/remotes\/(.*)\/HEAD$/,
  ];

  // This is the same (ambiguous) algorithm as ReferenceName.Short().
  // See: https://github.com/src-d/go-git/blob/v4.13.1/plumbing/reference.go#L113
  // Matching shorten_unambigous_ref's behavior would require us to
  // have a full listing of refnames.
  var ret;
  for (let rule of rules) {
    let match = refname.match(rule);
    if (match) {
      ret = match[1];
    }
  }
  return ret;
};
