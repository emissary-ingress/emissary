import { LitElement, html, css } from '/edge_stack/vendor/lit-element.min.js'
import { useContext } from '/edge_stack/components/context.js';

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

  /**
   * A list of properties to track for "re-render".
   *
   * NOTE: this is not guaranteed to be a full list of properties
   *       but only a list of items that when changed will trigger
   *       a re-render (these will be debounced if a lot of updates
   *       happen at once).
   */
  static get properties() {
    return {
      current: { type: String },
      tabs: { type: Array }
    };
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

  handleHashChange() {
    for (let idx = 0; idx < this.tabs.length; idx++) {
      if(window.location.hash === ('#' + this.tabs[idx].tabHashName())) {
        this.current = this.tabs[idx].name;
        break;
      }
    }
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
  }

  handleClick(name) {
    this.current = name;
  }

  renderLinks() {
    let links = [];
    let currentTab = this.current;
    if (currentTab == '' && this.tabs.length > 0) {
      currentTab = this.tabs[0].name;
    }

    for (let idx = 0; idx < this.tabs.length; ++idx) {
      let classes = "tab";
      if (this.tabs[idx].name === currentTab || this.tabs[idx].slot === "sticky") {
        this.tabs[idx].style.display = "block";

        if( window.location.hash.length > 300 ) {
          /* if a long hash, then it might be a login cookie */
          if (useContext('auth-state', null)[0]) {
            /* logged in, so don't need to preserve the hash tag */
            window.location.hash = "#" + this.tabs[idx].tabHashName();
          } else {
            /* not logged in, so don't change the hash cookie */
          }
        } else {
          /* shorter hash, so it can't be a login cookie, so go ahead and change it */
          window.location.hash = "#" + this.tabs[idx].tabHashName();
        }

        classes += " active";
      } else {
        this.tabs[idx].style.display = "none";
      }

      if (this.tabs[idx].tabIconFilename()) {
        links.push(html`
          <span class="${classes}" @click=${() => this.handleClick(this.tabs[idx].name)}>
            <img src="${this.tabs[idx].tabIconFilename()}"/>
            <span class="icon-aligned">${this.tabs[idx].tabName()}</span>
          </span>
        `);
      } else {
        links.push(html`
          <span class="${classes}" @click=${() => this.handleClick(this.tabs[idx].name)}>
            <span class="with-no-icon">${this.tabs[idx].tabName()}</span>
          </span>
        `);
      }
    }

    return links;
  }

  render() {
    return html`
      <div class="tabs">
        ${this.renderLinks()}
      </div>
      <div class="main">
        <slot name="sticky"></slot>
        <slot @slotchange=${this.handleSlotChange}></slot>
      </div>
    `;
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
    return html`<slot></slot>`;
  }
}

customElements.define('dw-tab', Tab);
