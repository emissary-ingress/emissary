/*
 * ProjectCollectionView
 */

import { AllProjects } from "../models/project_collection.js"
import { IResourceCollectionView } from '../interfaces/iresourcecollection_view.js'
import { Model } from '../framework/model.js'
import { View, html, css, repeat } from '../framework/view.js'
import {HASH} from "../../components/hash.js";
import './project_view.js'

export class ProjectCollectionView extends View {

  static get properties() {
    return {
      projects: {type: Model},
      sortFields: {type: Array},
      sortBy: {type: String},
      log: {type: String},
    }
  }

  static get styles() {
    return css`
      div.sortby {
          text-align: right;
      }
      div.sortby select {
        font-size: 0.85rem;
        border: 2px #c8c8c8 solid;
        text-transform: none; 
      }
      div.sortby select:hover {
        color: #5f3eff;
        transition: all .2s ease;
        border: 2px #5f3eff solid;
      }
      .card {
	background: #fff;
	border-radius: 10px;
	padding: 10px 30px 10px 30px;
	box-shadow: 0 10px 5px -11px rgba(0, 0, 0, .6);
	width: 100%;
	-webkit-flex-direction: row;
	-ms-flex-direction: row;
	flex-direction: row;
	-webkit-flex: 1 1 1;
	-ms-flex: 1 1 1;
	flex: 1 1 1;
	margin: 5px 0 0;
	font-size: .9rem;
      }
      .global {
        margin-top: 10px;
      }
      .global label {
	font-weight: 600;
      }
      .off {
        display: none;
      }
    `
   }

  constructor() {
    super()
    this.projects = AllProjects
    this.sortFields = [
      {value: "name", label: "Name"},
      {value: "namespace", label: "Namespace"}
    ]
    this.sortBy = "name"
    // holds the log selector string from the hash
    this.log = HASH.get("log")
  }

  connectedCallback() {
    super.connectedCallback()
    this.hashchange = ()=>{this.log = HASH.get("log")}
    window.addEventListener("hashchange", this.hashchange, false)
  }

  disconnectedCallback() {
    super.disconnectedCallback()
    window.removeEventListener("hashchange", this.hashchange, false)
  }

  get sorted() {
    if (this.selected) {
      return [this.selected]
    }

    let result = Array.from(this.projects)
    result.sort((a, b)=>{
      if (a.isNew()) {
        return -1
      }
      if (b.isNew()) {
        return 1
      }
      return a[this.sortBy].localeCompare(b[this.sortBy])
    })
    return result
  }

  closeTerminal() {
    HASH.delete("log")
  }

  render() {
    let parsed = this.parseLogSelector(this.log)
    let displayed = parsed.selected ? [parsed.selected] : this.sorted
    let global = this.projects.errors

    return html`
<div>
  <link rel="stylesheet" href="../styles/resources.css">
  <div class="header_con">
    <div class="col">
      <img alt="Projects Logo" class="logo" src="../images/svgs/projects.svg">
        <defs><style>.cls-1{fill:#fff;}</style></defs>
        <g id="Layer_2" data-name="Layer 2">
          <g id="Layer_1-2" data-name="Layer 1"></g>
        </g>
      </img>
    </div>

    <div class="col">
      <h1>Projects</h1>
      <p>Projects are custom HTTP services managed by Ambassador Edge Stack</p>
    </div>

    <div class="col2">
      <a class="cta add" @click=${()=>this.projects.new()}>
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 30 30"><defs><style>.cls-a{fill:none;stroke:#000;stroke-linecap:square;stroke-miterlimit:10;stroke-width:2px;}</style></defs><title>add_1</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><line class="cls-a" x1="15" y1="9" x2="15" y2="21"/><line class="cls-a" x1="9" y1="15" x2="21" y2="15"/><circle class="cls-a" cx="15" cy="15" r="14"/></g></g></svg>
        <div class="label">add</div>
      </a>

      <div class="sortby" >
        <select id="sortByAttribute" @change=${(e)=>{this.sortBy = e.target.value}}>
          ${this.sortFields.map(f => {return html`<option value="${f.value}">${f.label}</option>`})}
        </select>
      </div>
    </div>
  </div>

  <div class="${global.length > 0 ? "global card" : "off"}">
   <label>Global Errors:</label> <dw-errors .errors=${global}></dw-errors>
  </div>

  <div>
    ${repeat(displayed, (r)=>r.key(), (r)=>html`<dw-mvc-project .model=${r}></dw-mvc-project>`)}
  </div>

  <div class="${parsed.source ? "card" : "off"}">
    <dw-terminal source=${parsed.source} @close=${(e)=>this.closeTerminal()}></dw-terminal>
  </div>
</div>
`
  }

  parseLogSelector(log) {
    let result = {
      // the selected project
      selected: null,
      // the source url for logs
      source: ""
    }
    if (log) {
      let parts = log.split("/")
      if (parts.length === 2) {
        let logType = parts[0];
        let commitQName = parts[1];

        result.selected = this.projectForCommit(commitQName)
        if (result.selected) {
          result.source = `../api/${logType === "build" ? "logs" : "slogs"}/${commitQName}`
        }
      }
    }

    return result
  }

  projectForCommit(qname) {
    for (let p of this.projects) {
      if (p.commits.some((c) => `${c.metadata.name}.${c.metadata.namespace}` === qname)) {
        return p
      }
    }
    return null
  }

}

customElements.define('dw-mvc-projects', ProjectCollectionView);
