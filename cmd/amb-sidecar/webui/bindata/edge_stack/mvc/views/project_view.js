import { Model } from '../framework/model.js'
import { html } from '../framework/view.js'
import { IResourceView } from '../interfaces/iresource_view.js'
import "./terminal.js";

import {getCookie} from '../../components/cookies.js';
import {ApiFetch} from "../../components/api-fetch.js";
import {HASH} from "../../components/hash.js";

import {AllHosts} from '../models/host_collection.js'

export class ProjectView extends IResourceView {

  static get properties() {
    let props = super.properties
    props.source = {type: String}
    props.hosts = {type: Model}
    return props
  }

  /* extend */
  constructor() {
    super();
    this.source = ""
    this.hosts = AllHosts
  }

  // alias for readability, our model is a project
  get project() {
    return this.model
  }

  connectedCallback() {
    super.connectedCallback();
    window.addEventListener("hashchange", this.handleHashChange.bind(this), false);
    // make sure we look at the hash on first load
    this.handleHashChange()
  }

  disconnectedCallback() {
    super.disconnectedCallback();
    window.removeEventListener("hashchange", this.handleHashChange.bind(this), false);
  }

  handleHashChange() {
    let log = HASH.get("log")
    if (log) {
      let parts = log.split("/")
      if (parts.length === 2) {
        let logType = parts[0];
        let commitQName = parts[1];

        let commitBelongsToThisProject = this.project.commits.some((commit) => {
          return `${commit.metadata.name}.${commit.metadata.namespace}` == commitQName;
        });

        if (commitBelongsToThisProject) {
          this.source = `../api/${logType === "build" ? "logs" : "slogs"}/${commitQName}`
          return
        }
      }
    }

    this.source = ""
  }

  /* override */
  validateSelf() {
    let errors = new Map();

    return errors;
  }

  /* override */
  renderSelf() {
    let hostnames = Array.from(this.hosts).map((h)=>h.hostname)
    return html`
<div class="${this.visibleWhen("list")}">
  ${this.renderDeployedCommits(this.project.prefix, this.project.commits)}

  <dw-terminal source=${this.source} @close=${(e)=>this.closeTerminal()}></dw-terminal>
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
    <label class="row-col margin-right justify-right">github repo:</label>
    <div class="row-col">
      ${this.input("text", "repo")}
    </div>
  </div>
  <div class="row line">
    <label class="row-col margin-right justify-right">github token:</label>
    <div class="row-col">
      ${this.input("password", "token")}
    </div>
  </div>
<div>
`
  }

  renderDeployedCommits(prefix, commits) {
    commits = Array.from(commits);

    commits.sort((a,b) => {
      return Date.parse(a.metadata.creationTimestamp) - Date.parse(b.metadata.creationTimestamp);
    })

    return html`
<div class="row line">
  <label class="row-col margin-right justify-right">Deployed Commits:</label>
  <div class="row-col">
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
    let matches = commit.spec.ref.match(/^refs\/pull\/([0-9])+\/(head|merge)$/);
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
    return html`
<a style="cursor:pointer;${styles};${selected}" @click=${()=>this.openTerminal("deploy", commit)}>log</a> |
<a style="text-decoration:none;${styles}" href="/.previews/${this.project.prefix}/${commit.spec.rev}/">url</a>
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

  closeTerminal() {
    HASH.delete("log")
  }

  input(type, name) {
    return html`<input type="${type}"
                       name="${name}"
                       .value="${this.project[name]}"
                       @change=${(e)=>{this.project[name]=e.target.value}}/>`
  }

  select(name, options) {
    let sorted = Array.from(options)
    sorted.sort()
    if (this.project.isNew() && !this.project.isReadOnly()) {
      this.project[name] = sorted[0]
    }
    return html`
<select ?disabled=${this.project.isReadOnly()}
        @change=${(e)=>{
this.project[name]=e.target.value
}}>
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
