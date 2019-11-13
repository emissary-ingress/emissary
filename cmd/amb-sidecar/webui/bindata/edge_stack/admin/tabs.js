import { LitElement, html, css } from 'https://cdn.pika.dev/-/lit-element/2.2.1/dist-es2019/lit-element.min.js'

export class Tabs extends LitElement {

  static get styles() {
    return css`
.tabs {
  float: left;
  width: 15%;
}

.main {
  margin-left: 16%;
}

.tab {
  border: 1px solid #ede7f3;
  box-shadow: 0 2px 4px rgba(0,0,0,.1);
  border-radius: 0.2em;
  display: block;
  padding: 0.5em;
  line-height: 1.3;
  cursor: pointer;
  color: blue;
}

.tab:hover {
  background-color: #ede7f3;
}

.active {
  background-color: #ede7f3;
}
`
  }

  constructor() {
    super()
    this.tabs = []
    this.links = []
    this.current = 0
  }

  handleSlotChange({target}) {
    this.tabs = target.assignedNodes().filter(n => 'tabName' in n)
    this.showCurrent()
  }

  showCurrent() {
    this.links = []
    for (let i = 0; i < this.tabs.length; i++) {
      let classes = "tab"
      if (i == this.current || this.tabs[i].slot == "sticky") {
        this.tabs[i].style.display = "block"
        classes += " active"
      } else {
        this.tabs[i].style.display = "none"
      }
      this.links.push(html`<span class="${classes}" @click=${e=>this.handleClick(i, e)}>${this.tabs[i].tabName()}</span> `)
    }
    this.requestUpdate()
  }

  handleClick(i, e) {
    this.current = i
    this.showCurrent()
  }

  render() {
    return html`
<div class="tabs">
  ${this.links}
</div>
<div class="main">
  <slot name="sticky"></slot>
</div>
<div class="main">
  <slot @slotchange=${this.handleSlotChange}></slot>
</div>
`
  }

}

customElements.define('dw-tabs', Tabs)

export class Tab extends LitElement {
  static get properties() {
    return {
      name: { type: String }
    }
  }

  constructor() {
    super()
    this.name = ""
  }

  tabName() {
    return this.name
  }

  render() {
    return html`<div><slot></slot></div>`
  }
}

customElements.define('dw-tab', Tab)
