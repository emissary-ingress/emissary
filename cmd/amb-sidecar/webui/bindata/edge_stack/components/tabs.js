import { LitElement, html, css } from 'https://cdn.pika.dev/-/lit-element/2.2.1/dist-es2019/lit-element.min.js'

export class Tabs extends LitElement {

  static get styles() {
    return css`
.tabs {
  float: left;
  width: 190px;
  background-color: black;
  color: white;
  height: 90vh;
  font-size: 85%;
  font-weight: bold;
}

.main {
  margin-left: 190px;
}

.tab {
  display: block;
  padding: 1em;
  line-height: 1.3;
  cursor: pointer;
}
.tab img {
  vertical-align: middle;
  padding-right: 0.8em;
}
.tab span.icon-aligned {
  vertical-align: middle;
}
.tab.active {
  background-color: var(--dw-purple);
  color: white;
}
.tab:hover {
  background-color: #a9a9a9;
}
.tab.active:hover {
  background-color: var(--dw-purple);
  color: white;
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
      if( this.tabs[i].tabIconFilename()) {
        this.links.push(html`<span class="${classes}" @click=${e => this.handleClick(i, e)}><img src="${this.tabs[i].tabIconFilename()}"/><span class="icon-aligned">${this.tabs[i].tabName()}</span></span> `)
      } else {
        this.links.push(html`<span class="${classes}" @click=${e => this.handleClick(i, e)}>${this.tabs[i].tabName()}</span> `)
      }
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
      name: { type: String },
      icon: { type: String }
    }
  }

  constructor() {
    super();
    this.name = "";
    this.icon = "";
  }

  tabName() {
    return this.name;
  }

  tabIconFilename() {
    return this.icon;
  }

  render() {
    return html`<div><slot></slot></div>`
  }
}

customElements.define('dw-tab', Tab)
