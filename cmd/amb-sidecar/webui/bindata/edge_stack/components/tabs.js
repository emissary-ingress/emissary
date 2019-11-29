import { LitElement, html, css } from '/edge_stack/vendor/lit-element.min.js'
import {useContext} from '/edge_stack/components/context.js';

/**
 * Provides a small wrapper around named slots, to properly
 * render one of a series of "tabs". There is also a special
 * named slot called: "sticky" that will always be rendered.
 */
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

  handleHashChange() {
    for (let i = 0; i < this.tabs.length; i++) {
      if(window.location.hash === ('#' + this.tabs[i].tabHashName())) {
        this.current = this.tabs[i].name;
        break;
      }
    }
  }

  constructor() {
    super();

    this.current = '';
    this.tabs = [];
    Array.from(this.children).forEach(node => {
      if (node.localName == "dw-tab") {
        this.tabs.push(node);
      }
    });
  }

  connectedCallback() {
    super.connectedCallback();
    window.addEventListener("hashchange", this.handleHashChange.bind(this), false);
  }

  disconnectedCallback() {
    super.disconnectedCallback();
    window.remmoveEventListener("hashchange", this.handleHashChange.bind(this), false);
  }

  handleSlotChange() {
    // Create a variable, and set it instead of mutating
    // since mutations cause re-rendering.
    let newTabs = [];
    Array.from(this.children).forEach(node => {
      if (node.localName == "dw-tab") {
        newTabs.push(node);
      }
    });
    this.tabs = newTabs;

    this.showCurrent();
  }

  showCurrent() {
    this.links = [];
    if (this.current == '' && this.tabs.length > 0) {
      this.current = this.tabs[0].name;
    }
    for (let i = 0; i < this.tabs.length; i++) {
      let classes = "tab";
      if (this.tabs[i].name === this.current || this.tabs[i].slot === "sticky") {
        this.tabs[i].style.display = "block";
        if( window.location.hash.length > 300 ) {
          /* if a long hash, then it might be a login cookie */
          if (useContext('auth-state', null)[0]) {
            /* logged in, so don't need to preserve the hash tag */
            window.location.hash = "#" + this.tabs[i].tabHashName();
          } else {
            /* not logged in, so don't change the hash cookie */
          }
        } else {
          /* shorter hash, so it can't be a login cookie, so go ahead and change it */
          window.location.hash = "#" + this.tabs[i].tabHashName();
        }
        classes += " active"
      } else {
        this.tabs[i].style.display = "none"
      }
      if( this.tabs[i].tabIconFilename()) {
        this.links.push(html`<span class="${classes}" @click=${e => this.handleClick(this.tabs[i].name)}><img src="${this.tabs[i].tabIconFilename()}"/><span class="icon-aligned">${this.tabs[i].tabName()}</span></span> `)
      } else {
        this.links.push(html`<span class="${classes}" @click=${e => this.handleClick(this.tabs[i].name)}><span class="with-no-icon">${this.tabs[i].tabName()}</span></span> `)
      }
    }
    this.requestUpdate()
  }

  handleClick(name) {
    this.current = name;
    this.showCurrent();
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
