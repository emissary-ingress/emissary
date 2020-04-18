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
      hash: {type: Model},
      sortFields: {type: Array},
      sortBy: {type: String},
      log: {type: String}
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
      .projects-cta {
        display: flex;
      }
      .projects-cta > * {
        align-self: center;
      }
      .projects-cta a.cta {
        margin: 0 5px;
        flex-grow: 0;
      }

      * {
        margin: 0;
        padding: 0;
        border: 0;
        position: relative;
        box-sizing: border-box
      }

      *, textarea {
        vertical-align: top
      }


      .header_con, .header_con .col {
        display: -webkit-flex;
        display: -ms-flexbox;
        display: flex;
        -webkit-justify-content: center;
        -ms-flex-pack: center;
        justify-content: center
      }

      .header_con {
        margin: 30px 0 0;
        -webkit-flex-direction: row;
        -ms-flex-direction: row;
        flex-direction: row
      }

      .header_con .col {
        -webkit-flex: 0 0 80px;
        -ms-flex: 0 0 80px;
        flex: 0 0 80px;
        -webkit-align-content: center;
        -ms-flex-line-pack: center;
        align-content: center;
        -webkit-align-self: center;
        -ms-flex-item-align: center;
        align-self: center;
        -webkit-flex-direction: column;
        -ms-flex-direction: column;
        flex-direction: column
      }

      .header_con .col svg {
        width: 100%;
        height: 60px
      }

      .header_con .col img {
        width: 100%;
        height: 60px;
      }

      .header_con .col img path {
        fill: #5f3eff
      }

      .header_con .col svg path {
        fill: #5f3eff
      }

      .header_con .col:nth-child(2) {
        -webkit-flex: 2 0 auto;
        -ms-flex: 2 0 auto;
        flex: 2 0 auto;
        padding-left: 20px
      }

      .header_con .col h1 {
        padding: 0;
        margin: 0;
        font-weight: 400
      }

      .header_con .col p {
        margin: 0;
        padding: 0
      }

      .header_con .col2, a.cta .label {
        -webkit-align-self: center;
        -ms-flex-item-align: center;
        -ms-grid-row-align: center;
        align-self: center
      }

      .logo {
        filter: invert(19%) sepia(64%) saturate(4904%) hue-rotate(248deg) brightness(107%) contrast(101%);
      }

      a.cta {
        text-decoration: none;
        border: 2px #efefef solid;
        border-radius: 10px;
        width: 90px;
        padding: 6px 8px;
        max-height: 35px;
        -webkit-flex: auto;
        -ms-flex: auto;
        flex: auto;
        margin: 10px auto;
        color: #000;
        transition: all .2s ease;
        cursor: pointer;
      }

      .header_con a.cta  {
        border-color: #c8c8c8;
      }

      a.cta .label {
        text-transform: uppercase;
        font-size: .8rem;
        font-weight: 600;
        line-height: 1rem;
        padding: 0 0 0 10px;
        -webkit-flex: 1 0 auto;
        -ms-flex: 1 0 auto;
        flex: 1 0 auto
      }

      a.cta svg {
        width: 15px;
        height: auto
      }

      a.cta svg path, .col2 a.cta svg polygon {
        transition: fill .7s ease;
        fill: #000
      }

      a.cta:hover {
        color: #5f3eff;
        transition: all .2s ease;
        border: 2px #5f3eff solid
      }

      a.cta:hover svg path, .col2 a.cta:hover svg polygon {
        transition: fill .2s ease;
        fill: #5f3eff
      }

      a.cta {
        display: -webkit-flex;
        display: -ms-flexbox;
        display: inline-flex;
        -webkit-flex-direction: row;
        -ms-flex-direction: row;
        flex-direction: row
      }

      a.off {
        display: none;
      }
    `
   }

  constructor() {
    super()
    this.projects = AllProjects
    this.hash = HASH
    this.sortFields = [
      {value: "name", label: "Name"},
      {value: "namespace", label: "Namespace"}
    ]
    this.sortBy = "name"
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
    this.hash.delete("log")
  }

  renderEmptyDescription() {
      return html`<p>Projects are custom HTTP services managed by Ambassador Edge Stack</p>`
  }

  renderAdd() {
      return html`<a class="cta add" @click=${()=>this.projects.new()}>
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 30 30"><defs><style>.cls-a{fill:none;stroke:#000;stroke-linecap:square;stroke-miterlimit:10;stroke-width:2px;}</style></defs><title>add_1</title><g id="Layer_2" data-name="Layer 2"><g id="Layer_1-2" data-name="Layer 1"><line class="cls-a" x1="15" y1="9" x2="15" y2="21"/><line class="cls-a" x1="9" y1="15" x2="21" y2="15"/><circle class="cls-a" cx="15" cy="15" r="14"/></g></g></svg>
        <div class="label">add</div>
      </a>`
  }

  renderEmpty() {
    return html`
