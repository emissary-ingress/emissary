import {html} from '../vendor/lit-element.min.js'
import {SingleResource, SortableResourceSet} from './resources.js';
import {getCookie} from './cookies.js';
import {ApiFetch} from "./api-fetch.js";
import {HASH} from "./hash.js";
import "./terminal.js";

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
      if (parts.length === 4) {
        let type = parts[0]
        let ns = parts[1]
        let name = parts[2]
        let sha = parts[3]

        if (ns == this.resource.metadata.namespace && name == this.resource.metadata.name) {
          this.source = `../api/${type === "build" ? "logs" : "slogs"}/${ns}/${name}/${sha}`
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

  renderPods(prefix, pods) {
    let commits = new Map()
    for (let name in pods) {
      let pod = pods[name]
      let id = pod.metadata.labels.commit
      let commit = null
      if (commits.has(id)) {
        commit = commits.get(id)
      } else {
        commit = {id: id, prefix: prefix, builds: [], previews: []}
        commits.set(id, commit)
      }

      if (pod.metadata.labels.hasOwnProperty("build")) {
        commit.builds.push(pod)
      } else {
        commit.previews.push(pod)
      }
    }
    let values = Array.from(commits.values())
    values.sort((a,b) => {
      let amax = this.max(a.builds.map(p=>Date.parse(p.metadata.creationTimestamp)))
      let bmax = this.max(b.builds.map(p=>Date.parse(p.metadata.creationTimestamp)))
      if (amax === null && bmax === null) {
        return 0
      } else if (amax > bmax) {
        return -1
      } else {
        return 1
      }
    })

    return html`
<div class="row line">
  <label class="row-col margin-right justify-right">Previews:</label>
  <div class="row-col">
    <div style="display:grid; grid-template-columns: 1fr 1fr;">
      ${values.map((c)=>this.renderCommit(c))}
    </div>
  </div>
</div>
`
  }

  renderCommit(commit) {
    return html`
  <div>
    ${commit.id.slice(0, 7)}...
  </div>
  <div class="justify-right">
    ${commit.builds.length > 0 ? commit.builds.map(p=>this.renderBuild(commit, p)) : html`<span style="opacity:0.5">build</span>`} |
    ${commit.previews.length > 0 ? commit.previews.map(p=>this.renderPreview(commit, p)) : html`<span style="opacity:0.5">log</span> | <span style="opacity:0.5">url</span>`}
  </div>
`
  }

  renderBuild(commit, pod, idx) {
    var styles = "color:blue"
    switch (pod.status.phase) {
    case "Succeeded":
      styles = "color:green"
      break
    case "Failed":
      styles = "color:red"
      break
    }
    return html`<a style="cursor:pointer;${styles}" @click=${()=>this.toggleTerminal(commit, pod)}>build</a>`
  }

  renderPreview(commit, pod) {
    let cstats = pod.status.containerStatuses
    var styles = "color:blue"
    if (cstats && cstats.length > 0 && cstats[0].ready) {
      styles = "color:green"
    } else {
      styles = "color:red"
    }
    return html`
<a style="cursor:pointer;${styles}" @click=${()=>this.toggleTerminal(commit, pod)}>log</a> |
<a style="text-decoration:none;${styles}" href="/.previews/${commit.prefix}/${commit.id}/">url</a>
`
  }

  toggleTerminal(commit, pod) {
    let name = this.resource.metadata.name
    var log
    if (pod.metadata.labels.hasOwnProperty("build")) {
      log = `build/${pod.metadata.namespace}/${name}/${commit.id}`
    } else {
      log = `deploy/${pod.metadata.namespace}/${name}/${commit.id}`
    }

    let old = HASH.get("log")

    if (old == log) {
      HASH.delete("log")
    } else {
      HASH.set("log", log)
    }
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
${this.renderPods(this._spec.prefix, this.resource.pods)}
<dw-terminal source=${this.source}></dw-terminal>
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
    let projects = [];
    snapshot.forEach((obj)=>{
      obj.project.pods = obj.pods
      projects.push(obj.project)
    });
    return projects;
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
