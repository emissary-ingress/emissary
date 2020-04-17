//SECTION:Yak
import { IResourceCollection } from '../interfaces/iresource_collection.js'
import { IResource } from '../interfaces/iresource.js'
import { MemoryStore } from '../tests/store_mocks.js'
import { Model } from '../framework/model2.js'

class Yak extends IResource {

  static get defaultYaml() {
    let yaml = IResource.defaultYaml
    yaml.kind = "Yak"
    yaml.spec = {
      odor: "<enter odor intensity>"
    }
    return yaml
  }

  // optional convenience getter and setter
  get odor() {
    return this.yaml.spec.odor
  }

  // we don't need to bother with calling this.notify() because any
  // changes to the yaml will automatically invoke notify
  set odor(value) {
    this.yaml.spec.odor = value
  }

  validateSelf() {
    let messages = new Map()
    let allowed = ["very smelly", "super smelly", "extremely smelly"]
    if (!allowed.includes(this.odor)) {
      messages.set("odor", "the only allowed value are " + JSON.stringify(allowed))
    }
    return messages
  }

}
//SECTION:YakHerd
import { View, html, repeat, css } from '../framework/view2.js'

class YakHerd extends View {

  static get styles() {
    return yakStyles()
  }

  static get properties() {
    return {
      yaks: { type: Model },
      renderYak: { type: Function }
    }
  }

  constructor() {
    super()
    // MemoryStore is an in-memory store implementation used for
    // testing. It is also useful for prototyping.
    this.yaks = new IResourceCollection(new MemoryStore(Yak))
    this.renderYak = this.renderSimpleYak
  }

  connectedCallback() {
    super.connectedCallback()
    // make some yaks up front, so we have something interesting to display by default
    this.makeSomeYaks()
  }

  makeSomeYaks() {
    let y1 = this.yaks.new()
    y1.name = "hairy"
    let y2 = this.yaks.new()
    y2.name = "fuzzy"
    let y3 = this.yaks.new()
    y3.name = "smelly"
  }

  // switch between our two views
  toggleRender() {
    if (this.renderYak === this.renderSimpleYak) {
      this.renderYak = this.renderStandardYak
    } else {
      this.renderYak = this.renderSimpleYak
    }
  }

  // Use repeat to render each yak view.
  render() {
    return html`
      <div>
        <button @click=${()=>this.toggleRender()}>Toggle Yak Views!</button>
        <button @click=${()=>this.makeSomeYaks()}>Make More Yaks!</button>
      </div>
      ${Array.from(this.yaks).length === 0 ? html`<div class="cell">no yaks, try making more!</div>` : ""}
      <div class="grid">${repeat(this.yaks, (y)=>y.key(), (y)=>this.renderYak(y))}</div>
    `
  }

  // We provide two ways to render our yaks, a simple view, and the standard one:
  renderSimpleYak(yak) {
    return html`<div class="cell"><simple-yak-view .model=${yak}></simple-yak-view></div>`
  }

  renderStandardYak(yak) {
    return html`<div class="cell"><yak-view .model=${yak}></yak-view></div>`
  }

}

customElements.define('yak-herd', YakHerd)
//SECTION:SimpleYakView
class SimpleYakView extends View {

  static get properties() {
    return {
      model: {type: Model}
    }
  }

  render() {
    return html`<div class="cell"><pre>${JSON.stringify(this.model.yaml, undefined, 2)}</pre></div>`
  }

}

customElements.define('simple-yak-view', SimpleYakView)
//SECTION:YakView
import { IResourceView } from '../interfaces/iresource_view.js'

class YakView extends IResourceView {

