import {html} from '../vendor/lit-element.min.js'
import {SingleResource, SortableResourceSet} from './resources.js';
import {getCookie} from './cookies.js';
import {ApiFetch} from "./api-fetch.js";
import {HASH} from "./hash.js";
import "./terminal.js";

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

class Project extends SingleResource {

  static get properties() {
    let props = super.properties;
    props["source"] = {type: String};
    props["spec"] = {type: Object};
    return props;
  }

  constructor() {
    super()
    this.source = ""
    this._spec = {}
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

        let commitBelongsToThisProject = this.resource.children.commits.some((commit) => {
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

  /**
   * Implement.
   */
  init() {
    this._spec = this.resource.spec
  }

  /**
   * Implement.
   */
  kind() {
    return "Project"
  }

  /**
   * Implement.
   */
  spec() {
    return this._spec
  }

  // override to ignore pods, since that's an artifical resource we stuff into things
  mergeStrategy(pathName) {
    switch (pathName) {
    case "commits":
    case "pods":
      return "ignore";
    default:
      return undefined;
    }
  }

  max(dates) {
    let result = null
    for (let d of dates) {
      if (result === null || d > result) {
        result = d
      }
    }
    return result
  }

  renderDeployedCommits(prefix, commits) {
    commits = commits || [];

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
    ${commit ? html`<a href="https://github.com/${this.resource.spec.githubRepo}/tree/${commit.spec.ref}">${shortenRefName(commit.spec.ref)}</a>` : ""}
  </div>
  <div>
    <a href="https://github.com/${this.resource.spec.githubRepo}/commit/${commit.spec.rev}">${commit.spec.rev.slice(0, 7)}...</a>
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
    return html`<a href="https://github.com/${this.resource.spec.githubRepo}/pull/${prNumber}/">PR#${prNumber}</a>`;
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
<a style="text-decoration:none;${styles}" href="/.previews/${this.resource.spec.prefix}/${commit.spec.rev}/">url</a>
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
                       .value="${this._spec[name]}"
                       @change=${(e)=>{this._spec[name]=e.target.value; this.requestUpdate("spec")}}/>`
  }

  reset() {
    super.reset()
    this._spec = this.resource.spec
  }

  /**
   * Implement.
   */
  renderResource() {
    return html`
<visible-modes list>
${this.renderDeployedCommits(this._spec.prefix, this.resource.children.commits)}

<dw-terminal source=${this.source} @close=${(e)=>this.closeTerminal()}></dw-terminal>
</visible-modes>
<visible-modes add edit>
  <div class="row line">
    <label class="row-col margin-right justify-right">host:</label>
    <div class="row-col">
      ${this.input("text", "host")}
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
      ${this.input("text", "githubRepo")}
    </div>
  </div>
  <div class="row line">
    <label class="row-col margin-right justify-right">github token:</label>
    <div class="row-col">
      ${this.input("password", "githubToken")}
    </div>
  </div>
<visible-modes>
`
  }

}

customElements.define('dw-project', Project);

export class Projects extends SortableResourceSet {

  constructor() {
    super([
      {value: "name", label: "Name"},
      {value: "namespace", label: "Namespace"},
      {value: "host", label: "Host"},
      {value: "prefix", label: "Prefix"},
      {value: "githubRepo", label: "Repo"}
    ]);
  }

  subscribe() {
    let looper = ()=>{
      this.poll();
      setTimeout(looper, 1000);
    }
    looper();
  }

  poll() {
    ApiFetch(`/edge_stack/api/projects`, {
      headers: {
        'Authorization': 'Bearer ' + getCookie("edge_stack_auth")
      }
    }).then(res => res.json())
      .then(this.onSnapshotChange.bind(this))
      .catch(err => console.log('Error fetching projects: ', err));
  }

  getResources(snapshot) {
    return snapshot;
  }

  sortFn(sortByAttribute) {
    return function(r1, r2) {
      if (sortByAttribute === "name" || sortByAttribute === "namespace") {
        return r1.metadata[sortByAttribute].localeCompare(r2.metadata[sortByAttribute]);
      } else {
        return r1.spec[sortByAttribute].localeCompare(r2.spec[sortByAttribute]);
      }
    }
  }

  renderInner() {
    return this.renderBoilerplate("Projects", "Published Github projects.", "dw-project", {
      metadata: {
        namespace: "default",
        name: ""
      },
      spec: {
        host: "",
        prefix: "",
        githubRepo: "",
        githubToken: ""
      },
      children: {
        commits: [],
      }
    });
  }

  // todo: everything below here is generic and should be able to be
  //       hoisted into SortableResourceSet and friends instead of
  //       being copied everywhere

  renderSet() {
    return html`
<div>
  ${this.resources.map(r => {return this.renderSingle("dw-project", r, this.state(r))})}
</div>`
  }

  renderSingle(component, resource, state, id) {
    if (id === undefined) {
      return html([`<${component} .resource=`, ` .state=`, `></${component}>`], resource, state)
    } else {
      return html([`<${component} id=`, ` .resource=`, ` .state=`, `></${component}>`], id, resource, state)
    }
  }

  renderBoilerplate(title, subtitle, component, defaultYaml) {
    let shtml = super.renderInner();
    return html`
<div class="header_con">
  <div class="col">
    <img alt="projects logo" class="logo" src="../images/svgs/projects.svg">
      <defs><style>.cls-1{fill:#fff;}</style></defs>
        <g id="Layer_2" data-name="Layer 2">
          <g id="Layer_1-2" data-name="Layer 1"></g>
        </g>
    </img>
  </div>
  <div class="col">
    <h1>${title}</h1>
    <p>${subtitle}</p>
  </div>
  <div class="col2">
    <a class="cta add ${this.readOnly() ? "off" : ""}" @click=${()=>this.shadowRoot.getElementById("add-resource").onAdd()}>
      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 30 30"><defs><style>.cls-a{fill:none;stroke:#000;stroke-linecap:square;stroke-miterlimit:10;stroke-width:2px;}</style></defs><title>add_1</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><line class="cls-a" x1="15" y1="9" x2="15" y2="21"/><line class="cls-a" x1="9" y1="15" x2="21" y2="15"/><circle class="cls-a" cx="15" cy="15" r="14"/></g></g></svg>
      <div class="label">add</div>
    </a>
    <div class="sortby">
      <select id="sortByAttribute" @change=${this.onChangeSortByAttribute.bind(this)}>
    ${this.sortFields.map(f => {
      return html`<option value="${f.value}">${f.label}</option>`
    })}
      </select>
    </div>
  </div>
</div>
${this.renderSingle(component, defaultYaml, this.addState, "add-resource")}
${shtml}
`;
  }

}

customElements.define('dw-projects', Projects);
