//SECTION:Yak
import { IResourceCollection } from '../interfaces/iresource_collection.js'
import { IResource } from '../interfaces/iresource.js'
import { MemoryStore } from '../tests/store_mocks.js'
import { Model } from '../framework/model.js'

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
import { View, html, repeat, css } from '../framework/view.js'

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

  renderSelf() {
    return html`
      <link rel="stylesheet" href="../../styles/oneresource.css">
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