  static get styles() {
    return css`
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
      margin: 30px 0 0;
      font-size: .9rem;
    }
    .card, .card .col .con {
      display: -webkit-flex;
      display: -ms-flexbox;
      display: flex
    }
    .card .col {
      -webkit-flex: 1 0 50%;
      -ms-flex: 1 0 50%;
      flex: 1 0 50%;
      padding: 0 30px 0 0
    }
    .card .col .con {
      margin: 10px 0;
      -webkit-flex: 1;
      -ms-flex: 1;
      flex: 1;
      -webkit-justify-content: flex-end;
      -ms-flex-pack: end;
      justify-content: flex-end;
      height: 30px
    }
    .card .col, .card .col .con label, .card .col2, .col2 a.cta .label {
      -webkit-align-self: center;
      -ms-flex-item-align: center;
      -ms-grid-row-align: center;
      align-self: center
    }
    .col2 a.cta  {
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
    .col2 a.cta .label {
      text-transform: uppercase;
      font-size: .8rem;
      font-weight: 600;
      line-height: 1rem;
      padding: 0 0 0 10px;
      -webkit-flex: 1 0 auto;
      -ms-flex: 1 0 auto;
      flex: 1 0 auto
    }
    .col2 a.cta svg {
      width: 15px;
      height: auto
    }
    .col2 a.cta svg path, .col2 a.cta svg polygon {
      transition: fill .7s ease;
      fill: #000
    }
    .col2 a.cta:hover {
      color: #5f3eff;
      transition: all .2s ease;
      border: 2px #5f3eff solid
    }
    .col2 a.cta:hover svg path, .col2 a.cta:hover svg polygon {
      transition: fill .2s ease;
      fill: #5f3eff
    }
    .col2 a.cta {
      display: -webkit-flex;
      display: -ms-flexbox;
      display: flex;
      -webkit-flex-direction: row;
      -ms-flex-direction: row;
      flex-direction: row
    }
    .col2 a.off {
      display: none;
    }
    div.off, span.off, input.off, b.off {
      display: none;
    }
    .row {
      display: -webkit-flex;
      display: -ms-flexbox;
      display: flex;
      -webkit-flex-direction: row;
      -ms-flex-direction: row;
      flex-direction: row
    }
    .row .row-col {
      flex: 1;
      padding: 10px 5px;
    }
    .row .row-col:nth-child(1) {
      font-weight: 600;
    }
    .row .row-col:nth-child(2) {
      flex: 2;
    }
    .card .row {
      border-bottom: 1px solid rgba(0, 0, 0, .1);
    }
    .card div.line:nth-last-child(2) {
      border-bottom: none;
    }
    .errors ul {
      color: red;
    }
    .margin-right {
      margin-right: 20px;
    }
    .justify-right {
      text-align: right;
    }
    button, input, select, textarea {
      font-family: inherit;
      font-size: 100%;
      margin: 0
    }
    button, input {
      line-height: normal
    }
    button, html input[type=button], input[type=reset], input[type=submit] {
      -webkit-appearance: button;
      cursor: pointer
    }
    input[type=search] {
      -webkit-appearance: textfield;
      box-sizing: content-box
    }
    input {
      background: #efefef;
      padding: 5px;
      margin: -5px 0;
      width: 100%;
    }
    input[type=checkbox], input[type=radio] {
      box-sizing: border-box;
      padding: 0;
      width: inherit;
      margin: 0.2em 0;
    }
    textarea {
      background-color: #efefef;
      padding: 5px;
      width: 100%;
    }
    div.yaml-wrapper {
      overflow-x: scroll;
    }
    div.yaml-wrapper pre {
      width: 500px;
      font-size: 90%;
    }
    div.yaml ul {
      color: var(--dw-purple);
      margin-left: 2em;
      margin-bottom: 0.5em;
    }
    div.yaml li {
    }
    div.yaml li .yaml-path {
    }
    div.yaml li .yaml-change {
    }
    div.namespace {
      display: inline;
    }
    div.namespaceoff {
      display: none;
    }
    div.namespace-input {
      margin-top: 0.1em;
    }
    .pararen {
      -webkit-justify-content: center;
      -ms-flex-pack: center;
      justify-content: center;
      -webkit-align-content: center;
      -ms-flex-line-pack: center;
      align-content: center;
      -webkit-align-items: center;
      -ms-flex-align: center;
      align-items: center;
    }
    div.namespace-input .pararen {
      font-size: 1.2rem;
      padding: 0 5px;
      display: inline;
      top: -0.2em;
    }
    div.namespace-input input {
      width: 90%;
    }
  `
  }

  renderSelf() {
    return html`
      <div class="row line">
        <label class="row-col margin-right justify-right">odor:</label>
        <div class="row-col">
          <input type=text
                 .value=${this.model.odor}
                 @input=${(e)=>{this.model.odor = e.target.value}}
                 ?disabled=${this.model.isReadOnly()}>
        </div>
      </div>`
  }

  validateSelf() {
    return new Map()
  }

}

customElements.define('yak-view', YakView)
//SECTION:ignored
function yakStyles() {
  return css`
      div {
          margin: 0.5em;
          padding: 0.5em;
          width: 80%;
      }

      .grid {
          display: inline-grid;
      }

      .cell {
          border-style: solid;
          border-radius: 0.3em;
      }`
}
