import { LitElement, html, css } from '../vendor/lit-element.min.js'
import {useContext} from './context.js';

/**
 * Provides a small wrapper around named slots, to properly
 * render one of a series of "tabs".
 */
export class Tabs extends LitElement {
  static get styles() {
    return css`
/*MOREMORE moved to css file
.col_left {
  float: left;
  width: 190px;
  background-color: black;
  color: white;
  height: 90vh;
  font-size: 85%;
  font-weight: bold;
}

.col_right {
  margin-left: 190px;
}

.tab {
  display: block;
  padding: 1em;
  line-height: 1.3;
  cursor: pointer;
  text-decoration: none;
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

.tab-text {
  color: white;
}
*/
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

  renderLinks() {
    let links = [];
    let currentTab = this.current;
    if (currentTab == '' && this.tabs.length > 0) {
      currentTab = this.tabs[0].name;
    }

    for (let idx = 0; idx < this.tabs.length; ++idx) {
      let classes = "";
      if (this.tabs[idx].name === currentTab) {
        //MOREMORE this.tabs[idx].style.display = "block";

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

        classes += " selected";
      }

      if (this.tabs[idx].tabIconFilename()) {
        links.push(html`
          <a href="#${this.tabs[idx].tabHashName()}" class="${classes}">
            <div class="selected_stripe"></div>
            <div class="label">
              <div class="icon">
                <!-- MOREMORE potentially another div class=icon -->
                  <img src="${this.tabs[idx].tabIconFilename()}"/> <!-- MOREMORE replace with svg -->
                <!-- MOREMORE potentially end-div -->
              </div>
              <div class="name">${this.tabs[idx].tabName()}</div>
            </div>
          </a>
        `);
      } else {
        links.push(html`
          <a href="#${this.tabs[idx].tabHashName()}" class="${classes}">
            <div class="selected_stripe"></div>
            <div class="label">
              <div class="icon">
                <!-- MOREMORE potentially another div class=icon -->
                <!-- MOREMORE potentially end-div -->
              </div>
              <div class="name">${this.tabs[idx].tabName()}</div>
            </div>
          </a>
        `);
      }
    }

    return links;
  }

  render() {
    if (this.current === '') {
      let hash = window.location.hash.slice(1);
      this.tabs.forEach(tab => {
        if (hash === tab.tabHashName()) {
          this.current = tab.name;
        }
      })
    }

    let tabName = this.current;
    if (tabName === '') {
      tabName = this.tabs[0].name;
    }

    return html`
      <link rel="stylesheet" href="../styles/tabs.css">
      <div class="col_wrapper">
        <div class="col_left">
          <div class="logo"><img src="../images/ambassador-logo-white.svg"/></div>
          <navigation>
            ${this.renderLinks()}
          </navigation>
        </div>
        <div class="col_right">
          <div class="content">
            <slot name="${tabName}"></slot>
          </div>
        </div>
      </div>
    `;
  }

}

customElements.define('dw-tabs', Tabs);

export class Tab extends LitElement {
  static get properties() {
    return {
      _name: { type: String },
      icon: { type: String },
      hashname: { type: String }
    }
  }

  constructor() {
    super();

    this.icon = "";
    this._name = (this.getAttribute('slot') || '');
    this.updateHashname();
  }

  updateHashname() {
    this.hashname = this.name.toLowerCase().replace(/[^\w]+/, '-');
  }

  set slot(val) {
    this._name = val;
    this.updateHashname();
  }

  set name(val) {
    this._name = val;
    this.updateHashname();
  }

  get name() {
    return this._name;
  }

  tabName() {
    return this._name;
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
