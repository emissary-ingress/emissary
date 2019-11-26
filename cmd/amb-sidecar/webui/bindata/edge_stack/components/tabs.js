import { LitElement, html, css } from '/edge_stack/vendor/lit-element.min.js'

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
.tab span.with-no-icon {
  vertical-align: middle;
  padding-left: 33px;
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

  displayHash() {
    console.log(window.location.hash);
    for (let i = 0; i < this.tabs.length; i++) {
      if(window.location.hash === ('#' + this.tabs[i].tabHashName())) {
        if( this.current !== i ) {
          this.handleClick(i, null);
        }
        break;
      }
    }
  }

  constructor() {
    super();
    this.tabs = [];
    this.links = [];
    this.current = 0
  }

  connectedCallback() {
    super.connectedCallback();
    window.addEventListener("hashchange", this.displayHash.bind(this), false);
  }

  disconnectedCallback() {
    super.disconnectedCallback();
    window.remmoveEventListener("hashchange", this.displayHash.bind(this), false);
  }

  handleSlotChange({target}) {
    this.tabs = target.assignedNodes().filter(n => 'tabName' in n);
    this.showCurrent()
  }

  showCurrent() {
    this.links = [];
    for (let i = 0; i < this.tabs.length; i++) {
      let classes = "tab";
      if (i === this.current || this.tabs[i].slot === "sticky") {
        this.tabs[i].style.display = "block";
        window.location.hash = "#" + this.tabs[i].tabHashName();
        classes += " active"
      } else {
        this.tabs[i].style.display = "none"
      }
      if( this.tabs[i].tabIconFilename()) {
        this.links.push(html`<span class="${classes}" @click=${e => this.handleClick(i, e)}><img src="${this.tabs[i].tabIconFilename()}"/><span class="icon-aligned">${this.tabs[i].tabName()}</span></span> `)
      } else {
        this.links.push(html`<span class="${classes}" @click=${e => this.handleClick(i, e)}><span class="with-no-icon">${this.tabs[i].tabName()}</span></span> `)
      }
    }
    this.requestUpdate()
  }

  handleClick(i, e) {
    this.current = i;
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

customElements.define('dw-tabs', Tabs);

export class Tab extends LitElement {
  static get properties() {
    return {
      name: { type: String },
      icon: { type: String },
      hashname: { type: String }
    }
  }

  constructor() {
    super();
    this.name = "";
    this.hashname = "";
    this.icon = "";
  }

  set name(val) {
    let oldVal = this._name;
    this._name = val;
    this.hashname = val.toLowerCase().replace(/[^\w]+/,'-');
  }

  get name() {
    return this._name;
  }

  tabName() {
    return this.name;
  }

  tabIconFilename() {
    return this.icon;
  }

  tabHashName() {
    return this.hashname;
  }

  render() {
    return html`<div><slot></slot></div>`
  }
}

customElements.define('dw-tab', Tab);