<div class="card">
  <div class="projects-cta">
    <div>There are no projects to display. You can click</div>
    ${this.renderAdd()}
    <div>to create one. You will need:</div>
  </div>

  <div style="margin: 1em; margin-left: 2em;">
    <ol>
      <li>A github repo with an HTTP service implementation.</li>
      <li>A Dockerfile in the root of your repo that builds and runs your service on port 8080.</li>
      <li>A github token with repo scope.</li>
    </ol>
  </div>

  <div>
    If you'd like an example github repo to get you started, please
    <a target="_blank" href="https://github.com/datawire/project-template/generate">
      click here to generate one from our template
    </a>.
  </div>
</div>
`
  }

  render() {
    let parsed = this.parseLogSelector(this.hash.get("log"))
    let displayed = parsed.selected ? [parsed.selected] : this.sorted
    let global = this.projects.errors

    let title = parsed.selected ? `Project ${parsed.selected.name}` : 'Projects'

    return html`
<div>
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
      <h1>${title}</h1>
      ${displayed.length == 0 ? this.renderEmptyDescription() : ""}
    </div>

    <div class="col2">
      ${this.renderAdd()}

      <div class="sortby" >
        <select id="sortByAttribute" @change=${(e)=>{this.sortBy = e.target.value}}>
          ${this.sortFields.map(f => {return html`<option value="${f.value}">${f.label}</option>`})}
        </select>
      </div>
    </div>
  </div>

  <div class="${global.length > 0 ? "global card" : "off"}">
   <label>Global Errors:</label> <dw-errors .errors=${global} .columns=${80}></dw-errors>
  </div>

  <div>
    ${repeat(displayed, (r)=>r.key(), (r)=>html`<dw-mvc-project .model=${r}></dw-mvc-project>`)}
    ${displayed.length == 0 ? this.renderEmpty() : ""}
  </div>

  <div class="${parsed.source ? "card" : "off"}">
    <dw-terminal
      source=${parsed.source}
      @close=${(e)=>this.closeTerminal()}></dw-terminal>
  </div>
</div>
`
  }

  parseLogSelector(log) {
    let result = {
      // the selected project
      selected: null,
      // the source url for logs
      source: "",
      // the type of logs (build vs server logs)
      type: ""
    }
    if (log) {
      let parts = log.split("/")
      if (parts.length === 2) {
        let logType = parts[0];
        let revisionQName = parts[1];

        result.selected = this.projectForRevision(revisionQName)
        if (result.selected) {
          result.source = `../api/projects/logs/${logType}/${revisionQName}`
          result.type = logType
        }
      }
    }

    return result
  }

  projectForRevision(qname) {
    for (let p of this.projects) {
      if (p.revisions.some((c) => `${c.metadata.name}.${c.metadata.namespace}` === qname)) {
        return p
      }
    }
    return null
  }

}

customElements.define('dw-mvc-projects', ProjectCollectionView);
